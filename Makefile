GO       ?= go
TOOL     := $(GO) tool -modfile=tools/go.mod
FUZZTIME ?= 60s
MONGO_URI ?= mongodb://127.0.0.1:27017

# This machine's LAN IP: macOS (Wi-Fi or Ethernet), then Linux. Empty if none.
PUBLIC_IP ?= $(shell ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || hostname -I 2>/dev/null | cut -d' ' -f1)
PUBLIC_IP_OR_LOCAL = $(or $(strip $(PUBLIC_IP)),127.0.0.1)

.PHONY: all build data test race cover fuzz lint vuln scripts-check integration smoke up down logs reload-scripts clean

# The L2J Mobius Interlude checkout that cmd/datagen reads (not in git).
MOBIUS ?= L2J_Mobius/L2J_Mobius_CT_0_Interlude

all: build test

## build: compile the l2go binary into bin/
build:
	$(GO) build -o bin/l2go ./cmd/l2go

## data: regenerate everything cmd/datagen makes from the Mobius checkout
## (MOBIUS): the Go enums of internal/data, then all of data/, including the
## dialog pages in data/html (git-ignored)
data:
	$(GO) run ./cmd/datagen -enums -mobius $(MOBIUS)
	$(GO) run ./cmd/datagen -mobius $(MOBIUS)

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
## L2GO_LOGIN_ADDR changes the login address; the game address comes from
## the server list unless L2GO_GAME_ADDR is set)
smoke:
	$(GO) test -tags smoke -count=1 ./internal/e2e/

## up: build the image and run MongoDB, the login server and game server 1.
## The server list announces the game server at PUBLIC_IP: this machine's LAN
## IP by default, so clients on any machine of the network can connect.
## Override with make up PUBLIC_IP=1.2.3.4 (e.g. a public IP).
up:
	@echo "Game server announced at $(PUBLIC_IP_OR_LOCAL)"
	L2GO_PUBLIC_IP=$(PUBLIC_IP_OR_LOCAL) docker compose up --build

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

# Mobius reference stack, the oracle for cmd/refdiff (deploy/reference/README.md).
# Its own compose project, l2go-reference, so it never touches the l2go stack.
.PHONY: reference-up reference-down reference-logs reference-build

## reference-up: start the Mobius reference stack (login 127.0.0.1:12106, game
## 127.0.0.1:17777); the first run also compiles Mobius, which takes minutes
reference-up:
	docker compose -p l2go-reference -f deploy/reference/docker-compose.yml up -d --build

## reference-down: stop the reference stack (the database and the build stay in volumes)
reference-down:
	docker compose -p l2go-reference -f deploy/reference/docker-compose.yml down

## reference-logs: follow the reference stack's logs
reference-logs:
	docker compose -p l2go-reference -f deploy/reference/docker-compose.yml logs -f

## reference-build: rebuild the reference image and recompile Mobius into its
## volume; stops the Mobius servers first (start them with make reference-up)
reference-build:
	docker compose -p l2go-reference -f deploy/reference/docker-compose.yml build
	docker compose -p l2go-reference -f deploy/reference/docker-compose.yml stop mobius-login mobius-game
	docker compose -p l2go-reference -f deploy/reference/docker-compose.yml run --rm -e MOBIUS_FORCE_BUILD=1 mobius-build

# Load test and reference diff (cmd/bots, cmd/refdiff).
.PHONY: bots refdiff

N ?= 50
SCENARIO ?= login-create-enter

## bots: run N load-test bots (default 50) against the running stack until
## Ctrl+C, e.g. make bots N=500 BOTS_ARGS='-duration 5m -rate 50'
bots:
	$(GO) run ./cmd/bots -n $(N) $(BOTS_ARGS)

## refdiff: run SCENARIO against L2Go and the Mobius reference stack and print
## the differences (make up and make reference-up first; refdiff -list shows
## the scenarios), e.g. make refdiff SCENARIO=chat REFDIFF_ARGS='-ignore NpcSay'
refdiff:
	$(GO) run ./cmd/refdiff $(REFDIFF_ARGS) $(SCENARIO)
