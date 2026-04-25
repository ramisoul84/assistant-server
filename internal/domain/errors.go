package domain

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidInput = errors.New("invalid input")
	ErrUserNotFound = errors.New("user not found")
)

type ctxKey string
const RequestIDKey ctxKey = "request_id"
