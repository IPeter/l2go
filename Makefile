GO       ?= go
TOOL     := $(GO) tool -modfile=tools/go.mod
FUZZTIME ?= 60s
MONGO_URI ?= mongodb://127.0.0.1:27017

.PHONY: all build test race cover fuzz lint vuln scripts-check integration smoke up down logs reload-scripts clean

all: build test

## build: compile the l2go binary into bin/
build:
	$(GO) build -o bin/l2go ./cmd/l2go

## test: run the unit tests (no database needed)
test:
	$(GO) test ./...

## race: run the unit tests with the race detector, in random order
race:
	$(GO) test -race -shuffle=on ./...

## cover: race tests with a coverage report
cover:
	$(GO) test -race -shuffle=on -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

## fuzz: run every fuzz target for FUZZTIME (default 60s)
fuzz:
	$(GO) test ./internal/packets -run '^$$' -fuzz '^FuzzReadPacket$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/packets -run '^$$' -fuzz '^FuzzReader$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/packets -run '^$$' -fuzz '^FuzzStringRoundTrip$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/blowfish -run '^$$' -fuzz '^FuzzDecrypt$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/logincrypt -run '^$$' -fuzz '^FuzzDecryptClientPacket$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/logincrypt -run '^$$' -fuzz '^FuzzClientDecryptServerPacket$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/logincrypt -run '^$$' -fuzz '^FuzzXORPass$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/logincrypt -run '^$$' -fuzz '^FuzzChecksum$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/rsakey -run '^$$' -fuzz '^FuzzDecryptRaw$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/rsakey -run '^$$' -fuzz '^FuzzParseCredentials$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/rsakey -run '^$$' -fuzz '^FuzzUnscramble$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/crypt/gamecrypt -run '^$$' -fuzz '^FuzzDecrypt$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/link -run '^$$' -fuzz '^FuzzDecode$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/loginserver/clientpackets -run '^$$' -fuzz '^FuzzParsers$$' -fuzztime $(FUZZTIME)
	$(GO) test ./internal/gameserver/clientpackets -run '^$$' -fuzz '^FuzzParsers$$' -fuzztime $(FUZZTIME)

## lint: golangci-lint (version pinned in tools/go.mod)
lint:
	$(TOOL) golangci-lint run ./...

## vuln: govulncheck (version pinned in tools/go.mod)
vuln:
	$(TOOL) govulncheck ./...

## scripts-check: load every Lua script in scripts/ without starting a server
scripts-check:
	$(GO) run ./cmd/l2go scripts check -dir scripts

## integration: MongoDB tests against the compose mongo service
integration:
	docker compose up -d --wait mongo
	L2GO_MONGO_URI=$(MONGO_URI) $(GO) test -tags integration -count=1 ./internal/storage/...

## smoke: run the full client flow against the running stack (make up first;
## L2GO_LOGIN_ADDR and L2GO_GAME_ADDR change the addresses)
smoke:
	$(GO) test -tags smoke -count=1 ./internal/e2e/

## up: build the image and run MongoDB, the login server and game server 1
up:
	docker compose up --build

## down: stop the compose stack (data stays in the mongo-data volume)
down:
	docker compose down

## logs: follow the compose logs
logs:
	docker compose logs -f

## reload-scripts: reload the Lua scripts of the running compose game server
reload-scripts:
	docker compose kill -s HUP game

clean:
	rm -rf bin coverage.out
