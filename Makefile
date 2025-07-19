.PHONY: build test lint clean run deps check dev

build:
	go build -o bin/agent .

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

run: build
	./bin/agent

deps:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

check: lint test

dev: clean lint test build
