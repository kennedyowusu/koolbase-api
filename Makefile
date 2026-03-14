.PHONY: run build migrate-up migrate-down docker-up docker-down

# Run locally
run:
	go run ./cmd/api

# Build binary
build:
	CGO_ENABLED=0 go build -o bin/hatchway-api ./cmd/api

# Run migrations (requires golang-migrate CLI)
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down

# Docker
docker-up:
	docker-compose up --build

docker-down:
	docker-compose down

# Tidy modules
tidy:
	go mod tidy
