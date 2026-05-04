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
	Parse(ctx context.Context, text string, localNow time.Time) (*domain.AIResult, error)
	// DetectTimezone extracts an IANA timezone from a free-text message.
	// Returns empty string if no timezone can be detected — always non-blocking.
	DetectTimezone(ctx context.Context, text string) string
}

type aiService struct {
	client *openai.Client
	model  string
}

func NewAIService(client *openai.Client, model string) AIService {
	return &aiService{client: client, model: model}
}

// ── Parse ─────────────────────────────────────────────────────────────────────

func buildPrompt(localNow time.Time) string {
	return fmt.Sprintf(`You are a smart personal assistant parser.
The user's current local date and time (with timezone offset) is: %s

Your job: read the user's message and extract structured data.
Always respond with ONLY a raw JSON object — no markdown, no explanation.

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
    "happened_at": "RFC3339 with timezone offset"
  },
  "income": {
    "amount": 0.0,
    "currency": "EUR",
    "category": "salary | freelance | business | gift | investment | other",
    "description": "source or description of income",
    "happened_at": "RFC3339 with timezone offset"
  },
  "note": {
    "content": "the full note or reminder text",
    "datetime": "RFC3339 with timezone offset, or null",
    "tags": ["optional", "tags"]
  }
}

Rules:
- Only include the object matching the intent (expense, income, or note).
- ALWAYS classify even if info is vague. Never ask for more info — infer everything.
- Amounts: extract numbers, detect currency symbol (€=$=£ etc.), default EUR.
- Dates: resolve ALL relative expressions ("tomorrow", "next Monday", "in 2 hours", "at 16")
  using the user's LOCAL time shown above. Output times MUST include the timezone offset.
  Example: user at +03:00 says "tomorrow at 16" → "2026-04-27T16:00:00+03:00"
  If time not given for appointments, default to 09:00 in the user's local time.
- happened_at for expenses/income: if not specified, use the user's current local time with offset.
- Note datetime: set ONLY when user specifies a time/date. Leave null for plain notes.
- Tags: infer 1-3 relevant tags for notes (e.g. ["health"], ["work"], ["family"]).
- reply: warm, concise, confirm exactly what was saved. Match user language.
- unknown: only if truly impossible to understand. Extremely rare.
`, localNow.Format(time.RFC3339))
}

func (s *aiService) Parse(ctx context.Context, text string, localNow time.Time) (*domain.AIResult, error) {
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: buildPrompt(localNow)},
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

// ── DetectTimezone ────────────────────────────────────────────────────────────

const tzPrompt = `You are a timezone detection assistant.
The user sent a message that may contain clues about their timezone:
a city name, country, time reference, or language.

Respond with ONLY a raw JSON object:
{ "timezone": "IANA_timezone_or_empty_string" }

Rules:
- Return a valid IANA timezone name (e.g. "Europe/Moscow", "America/New_York", "Asia/Dubai").
- If the message contains a clear city, country or region → return its primary IANA timezone.
- If the message is in a language strongly associated with one timezone region, use that.
- If there is not enough information to guess confidently → return { "timezone": "" }.
- Never guess blindly. Empty string is correct when unsure.
- Never return "UTC" as a guess — only return UTC if the user explicitly said UTC.

Examples:
  "dentist in Moscow tomorrow" → { "timezone": "Europe/Moscow" }
  "meeting in Dubai at 3pm"    → { "timezone": "Asia/Dubai" }
  "купить молоко"              → { "timezone": "Europe/Moscow" }
  "buy groceries"              → { "timezone": "" }
  "Zahnarzt morgen"            → { "timezone": "Europe/Berlin" }`

func (s *aiService) DetectTimezone(ctx context.Context, text string) string {
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: tzPrompt},
			{Role: openai.ChatMessageRoleUser, Content: text},
		},
		Temperature: 0,
		MaxTokens:   40,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil || len(resp.Choices) == 0 {
		return ""
	}
	var result struct {
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return ""
	}
	// Validate before trusting
	if result.Timezone == "" {
		return ""
	}
	if _, err := time.LoadLocation(result.Timezone); err != nil {
		return ""
	}
	return result.Timezone
}
