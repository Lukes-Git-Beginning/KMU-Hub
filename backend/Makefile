.PHONY: build test lint clean run-gateway run-crm run-chat run-auth run-notification run-work migrate-up migrate-down migrate-create proto docker-build docker-up docker-down docker-logs docker-clean e2e-test

# Build all services
build:
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/crm ./cmd/crm
	go build -o bin/chat ./cmd/chat
	go build -o bin/auth ./cmd/auth
	go build -o bin/notification ./cmd/notification
	go build -o bin/work ./cmd/work

# Run individual services
run-gateway:
	go run ./cmd/gateway

run-crm:
	go run ./cmd/crm

run-chat:
	go run ./cmd/chat

run-auth:
	go run ./cmd/auth

run-notification:
	go run ./cmd/notification

run-work:
	go run ./cmd/work

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
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/crm/v1/crm.proto
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/chat/v1/chat.proto
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/notification/v1/notification.proto
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/work/v1/work.proto
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/calendar/v1/calendar.proto
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/video/v1/video.proto
	protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub \
		--go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub \
		proto/security/v1/security.proto

# Docker operations
docker-build:
	cd ../deploy/docker && docker-compose build

docker-up:
	cd ../deploy/docker && docker-compose up -d

docker-down:
	cd ../deploy/docker && docker-compose down

docker-logs:
	cd ../deploy/docker && docker-compose logs -f

docker-clean:
	cd ../deploy/docker && docker-compose down -v

# E2E tests (requires running Docker stack)
e2e-test:
	go test -tags=e2e ./test/e2e/... -v

# Health checks
health-gateway:
	curl -s http://localhost:8080/health

health-auth:
	curl -s http://localhost:9091/health

# Metrics
metrics-gateway:
	curl -s http://localhost:9090/metrics

metrics-auth:
	curl -s http://localhost:9091/metrics
