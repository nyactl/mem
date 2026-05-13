BIN := ref-cli

.PHONY: build install test

build:
	go build -o $(BIN) ./cmd/ref-cli

install:
	go install ./cmd/ref-cli

test:
	go test ./...
