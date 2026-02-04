.PHONY: build test lint clean run-gateway run-crm run-chat run-auth migrate-up migrate-down migrate-create proto

# Build all services
build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/crm ./cmd/crm
	go build -o bin/chat ./cmd/chat
	go build -o bin/auth ./cmd/auth

# Run individual services
run-gateway:
	go run ./cmd/gateway

run-crm:
	go run ./cmd/crm

run-chat:
	go run ./cmd/chat

run-auth:
	go run ./cmd/auth

# Test
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Database migrations (requires golang-migrate)
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# Proto generation (requires protoc, protoc-gen-go, protoc-gen-go-grpc)
proto:
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/auth/v1/auth.proto
