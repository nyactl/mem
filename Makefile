BIN := mem

.PHONY: build install test \
        sandbox-serve sandbox-new sandbox-snap sandbox-day \
        sandbox-pull sandbox-push sandbox-ls

build:
	go build -o $(BIN) ./cmd/mem

install:
	go install ./cmd/mem

test:
	go test ./...

# ── sandbox targets ───────────────────────────────────────────────────────
# Two separate dirs mirror real deployment:
#   .sandbox/server/ — what mem serve reads/writes (git-backed)
#   .sandbox/local/  — local client notes, syncs to/from the server
#
# Typical flow:
#   make sandbox-serve          (terminal 1 — leave running)
#   make sandbox-snap ARGS="hello world #test"
#   make sandbox-push
#   make sandbox-pull

sandbox-serve: build
	MEM_CONFIG=.sandbox/server/config.json ./$(BIN) serve

sandbox-new: build
	MEM_CONFIG=.sandbox/local/config.json ./$(BIN) new $(ARGS)

sandbox-snap: build
	MEM_CONFIG=.sandbox/local/config.json ./$(BIN) snap $(ARGS)

sandbox-day: build
	MEM_CONFIG=.sandbox/local/config.json ./$(BIN) day $(ARGS)

sandbox-ls: build
	MEM_CONFIG=.sandbox/local/config.json ./$(BIN) ls $(ARGS)

sandbox-pull: build
	MEM_CONFIG=.sandbox/local/config.json ./$(BIN) sync pull

sandbox-push: build
	MEM_CONFIG=.sandbox/local/config.json ./$(BIN) sync push
