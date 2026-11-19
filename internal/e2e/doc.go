// Package e2e holds the end-to-end tests that run the login server and the
// game server together and walk a fake Interlude client (internal/testclient)
// through everything L2Go supports: the parity checklist F1-F11 of the
// Interlude plan.
//
// TestFullFlow starts both real servers in the test process, on
// 127.0.0.1:0, with in-memory storage and the real data/ and scripts/
// folders. It runs with the other unit tests.
//
// TestSmoke (build tag smoke) runs the same flow against a stack that is
// already running, such as the one docker compose starts:
//
//	make up
//	make smoke
//
// L2GO_LOGIN_ADDR (default 127.0.0.1:2106) and L2GO_GAME_ADDR (default
// 127.0.0.1:7777) set the addresses. Every run uses a new account and
// character name, so it can run again and again against the same database.
//
// The package has no non-test code.
package e2e
