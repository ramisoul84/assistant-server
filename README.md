# Assistant Server

A Go backend that powers a personal AI assistant via **Telegram bot**, with a **REST API** for an Angular dashboard.

## Architecture

```
Telegram User
     │  (natural language)
     ▼
internal/bot/bot.go            ← delivery: receives raw messages
     │
     ▼
internal/service/ai_parser_service.go  ← calls OpenAI, returns ParsedIntent
     │
     ├─► internal/service/appointment_service.go
     ├─► internal/service/expense_service.go
     └─► internal/service/gym_service.go
              │
              ▼
     internal/repository/*     ← writes to PostgreSQL
              │
              ▼
     PostgreSQL DB


Angular Site
     │  (HTTP GET)
     ▼
internal/server/http/          ← Fiber REST API
     └── /api/v1/users/:id/appointments
     └── /api/v1/users/:id/expenses
     └── /api/v1/users/:id/gym-sessions
```

## Quick Start

### 1. Prerequisites
- Go 1.23+
- PostgreSQL 14+
- A Telegram bot token ([@BotFather](https://t.me/BotFather))
- An OpenAI API key

### 2. Configure environment
```bash
cp .env.dev.example .env.dev
# Fill in BOT_TOKEN, OPENAI_API_KEY, DB_USER, DB_PASSWORD, DB_NAME
```

### 3. Run migrations
```bash
./scripts/migrate_up.sh
```

### 4. Run the server
```bash
go run ./cmd/main.go
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/users/:id/appointments` | List appointments |
| GET | `/api/v1/users/:id/expenses` | List expenses |
| GET | `/api/v1/users/:id/gym-sessions` | List gym sessions |

## Telegram Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Welcome message |
| `/help` | Show usage examples |

Or just type naturally:
- _"Add dentist appointment tomorrow at 3pm"_
- _"Spent 45€ on groceries"_
- _"Bench press 4 sets 8 reps 90kg"_

## Adding New Domains

1. Add struct to `internal/domain/`
2. Add intent constants to `domain/intent.go` + `AIResponse` field
3. Create repo in `internal/repository/`
4. Create service in `internal/service/`
5. Create bot handler in `internal/bot/handlers/`
6. Add case to `bot.go` dispatcher switch
7. Add HTTP handler + route in `internal/server/http/`
8. Wire everything in `cmd/main.go`
