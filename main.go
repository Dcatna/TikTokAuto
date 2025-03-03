package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

const apiURL = "https://api-inference.huggingface.co/models/tiiuae/falcon-7b-instruct"

var client = &http.Client{}

func generateStory(prompt string) string {
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

func generateVoiceOver(story string, outPut string) {
	apikey := os.Getenv("HUGGINGFACE_API_KEY")
	url := "https://api-inference.huggingface.co/models/facebook/mms-tts-eng"
	request := map[string]string{"inputs": story}
	jsonData, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))

	req.Header.Set("Authorization", "Bearer "+apikey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	fmt.Println("Response Status:", resp.Status)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	file, err := os.Create(outPut)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	fmt.Println("Voice over saved as:", outPut)

}

func main() {
	story := generateStory("Write a scary story about a haunted house with 500 words")
	fmt.Println(story)
	generateVoiceOver(story, "testvoiceover.wav")
}
