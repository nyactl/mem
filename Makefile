BIN     := mem
INSTALL := ~/.local/bin/$(BIN)

.PHONY: build install vet test snapshot

build:
	go build -o $(BIN) ./cmd/mem-cli

install:
	go build -o $(INSTALL) ./cmd/mem-cli

vet:
	go vet ./...

test:
	go test ./...

snapshot:
	goreleaser release --snapshot --clean --skip=publish
