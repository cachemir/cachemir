.PHONY: build test clean server client example deps

# Build targets
build: server client

server:
	go build -o bin/cachemir-server cmd/server/main.go

client:
	go build -o bin/cachemir-client-example cmd/client-example/main.go

# Development targets
deps:
	go mod tidy
	go mod download

test:
	go test -v ./...

test-race:
	go test -race -v ./...

benchmark:
	go test -bench=. -benchmem ./...

# Run targets
run-server:
	go run cmd/server/main.go

run-client:
	go run cmd/client-example/main.go

run-cluster:
	go run cmd/server/main.go -port 8080 &
	go run cmd/server/main.go -port 8081 &
	go run cmd/server/main.go -port 8082 &
	@echo "Started 3-node cluster on ports 8080, 8081, 8082"

# Utility targets
clean:
	rm -rf bin/
	go clean

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

# Docker targets
docker-build:
	docker build -t cachemir:latest .

docker-run:
	docker run -p 8080:8080 cachemir:latest

# Documentation
docs:
	godoc -http=:6060

# All quality checks
check: fmt vet test

# Install tools
install-tools:
	go install golang.org/x/tools/cmd/godoc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

help:
	@echo "Available targets:"
	@echo "  build       - Build server and client binaries"
	@echo "  server      - Build server binary"
	@echo "  client      - Build client example binary"
	@echo "  test        - Run tests"
	@echo "  test-race   - Run tests with race detection"
	@echo "  benchmark   - Run benchmarks"
	@echo "  run-server  - Run server (port 8080)"
	@echo "  run-client  - Run client example"
	@echo "  run-cluster - Start 3-node cluster"
	@echo "  clean       - Clean build artifacts"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  lint        - Run golangci-lint"
	@echo "  check       - Run all quality checks"
	@echo "  docs        - Start godoc server"
	@echo "  help        - Show this help"
