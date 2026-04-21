package ai

import (
	openai "github.com/sashabaranov/go-openai"
	"github.com/ramisoul84/assistant-server/internal/config"
)

// NewGroqClient builds an OpenAI-compatible client pointed at Groq's API.
// Groq is a 100% OpenAI-compatible endpoint — no URL hacks needed,
// just swap the base URL and API key.
func NewGroqClient(cfg config.AIConfig) *openai.Client {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = "https://api.groq.com/openai/v1"
	return openai.NewClientWithConfig(clientConfig)
}
