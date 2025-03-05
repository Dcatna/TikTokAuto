package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error running Bark TTS:", err)
	} else {
		fmt.Println("Voiceover saved as", outputFile)
	}

}

func main() {
	story := generateStory("Write a scary story about a haunted house with 500 words")
	fmt.Println(story)
	generateVoiceOver(story, "testvoiceover.wav")
}
