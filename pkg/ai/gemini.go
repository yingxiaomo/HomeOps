package ai

import (
	"context"
	"errors"
	"fmt"
	"go_bot/config"
	"log"
	"sync"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	apiKeys         []string
	models          []string
	currentKeyIndex int
	mu              sync.Mutex
}

func NewGeminiClient() *GeminiClient {
	return &GeminiClient{
		apiKeys: config.AppConfig.GeminiAPIKeys,
		models: []string{
			"gemini-3-pro-preview",
			"gemini-2.5-pro",
			"gemini-3-flash-preview",
			"gemini-2.5-flash",
			"gemini-2.0-flash",
		},
		currentKeyIndex: 0,
	}
}

func (c *GeminiClient) rotateKey() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.apiKeys) <= 1 {
		return false
	}
	c.currentKeyIndex = (c.currentKeyIndex + 1) % len(c.apiKeys)
	log.Printf("Rotated to API Key index: %d", c.currentKeyIndex)
	return true
}

func (c *GeminiClient) getCurrentKey() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.apiKeys) == 0 {
		return ""
	}
	return c.apiKeys[c.currentKeyIndex]
}

func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string, imageParts []byte) (string, error) {
	if len(c.apiKeys) == 0 {
		return "", errors.New("no API keys configured")
	}

	var lastErr error

	for _, modelName := range c.models {
		startKeyIndex := c.currentKeyIndex
		
		for {
			key := c.getCurrentKey()
			log.Printf("Attempting model: %s with key index: %d", modelName, c.currentKeyIndex)

			client, err := genai.NewClient(ctx, option.WithAPIKey(key))
			if err != nil {
				lastErr = err
				if !c.rotateKey() || c.currentKeyIndex == startKeyIndex {
					break // Try next model
				}
				continue
			}
			defer client.Close()

			model := client.GenerativeModel(modelName)
			
			var resp *genai.GenerateContentResponse
			var genErr error

			if len(imageParts) > 0 {
				resp, genErr = model.GenerateContent(ctx, genai.Text(prompt), genai.ImageData("png", imageParts))
			} else {
				resp, genErr = model.GenerateContent(ctx, genai.Text(prompt))
			}

			if genErr == nil {
				return printResponse(resp), nil
			}

			lastErr = genErr
			log.Printf("Error with model %s: %v", modelName, genErr)

			// Rotate key and retry same model
			if !c.rotateKey() || c.currentKeyIndex == startKeyIndex {
				break // All keys failed for this model, try next model
			}
		}
	}

	return "", fmt.Errorf("all models failed, last error: %v", lastErr)
}

func printResponse(resp *genai.GenerateContentResponse) string {
	var result string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					result += string(txt)
				}
			}
		}
	}
	return result
}
