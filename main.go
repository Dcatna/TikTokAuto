package main

import (
	"bytes"

	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-audio/wav"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

var client = &http.Client{}

func generateStory(prompt string) string {
	apiURL := "https://api-inference.huggingface.co/models/tiiuae/falcon-7b-instruct"
	apiKey := os.Getenv("HUGGINGFACE_API_KEY")

	request := map[string]string{"inputs": prompt}
	jsonData, _ := json.Marshal(request)

	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error sending request:", err)
		return "Error communicating with AI."
	}
	defer resp.Body.Close()

	var result []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result) > 0 {
		if generatedText, ok := result[0]["generated_text"].(string); ok {
			cleanText := strings.TrimPrefix(generatedText, prompt)
			return strings.TrimSpace(cleanText)
		}
	}

	return "No response from AI."
}

func generateVoiceOver(story string, outputFile string) {
	cmd := exec.Command("python", "bark_tts.py", story)
	output, err := cmd.CombinedOutput()

	fmt.Println("python output:", string(output))

	if err != nil {
		fmt.Println("error running Bark TTS:", err)
	} else {
		fmt.Println("voiceover saved as", outputFile)
	}

}

func getWavLength(fileName string) (float64, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return 0, fmt.Errorf("Invalid WAV file")
	}

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return 0, err
	}

	duration := float64(len(buf.Data)) / float64(buf.Format.SampleRate)
	return duration, nil
}

func trimMP4(inputVideo string, outputVideo string, duration float64) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inputVideo, "-t", strconv.FormatFloat(duration, 'f', 2, 64), "-c", "copy", outputVideo)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func mergeAudioVideo(videoFile, audioFile, outputFile string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", videoFile, "-i", audioFile,
		"-c:v", "copy", "-map", "0:v:0", "-map", "1:a:0", "-shortest", outputFile)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createSRT(text string, audioDuration float64, outputSRT string) error {
	words := strings.Fields(text) // Split the story into words
	wordCount := len(words)

	// Calculate duration per word (rough estimation)
	wordDuration := audioDuration / float64(wordCount)

	file, err := os.Create(outputSRT)
	if err != nil {
		return err
	}
	defer file.Close()

	// Generate subtitles
	var startTime, endTime float64
	for i, word := range words {
		startTime = endTime
		endTime = startTime + wordDuration

		// Convert time to SRT format (hh:mm:ss,ms)
		startTimeStr := formatTimestamp(startTime)
		endTimeStr := formatTimestamp(endTime)

		// Write to the SRT file
		fmt.Fprintf(file, "%d\n%s --> %s\n%s\n\n", i+1, startTimeStr, endTimeStr, word)
	}

	fmt.Println("subtitle file created:", outputSRT)
	return nil
}
func generateSRTFromSegments(segments []interface{}, outputSRT string) error {
	file, err := os.Create(outputSRT)
	if err != nil {
		fmt.Println("❌ Error creating SRT file:", err)
		return err
	}
	defer file.Close()

	for i, segment := range segments {
		seg := segment.(map[string]interface{})
		start := seg["start"].(float64)
		end := seg["end"].(float64)
		text := seg["text"].(string)

		fmt.Fprintf(file, "%d\n%02d:%02d:%02d,%03d --> %02d:%02d:%02d,%03d\n%s\n\n",
			i+1,
			int(start/3600), int(start/60)%60, int(start)%60, int((start*1000))%1000,
			int(end/3600), int(end/60)%60, int(end)%60, int((end*1000))%1000,
			text,
		)
	}

	fmt.Println("✅ Subtitles generated successfully:", outputSRT)
	return nil
}

