//go:build smoke

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// TestSmokeWorldAI walks the world AI steps (a door, a raid kill, boarding
// the ferry) against servers that are already running, as TestSmoke does.
// The steps need GM commands, and the smoke test cannot make a character a
// GM, so it plays an existing GM account: L2GO_GM_ACCOUNT and
// L2GO_GM_PASSWORD, whose first character must have GM rights. It skips
// without them. The game server is at L2GO_GAME_ADDR (default
// 127.0.0.1:7777).
//
// The live server keeps its own raid boss: the raid step kills a Kutus it
// spawns with //spawn, so nothing is stored. The game time is real, so the
// ferry step boards only when the ferry stands at the Talking Island pier
// and does not wait for its departure.
func TestSmokeWorldAI(t *testing.T) {
	account, password := os.Getenv("L2GO_GM_ACCOUNT"), os.Getenv("L2GO_GM_PASSWORD")
	if account == "" || password == "" {
		t.Skip("set L2GO_GM_ACCOUNT and L2GO_GM_PASSWORD to an account whose first character is a GM")
	}
	loginAddr := envOr("L2GO_LOGIN_ADDR", "127.0.0.1:2106")
	gameAddr := envOr("L2GO_GAME_ADDR", "127.0.0.1:7777")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	keys, err := testclient.LoginAndSelect(ctx, loginAddr, account, password, serverID)
	if err != nil {
		t.Fatalf("logging in as %s: %v", account, err)
	}
	c, err := testclient.DialGame(ctx, gameAddr, testclient.InterludeProtocol)
	if err != nil {
		t.Fatalf("DialGame(%s): %v", gameAddr, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	list, err := c.AuthLogin(ctx, account, keys)
	if err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
	if len(list.Characters) == 0 {
		t.Fatalf("account %s has no character", account)
	}
	if _, err := c.SelectCharacter(ctx, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	res, err := c.EnterWorld(ctx)
	if err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}
	if !res.UserInfo.GM {
		t.Skipf("the first character of %s is not a GM", account)
	}
	worldAIWalk{spawnRaid: true}.run(t, ctx, c)
	if err := c.Logout(ctx); err != nil {
		t.Errorf("Logout: %v", err)
	}
}
