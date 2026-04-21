# ── Build stage ───────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/main.go

# ── Final stage (tiny image) ──────────────────────────────
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8000

CMD ["./server"]