func generateSRTFromText(text, outputSRT string, audioDuration float64) error {
	words := strings.Fields(text) // Split into words
	numWords := len(words)

	if numWords == 0 {
		fmt.Println("❌ No words to process for subtitles")
		return fmt.Errorf("empty subtitle text")
	}

	wordsPerSubtitle := 5 // Adjustable for readability
	subtitles := []string{}

	// Calculate total subtitles needed
	numSubtitles := numWords / wordsPerSubtitle
	if numWords%wordsPerSubtitle != 0 {
		numSubtitles++
	}

	// Calculate dynamic time per subtitle
	durationPerSubtitle := audioDuration / float64(numSubtitles)
	startTime := 0.0
	subtitleIndex := 1

	file, err := os.Create(outputSRT)
	if err != nil {
		fmt.Println("❌ Error creating SRT file:", err)
		return err
	}
	defer file.Close()

	for i := 0; i < len(words); i += wordsPerSubtitle {
		endTime := startTime + durationPerSubtitle
		chunk := strings.Join(words[i:min(i+wordsPerSubtitle, len(words))], " ")

		// Store the formatted SRT entry
		subtitles = append(subtitles, fmt.Sprintf(
			"%d\n%s --> %s\n%s\n",
			subtitleIndex,
			formatTime(startTime),
			formatTime(endTime),
			chunk,
		))

		// Update time and index
		startTime = endTime
		subtitleIndex++
	}

	// Write to SRT file
	for _, subtitle := range subtitles {
		file.WriteString(subtitle + "\n")
	}

	fmt.Println("✅ Subtitles saved to", outputSRT)
	return nil
}

// Utility function to format time for SRT
func formatTime(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	sec := int(seconds) % 60
	millisec := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, sec, millisec)
}

// Min function to prevent out-of-bounds issues
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateSubtitles(audioFile, outputSRT string, duration float64) error {
	apiURL := "https://api-inference.huggingface.co/models/openai/whisper-large-v3"
	apiKey := os.Getenv("HUGGINGFACE_API_KEY")

	// Read the audio file
	audioData, err := os.ReadFile(audioFile)
	if err != nil {
		fmt.Println("❌ Error reading audio file:", err)
		return err
	}

	// Create API request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(audioData))
	if err != nil {
		fmt.Println("❌ Error creating request:", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "audio/wav")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("❌ Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Println("🔍 API Response:", string(body)) // Debugging

	// Handle response with "segments"
	if segments, ok := result["segments"].([]interface{}); ok {
		return generateSRTFromSegments(segments, outputSRT)
	}

	// Handle response with "text" field instead
	if text, ok := result["text"].(string); ok {
		return generateSRTFromText(text, outputSRT, duration)
	}

	fmt.Println("❌ Error: No subtitles found in API response")
	return fmt.Errorf("no subtitles generated")
}

func formatTimestamp(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secondsPart := int(duration.Seconds()) % 60
	milliseconds := int(duration.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secondsPart, milliseconds)
}

func addSubtitles(videoFile, subtitleFile, outputFile string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", videoFile, "-vf", fmt.Sprintf("subtitles=%s", subtitleFile), outputFile)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func resizeVideo(input string, output string) {
	cmd := exec.Command("ffmpeg", "-y", "-i", input, "-vf", "scale=1080:1920,setsar=1:1", "-c:a", "copy", output)
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error resizing", err)
	} else {
		fmt.Println("video was resized")
	}

}

func main() {

	story := generateStory("Write a funny and engaging story in the style of a viral Reddit post. Use a relatable real-life situation with an unexpected twist. Keep it engaging and dramatic with a mix of humor and suspense. The story should start with a strong hook and have a surprising, funny ending.")
	fmt.Println(story)

	generateVoiceOver(story, "voiceover.wav")

	duration, err := getWavLength("voiceover.wav")
	if err != nil {
		fmt.Println("Error getting WAV duration:", err)
	}
	fmt.Println("WAV Duration:", math.Ceil(duration), "seconds")

	err = trimMP4("MC Parkour.mp4", "trimmed.mp4", duration)
	if err != nil {
		fmt.Println("Error trimming video:", err)
		return
	}

	err = mergeAudioVideo("trimmed.mp4", "voiceover.wav", "final_output.mp4")
	if err != nil {
		fmt.Println("Error merging video and audio:", err)
	} else {
		fmt.Println("output saved as final_output.mp4")
	}

	err = generateSubtitles("voiceover.wav", "subtitles.srt", duration)
	if err != nil {
		fmt.Println("Error generating subtitles:", err)
		return
	}

	resizeVideo("final_output.mp4", "final_output_resized.mp4")

	err = addSubtitles("final_output_resized.mp4", "subtitles.srt", "final_output_subtitled.mp4")
	if err != nil {
		fmt.Println("Error adding subtitles to video:", err)
	} else {
		fmt.Println("output saved as output_subtitled.mp4")
	}

}
