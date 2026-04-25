.PHONY: run build migrate

run:
	go run ./cmd/main.go

build:
	go build -o bin/assistant ./cmd/main.go

migrate:
	./scripts/migrate_up.sh
