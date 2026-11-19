//go:build smoke

package e2e

import (
	"crypto/rand"
	"os"
	"testing"
)

// TestSmoke runs the full flow against servers that are already running,
// by default the docker compose stack (make up, then make smoke). Each run
// uses a new account and character, so it can be repeated against the
// same database. Like a real client, it connects to the game server at the
// address the server list announces, unless L2GO_GAME_ADDR is set.
func TestSmoke(t *testing.T) {
	f := flow{
		loginAddr: envOr("L2GO_LOGIN_ADDR", "127.0.0.1:2106"),
		gameAddr:  os.Getenv("L2GO_GAME_ADDR"),
		// At most 14 characters, the limit of the 128-byte credentials
		// block. Digits only, so no forbidden word can appear by chance.
		account:  "Smoke" + digits(9),
		password: rand.Text()[:16],
		charName: "Smoke" + digits(10),
	}
	t.Logf("login %s, game %s, account %s, character %s", f.loginAddr, f.gameAddr, f.account, f.charName)
	f.run(t)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// digits returns n random decimal digits.
func digits(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // never fails (crypto/rand)
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b)
}
