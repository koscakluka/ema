package groq

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	defaultModel  = "llama-3.3-70b-versatile"
	defaultPrompt = "You are a helpful assistant, keep the conversation going and answer any questions to the best of your ability. Reply concisely and clearly unless asked to expand on something."

	url = "https://api.groq.com/openai/v1/chat/completions"

	endMessage  = "[DONE]"
	chunkPrefix = "data:"
)

type Message struct {
	Role    messageRole `json:"role"`
	Content string      `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ResponseBody struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type Client struct {
	apiKey   string
	messages []Message // TODO: make sure mesages are kept externally and the client is only responsible for communication
}

func NewClient() *Client {
	apiKey, ok := os.LookupEnv("GROQ_API_KEY")
	if !ok {
		return nil
	}

	return &Client{
		apiKey: apiKey,
		messages: []Message{
			{
				Role:    roleSystem,
				Content: defaultPrompt,
			},
		},
	}
}

func (c *Client) Prompt(ctx context.Context, message string, streamFunc func(string)) (string, error) {
	c.messages = append(c.messages, Message{
		Role:    roleUser,
		Content: message,
	})
	reqBody := RequestBody{
		Model:    defaultModel,
		Messages: c.messages,
		Stream:   true,
	}

	requestBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Non-OK HTTP status:", resp.Status)
	}

	var response strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		chunk := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), chunkPrefix))

		if len(chunk) == 0 {
			continue
		}

		if chunk == endMessage {
			break
		}

		var responseBody ResponseBody
		err := json.Unmarshal([]byte(chunk), &responseBody)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			continue
		}
		content := responseBody.Choices[0].Delta.Content
		response.WriteString(content)
		if streamFunc != nil {
			streamFunc(content)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading streamed response:", err)
	}

	c.messages = append(c.messages, Message{
		Role:    roleAssistant,
		Content: response.String(),
	})

	return response.String(), nil
}

type messageRole string

const (
	roleSystem    messageRole = "system"
	roleUser      messageRole = "user"
	roleAssistant messageRole = "assistant"
)
