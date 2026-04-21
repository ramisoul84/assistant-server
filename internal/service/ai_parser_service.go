package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/ramisoul84/assistant-server/internal/domain"
)

type AIParser interface {
	Parse(ctx context.Context, userMessage string, userTime time.Time) (*domain.AIResponse, error)
	ParseCompletion(ctx context.Context, category, userMessage string, userTime time.Time) (*domain.AIResponse, error)
	ParseExercise(ctx context.Context, userMessage string) (*domain.AIResponse, error)
}

type aiParserService struct {
	client *openai.Client
	model  string
}

func NewAIParserService(client *openai.Client, model string) AIParser {
	return &aiParserService{client: client, model: model}
}

// ── Normal add intent ─────────────────────────────────────────────────────────

func normalPrompt(userTime time.Time) string {
	return fmt.Sprintf(`You are a personal assistant. The user's current time is: %s (UTC).
Detect if the user wants to add an appointment, expense, or gym session.
Respond ONLY with a raw JSON object — no markdown, no backticks.

Schema:
{
  "intent": "<add_appointment|add_expense|incomplete|unknown>",
  "incomplete": "<appointment|expense>",
  "reply": "<short friendly 1-sentence in user's language>",
  "appointment": { "title": "string", "datetime": "RFC3339", "notes": "string" },
  "expense": { "amount": 0.0, "currency": "EUR", "category": "string", "description": "string", "spent_at": "RFC3339" }
}

Rules:
- add_appointment: user mentions a meeting/appointment/reminder with enough info (title + time).
  If title OR datetime is missing → intent: incomplete, incomplete: "appointment"
- add_expense: user mentions buying/spending/paying with an amount.
  If amount OR item is missing → intent: incomplete, incomplete: "expense"
- incomplete reply: ask for the missing info in ONE friendly sentence with an example.
  appointment example reply: "Sure! What's the appointment for and when? e.g. 'Dentist next Monday 3pm'"
  expense example reply: "Got it! What did you buy and how much? e.g. 'Shoes 80€'"
- unknown: user is asking a question or wants to view data → reply: "Use /appointments, /expenses, or /gym to view your data."
- appointment.title: infer if not stated. Resolve relative times from user's current time. Default time 09:00.
- expense currency: detect from message, default EUR. spent_at: default to now.
`, userTime.UTC().Format(time.RFC3339))
}

// ── Completion (user is answering a follow-up question) ───────────────────────

func completionPrompt(category string, userTime time.Time) string {
	schema := ""
	switch category {
	case "appointment":
		schema = `"appointment": { "title": "string", "datetime": "RFC3339", "notes": "string" }`
	case "expense":
		schema = `"expense": { "amount": 0.0, "currency": "EUR", "category": "string", "description": "string", "spent_at": "RFC3339" }`
	}
	return fmt.Sprintf(`You are a personal assistant. The user's current time is: %s (UTC).
The user is providing information to save a %s.
Extract all fields from their message.
Respond ONLY with a raw JSON object.

Schema:
{
  "intent": "add_%s",
  "reply": "<short friendly confirmation>",
  %s
}

Rules:
- Extract as much as possible from the message.
- appointment.title: infer if not stated.
- Resolve relative times from user's current time. Default appointment time 09:00.
- expense currency: detect or default EUR. category: food/transport/health/shopping/entertainment/other.
`, userTime.UTC().Format(time.RFC3339), category, category, schema)
}

// ── Gym exercise line ─────────────────────────────────────────────────────────

func exercisePrompt() string {
	return `You are a gym tracker. Parse the user's exercise description.
Respond ONLY with a raw JSON object — no markdown.

Schema:
{
  "intent": "add_gym_exercise",
  "reply": "<short confirmation e.g. '✓ Bench press logged'>",
  "gym_exercise": {
    "name": "string",
    "sets": 0,
    "reps": 0,
    "weight_kg": 0.0,
    "notes": "string"
  }
}

Rules:
- name: always required, infer/normalize (e.g. "bp" → "Bench Press")
- sets/reps/weight: extract from formats like "4x10 80kg", "3 sets 8 reps 60kg", "5x5 100"
- weight_kg: 0 if not mentioned (bodyweight exercise)
- If the message is clearly not an exercise → intent: unknown, reply: "I didn't understand that exercise. Try: 'bench press 4x10 80kg'"
`
}

func (s *aiParserService) Parse(ctx context.Context, userMessage string, userTime time.Time) (*domain.AIResponse, error) {
	return s.call(ctx, normalPrompt(userTime), userMessage)
}

func (s *aiParserService) ParseCompletion(ctx context.Context, category, userMessage string, userTime time.Time) (*domain.AIResponse, error) {
	return s.call(ctx, completionPrompt(category, userTime), userMessage)
}

func (s *aiParserService) ParseExercise(ctx context.Context, userMessage string) (*domain.AIResponse, error) {
	return s.call(ctx, exercisePrompt(), userMessage)
}

func (s *aiParserService) call(ctx context.Context, system, user string) (*domain.AIResponse, error) {
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
		Temperature: 0.1,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("aiParser: groq call failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("aiParser: empty response")
	}
	var result domain.AIResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("aiParser: bad JSON %q: %w", resp.Choices[0].Message.Content, err)
	}
	return &result, nil
}
