build:
	go build -o bin/rami-server ./cmd/...

run:
	APP_ENV=development go run ./cmd/main.go

tidy:
	go mod tidy