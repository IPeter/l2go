//go:build smoke

package e2e

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

// TestSmokeSocial runs the social flow (social_test.go) against servers
// that are already running, by default the docker compose stack, with new
// accounts and characters each run. The test cannot make the characters
// GMs or give them adena, so the store purchase, the hunt and the clan skip:
// what runs is the trade (a Tutorial Guide for nothing), the party (invite, accept,
// leave) and the friends with their notices.
func TestSmokeSocial(t *testing.T) {
	f := socialFlow{
		loginAddr: envOr("L2GO_LOGIN_ADDR", "127.0.0.1:2106"),
		gameAddr:  os.Getenv("L2GO_GAME_ADDR"),
		password:  rand.Text()[:16],
		accounts:  [2]string{"smoke" + digits(9), "smoke" + digits(9)},
		names:     [2]string{"Smoke" + digits(10), "Smoke" + digits(10)},
		htmlDir:   envOr("L2GO_HTML_DIR", filepath.Join("..", "..", "data", "html")),
	}
	t.Logf("login %s, game %s, accounts %s and %s, characters %s and %s",
		f.loginAddr, f.gameAddr, f.accounts[0], f.accounts[1], f.names[0], f.names[1])
	f.run(t)
}
