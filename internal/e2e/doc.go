// Package e2e holds the end-to-end tests that run the login server and the
// game server together and walk fake Interlude clients (internal/testclient)
// through everything L2Go supports: the parity checklist F1-F11 of the
// Interlude plan on the character screen, then, with two players in the
// world, the demo checklists of Milestone 3 (entering the world and the
// character window, seeing each other walk, chat, restart, delete and
// restore, logging in twice, GM commands) and Milestone 4 (NPCs, targets,
// dialogs and their links, a Lua-scripted NPC, the gatekeeper).
//
// TestFullFlow starts both real servers in the test process, on
// 127.0.0.1:0, with in-memory storage and the real data/ and scripts/
// folders. The world loop runs on a fake clock that follows real time, so
// the long walk jumps ahead instead of taking a minute, and the store makes
// the second player a GM with adena. It runs with the other unit tests.
//
// TestSmoke (build tag smoke) runs the same flow against a stack that is
// already running, such as the one docker compose starts:
//
//	make up
//	make smoke
//
// L2GO_LOGIN_ADDR (default 127.0.0.1:2106) sets the login server address.
// Like a real client, it connects to the game server at the address the
// server list announces; L2GO_GAME_ADDR overrides that. Every run uses new
// accounts and character names, so it can run again and again against the
// same database. There the game time is real time, so the walk takes about a
// minute, and the GM and gatekeeper steps skip: the test cannot reach into
// the server's store.
//
// The NPC dialog pages (data/html) are not in git: make data copies them
// from the Mobius checkout. Without them the steps that open those pages
// skip. TestSmoke looks for them in L2GO_HTML_DIR, by default the
// checkout's data/html, which the docker image is built from.
//
// The package has no non-test code.
package e2e
