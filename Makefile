.PHONY: build test lint clean run deps check dev fmt

build:
	go build -o bin/agent .

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

clean:
	rm -rf bin/

run: build
	./bin/agent

deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

check: fmt lint test

dev: clean fmt lint test build
