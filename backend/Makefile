.PHONY: dev test lint fmt

fmt:
	gofmt -l -w .

dev:
	go run ./cmd/server

test:
	go test ./... -race -count=1

lint:
	go vet ./...
