BIN := mem-cli

.PHONY: build install test

build:
	go build -o $(BIN) ./cmd/mem-cli

install:
	go install ./cmd/mem-cli

test:
	go test ./...
