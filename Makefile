BIN := mem-cli

.PHONY: build install test sandbox-serve sandbox-new sandbox-pull sandbox-push

build:
	go build -o $(BIN) ./cmd/mem-cli

install:
	go install ./cmd/mem-cli

test:
	go test ./...

# ── sandbox targets ───────────────────────────────────────────────────────
# Uses .sandbox/config.json; notes land in .sandbox/notes/ (git-tracked there,
# gitignored from this repo). Run mem commands against the sandbox with:
#   MEM_CONFIG=.sandbox/config.json ./mem-cli <command>

sandbox-serve: build
	MEM_CONFIG=.sandbox/config.json ./$(BIN) serve

sandbox-new: build
	MEM_CONFIG=.sandbox/config.json ./$(BIN) new $(ARGS)

sandbox-pull: build
	MEM_CONFIG=.sandbox/config.json ./$(BIN) sync pull

sandbox-push: build
	MEM_CONFIG=.sandbox/config.json ./$(BIN) sync push
