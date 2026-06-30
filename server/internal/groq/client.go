package groq

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatCompletionRequest struct {
	Messages       []Message       `json:"messages"`
	Model          string          `json:"model"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *resty.Client
}

func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: resty.New(),
	}
}

// CreateChatCompletion sends a request to Groq's chat completion endpoint
func (c *Client) CreateChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	return c.createChatCompletionGeneric(req)
}

// createChatCompletionGeneric handles both regular and vision requests
func (c *Client) createChatCompletionGeneric(req interface{}) (*ChatCompletionResponse, error) {
	if c.apiKey == "" {
		return nil, errors.New("groq api key is not configured")
	}

	var completionResponse ChatCompletionResponse

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)

	resp, err := c.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.apiKey)).
		SetBody(req).
		SetResult(&completionResponse).
		Post(url)

	if err != nil {
		return nil, fmt.Errorf("failed to send request to groq: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("groq api returned error code %d: %s", resp.StatusCode(), resp.String())
	}

	// Publish token usage event to active CLI subscribers
	GlobalBroadcaster.Publish(TokenUsageEvent{
		Model:            completionResponse.Model,
		PromptTokens:     completionResponse.Usage.PromptTokens,
		CompletionTokens: completionResponse.Usage.CompletionTokens,
		TotalTokens:      completionResponse.Usage.TotalTokens,
	})

	return &completionResponse, nil
}

