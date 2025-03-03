package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func generateStory(prompt string) string {
    apiKey := os.Getenv("HUGGINGFACE_API_KEY")

    request := map[string]string{"inputs": prompt}
    jsonData, _ := json.Marshal(request)

    req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
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
            // Remove the original prompt if it appears at the start of the generated text
            cleanText := strings.TrimPrefix(generatedText, prompt)
            return strings.TrimSpace(cleanText)
        }
    }

    return "No response from AI."
}

func main() {
	story := generateStory("Write a scary story about a haunted house with 500 words")
	fmt.Println(story)
}
