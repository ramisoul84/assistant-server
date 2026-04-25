package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ramisoul84/assistant-server/internal/domain"
	openai "github.com/sashabaranov/go-openai"
)

type AIService interface {
	Parse(ctx context.Context, text string, now time.Time) (*domain.AIResult, error)
}

type aiService struct {
	client *openai.Client
	model  string
}

func NewAIService(client *openai.Client, model string) AIService {
	return &aiService{client: client, model: model}
}

func buildPrompt(now time.Time) string {
	return fmt.Sprintf(`You are a smart personal assistant parser. The user's current date and time is: %s (UTC).

Your job: read the user's message and extract structured data. Always respond with ONLY a raw JSON object — no markdown, no explanation.

Classify into exactly one intent:
- "save_expense"  → user spent money, bought something, paid for something
- "save_income"   → user received money, got paid, salary, earnings
- "save_note"     → everything else: reminders, appointments, tasks, ideas, thoughts

JSON schema:
{
  "intent": "save_expense | save_income | save_note | unknown",
  "reply": "friendly 1-sentence confirmation in the same language as the user",
  "expense": {
    "amount": 0.0,
    "currency": "EUR",
    "category": "food | transport | health | shopping | bills | entertainment | other",
    "description": "what was bought or paid for",
    "happened_at": "RFC3339"
  },
  "income": {
    "amount": 0.0,
    "currency": "EUR",
    "category": "salary | freelance | business | gift | investment | other",
    "description": "source or description of income",
    "happened_at": "RFC3339"
  },
  "note": {
    "content": "the full note or reminder text",
    "datetime": "RFC3339 or null",
    "tags": ["optional", "tags"]
  }
}

Rules:
- Only include the object matching the intent (expense, income, or note).
- ALWAYS classify even if info is vague. Never ask for more info — infer everything.
  - "bought something" → expense, category=other, description="Purchase"
  - "got paid" → income, category=salary
  - "remember to call mom" → note, no datetime
  - "doctor appointment friday 3pm" → note WITH datetime resolved from user's current time
- Amounts: extract numbers, detect currency symbol (€=$=£ etc.), default EUR.
- Dates: resolve ALL relative expressions ("tomorrow", "next Monday", "in 2 hours") using user's current time.
  If time not given for appointments, default to 09:00.
- Note datetime: set ONLY when user specifies a time/date. Leave null for plain notes.
- Tags: infer 1-3 relevant tags for notes (e.g. ["health"], ["work"], ["family"]).
- reply: warm, concise, confirm exactly what was saved. Match user language.
- unknown: only if truly impossible to understand. Extremely rare.
`, now.UTC().Format(time.RFC3339))
}

func (s *aiService) Parse(ctx context.Context, text string, now time.Time) (*domain.AIResult, error) {
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: buildPrompt(now)},
			{Role: openai.ChatMessageRoleUser, Content: text},
		},
		Temperature: 0.1,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("aiService.Parse: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("aiService.Parse: empty response")
	}

	var result domain.AIResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("aiService.Parse: bad JSON: %w", err)
	}
	return &result, nil
}
