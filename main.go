package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	fmt.Println("✅ Subtitle file created:", outputSRT)
	return nil
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


func main() {
	story := generateStory("Write a scary story about a haunted house with 500 words")
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

	err = createSRT(story, duration, "subtitles.srt")
	if err != nil {
        fmt.Println("Error creating SRT file:", err)
    }

	err = addSubtitles("final_output.mp4", "subtitles.srt", "final_output_subtitled.mp4")
	if err != nil {
        fmt.Println("Error adding subtitles to video:", err)
    } else {
        fmt.Println("output saved as final_output_subtitled.mp4")
    }

}
