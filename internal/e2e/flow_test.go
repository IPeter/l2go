package e2e

import (
	"context"
	"errors"
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/testclient"
)

const (
	// serverID is the game server the flow plays on.
	serverID = 1
	// forbiddenName contains a word the default Lua script refuses
	// (scripts/events/character_create.lua, the Mobius ForbiddenNames).
	forbiddenName = "Announcer"

	flowTimeout  = time.Minute
	pollTimeout  = 10 * time.Second
	pollInterval = 20 * time.Millisecond
)

// startingClasses are the classes NewCharacterSuccess lists, in order:
// human, elf, dark elf and orc fighter and mystic, then dwarf fighter.
var startingClasses = []int32{0, 10, 18, 25, 31, 38, 44, 49, 53}

// flow is one walk through the parity checklist (F1-F11) against a login
// server and its game server 1.
type flow struct {
	loginAddr string
	// gameAddr is where the client connects to the game server. Empty
	// means the address the server list announces.
	gameAddr string
	// account is typed with upper case letters, which the servers drop. It
	// must not exist yet: the first login creates it.
	account  string
	password string
	// charName is the character to create. It must be free.
	charName string
	// exclusive is set when the test owns the servers, so the player count
	// in the server list counts only the flow's own players.
	exclusive bool

	// checkAccount and checkCharacter look inside the servers when they run
	// in the test process. Either may be nil.
	checkAccount   func(t *testing.T, typed string)
	checkCharacter func(t *testing.T, account, name string)
}

// run walks the client through the whole flow. Every step is a subtest,
// and a failed step stops the flow, since the later steps build on it.
func (f flow) run(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), flowTimeout)
	defer cancel()
	account := strings.ToLower(f.account)
	step := func(name string, fn func(t *testing.T)) {
		t.Helper()
		if !t.Run(name, fn) {
			t.FailNow()
		}
	}
	// closeAtEnd closes c when the whole flow ends, not the step.
	closeAtEnd := func(c interface{ Close() error }) {
		t.Cleanup(func() { _ = c.Close() })
	}

	var (
		lc       *testclient.LoginConn
		keys     session.Keys
		gameAddr = f.gameAddr
		gc       *testclient.GameConn
	)

	step("F1 Init", func(t *testing.T) {
		c, err := testclient.DialLogin(ctx, f.loginAddr)
		if err != nil {
			t.Fatalf("DialLogin: %v", err)
		}
		closeAtEnd(c)
		lc = c
		if in := c.Init(); in.Protocol != testclient.LoginProtocol {
			t.Errorf("Init protocol %#x, want %#x", in.Protocol, testclient.LoginProtocol)
		}
	})

	step("F2 GameGuard", func(t *testing.T) {
		if err := lc.AuthGameGuard(ctx); err != nil {
			t.Fatalf("AuthGameGuard: %v", err)
		}
	})

	step("F3 login creates the account", func(t *testing.T) {
		res, err := lc.Login(ctx, f.account, f.password, false)
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		if res.Ok == nil {
			t.Fatalf("Login(%q) = %v, want LoginOk", f.account, res)
		}
		if f.checkAccount != nil {
			f.checkAccount(t, f.account)
		}
	})

	step("F4 F6 server list shows the game server up", func(t *testing.T) {
		// The game server registers on the link in the background; poll
		// until the list shows it.
		var srv testclient.ServerInfo
		poll(t, ctx, "game server 1 to be up in the server list", func() (bool, error) {
			list, err := lc.ServerList(ctx)
			if err != nil {
				return false, err
			}
			s, ok := findServer(list, serverID)
			srv = s
			return ok && s.Up, nil
		})
		if srv.MaxPlayers == 0 {
			t.Errorf("server 1 has max players 0: %+v", srv)
		}
		listed := net.JoinHostPort(srv.IP.String(), strconv.Itoa(int(srv.Port)))
		if gameAddr == "" {
			gameAddr = listed
		}
		t.Logf("server list: server 1 at %s, %d/%d players", listed, srv.CurrentPlayers, srv.MaxPlayers)
	})

	step("F5 PlayOk", func(t *testing.T) {
		play, err := lc.ServerLogin(ctx, serverID)
		if err != nil {
			t.Fatalf("ServerLogin: %v", err)
		}
		if play.Ok == nil {
			t.Fatalf("ServerLogin(1) = %v, want PlayOk", play)
		}
		keys = lc.Keys()
		// The client leaves the login server for the game server.
		_ = lc.Close()
	})

	step("F7 F8 handshake", func(t *testing.T) {
		c, err := testclient.DialGame(ctx, gameAddr, testclient.InterludeProtocol)
		if err != nil {
			t.Fatalf("DialGame(%s): %v", gameAddr, err)
		}
		closeAtEnd(c)
		gc = c
		if k := c.Key(); k.Result != 1 || !k.Encryption || k.ServerID != serverID {
			t.Errorf("KeyPacket = %+v, want result 1, encryption on, server 1", k)
		}
	})

	step("F9 empty character list", func(t *testing.T) {
		list, err := gc.AuthLogin(ctx, f.account, keys)
		if err != nil {
			t.Fatalf("AuthLogin: %v", err)
		}
		if len(list.Characters) != 0 {
			t.Fatalf("new account has characters: %+v", list.Characters)
		}
	})

	step("F10 starting classes", func(t *testing.T) {
		templates, err := gc.NewCharacter(ctx)
		if err != nil {
			t.Fatalf("NewCharacter: %v", err)
		}
		var classes []int32
		for _, tm := range templates {
			classes = append(classes, tm.ClassID)
		}
		if !slices.Equal(classes, startingClasses) {
			t.Fatalf("template classes %v, want %v", classes, startingClasses)
		}
		// Human Fighter, from the Mobius StartingClass files.
		hf := templates[0]
		if hf.Race != 0 || hf.STR.Base != 40 || hf.DEX.Base != 30 || hf.CON.Base != 43 ||
			hf.INT.Base != 21 || hf.WIT.Base != 11 || hf.MEN.Base != 25 {
			t.Errorf("Human Fighter template = %+v", hf)
		}
	})

	step("F11 forbidden name", func(t *testing.T) {
		res, err := gc.CreateCharacter(ctx, humanFighter(forbiddenName))
		if err != nil {
			t.Fatalf("CreateCharacter: %v", err)
		}
		if res.Created || res.Reason != testclient.CreateFailIncorrectName {
			t.Fatalf("%q: %+v, want CharCreateFail incorrect name", forbiddenName, res)
		}
	})

	step("F11 create a Human Fighter", func(t *testing.T) {
		res, err := gc.CreateCharacter(ctx, humanFighter(f.charName))
		if err != nil {
			t.Fatalf("CreateCharacter: %v", err)
		}
		if !res.Created {
			t.Fatalf("%q: refused with %v", f.charName, res.Reason)
		}
		if !res.ListBeforeOk {
			t.Error("CharCreateOk came before the list; Mobius sends the list first")
		}
		if len(res.Characters.Characters) != 1 {
			t.Errorf("list has %d characters, want 1", len(res.Characters.Characters))
		}
		info, ok := res.Characters.Find(f.charName)
		if !ok {
			t.Fatalf("%q is not in the list: %+v", f.charName, res.Characters.Characters)
		}
		checkNewHumanFighter(t, info, account)
		if info.SessionID != keys.PlayOK1 {
			t.Errorf("session ID %d, want PlayOK1 %d", info.SessionID, keys.PlayOK1)
		}
		if f.checkCharacter != nil {
			f.checkCharacter(t, account, f.charName)
		}
	})

	step("F9 used keys are refused", func(t *testing.T) {
		c, err := testclient.DialGame(ctx, gameAddr, testclient.InterludeProtocol)
		if err != nil {
			t.Fatalf("DialGame: %v", err)
		}
		closeAtEnd(c)
		_, err = c.AuthLogin(ctx, f.account, keys)
		if lf, ok := errors.AsType[*testclient.GameLoginFailError](err); !ok ||
			lf.Reason != testclient.GameLoginFailSystemErrorLoginLater {
			t.Fatalf("AuthLogin with used keys: %v, want LoginFail system error", err)
		}
		if err := c.ExpectClosed(ctx); err != nil {
			t.Errorf("connection not closed: %v", err)
		}
	})

	step("log out and in again", func(t *testing.T) {
		// Logging out of the character list closes the game connection.
		_ = gc.Close()
		c := loggedIn(t, ctx, f.loginAddr, f.account, f.password)
		play, err := c.ServerLogin(ctx, serverID)
		if err != nil || play.Ok == nil {
			t.Fatalf("ServerLogin = %v, %v; want PlayOk", play, err)
		}
		_ = c.Close()

		g, err := testclient.DialGame(ctx, gameAddr, testclient.InterludeProtocol)
		if err != nil {
			t.Fatalf("DialGame: %v", err)
		}
		closeAtEnd(g)
		gc = g
		list, err := g.AuthLogin(ctx, f.account, c.Keys())
		if err != nil {
			t.Fatalf("AuthLogin: %v", err)
		}
		info, ok := list.Find(f.charName)
		if !ok || len(list.Characters) != 1 {
			t.Fatalf("list after logging in again: %+v, want only %q", list.Characters, f.charName)
		}
		checkNewHumanFighter(t, info, account)
	})

	step("F3 wrong password", func(t *testing.T) {
		c := dialGG(t, ctx, f.loginAddr)
		res, err := c.Login(ctx, f.account, wrongPassword(f.password), false)
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		if res.Fail == nil || res.Fail.Reason != testclient.LoginFailUserOrPassWrong {
			t.Fatalf("Login with a wrong password = %v, want LoginFail user or password wrong", res)
		}
		if err := c.ExpectClosed(ctx); err != nil {
			t.Errorf("connection not closed: %v", err)
		}
	})

	step("F4 player count", func(t *testing.T) {
		// A second client looks at the server list while the player is on
		// the game server.
		c := loggedIn(t, ctx, f.loginAddr, f.account, f.password)
		players := func(want func(n uint16) bool, what string) {
			t.Helper()
			poll(t, ctx, what, func() (bool, error) {
				list, err := c.ServerList(ctx)
				if err != nil {
					return false, err
				}
				s, ok := findServer(list, serverID)
				return ok && s.Up && want(s.CurrentPlayers), nil
			})
		}
		if !f.exclusive {
			players(func(n uint16) bool { return n >= 1 }, "at least 1 player in the server list")
			return
		}
		players(func(n uint16) bool { return n == 1 }, "1 player in the server list")
		_ = gc.Close()
		players(func(n uint16) bool { return n == 0 }, "0 players after the player left")
	})
}

// humanFighter is a valid CharacterCreate for a male Human Fighter (class
// 0) with the template's stats, as the client sends it.
func humanFighter(name string) testclient.CreateRequest {
	return testclient.CreateRequest{
		Name: name, Race: 0, Sex: 0, ClassID: 0,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11,
		HairStyle: 1, HairColor: 2, Face: 1,
	}
}

// checkNewHumanFighter checks a listed character made from humanFighter:
// level 1 with the template's HP and MP, wearing the class's starting
// equipment (data/players/initial_equipment.json). The Dagger and the
// Tutorial Guide are carried, so they are not shown.
func checkNewHumanFighter(t *testing.T, info testclient.CharInfo, account string) {
	t.Helper()
	if info.Account != account {
		t.Errorf("account %q, want %q", info.Account, account)
	}
	if info.Level != 1 || info.ClassID != 0 || info.BaseClassID != 0 || info.Race != 0 || info.Sex != 0 ||
		info.CurrentHP != 80 || info.MaxHP != 80 || info.CurrentMP != 30 || info.MaxMP != 30 ||
		info.HairStyle != 1 || info.HairColor != 2 || info.Face != 1 || info.DeleteTimer != 0 {
		t.Errorf("listed character: %+v", info)
	}

	var want [testclient.NumEquipSlots]int32
	want[testclient.SlotChest] = 1146  // Squire's Shirt
	want[testclient.SlotLegs] = 1147   // Squire's Pants
	want[testclient.SlotRHand] = 2369  // Squire's Sword
	want[testclient.SlotRHand2] = 2369 // Mobius writes the right hand twice
	if info.EquipItemIDs != want {
		t.Errorf("equipment item IDs %v, want %v", info.EquipItemIDs, want)
	}
	objects := map[int32]bool{info.ObjectID: true}
	for _, slot := range []int{testclient.SlotChest, testclient.SlotLegs, testclient.SlotRHand} {
		id := info.EquipObjectIDs[slot]
		if id == 0 || objects[id] {
			t.Errorf("slot %d: object ID %#x is missing or used twice: %v", slot, id, info.EquipObjectIDs)
		}
		objects[id] = true
	}
	if info.EquipObjectIDs[testclient.SlotRHand2] != info.EquipObjectIDs[testclient.SlotRHand] {
		t.Errorf("the second right hand slot shows object %#x, want the sword %#x",
			info.EquipObjectIDs[testclient.SlotRHand2], info.EquipObjectIDs[testclient.SlotRHand])
	}
	for slot, id := range info.EquipObjectIDs {
		if want[slot] == 0 && id != 0 {
			t.Errorf("empty slot %d shows object %#x", slot, id)
		}
	}
}

// dialGG connects to the login server and passes GameGuard. The
// connection closes when the test ends.
func dialGG(t *testing.T, ctx context.Context, addr string) *testclient.LoginConn {
	t.Helper()
	c, err := testclient.DialLogin(ctx, addr)
	if err != nil {
		t.Fatalf("DialLogin: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if err := c.AuthGameGuard(ctx); err != nil {
		t.Fatalf("AuthGameGuard: %v", err)
	}
	return c
}

// loggedIn logs in on a new connection and expects LoginOk.
// wrongPassword returns a different password that still fits the 16-byte
// field of the login packet.
func wrongPassword(p string) string {
	if len(p) < 16 {
		return p + "x"
	}
	if p[len(p)-1] == 'x' {
		return p[:len(p)-1] + "y"
	}
	return p[:len(p)-1] + "x"
}

func loggedIn(t *testing.T, ctx context.Context, addr, user, password string) *testclient.LoginConn {
	t.Helper()
	c := dialGG(t, ctx, addr)
	res, err := c.Login(ctx, user, password, false)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.Ok == nil {
		t.Fatalf("Login(%q) = %v, want LoginOk", user, res)
	}
	return c
}

func findServer(list testclient.ServerListResult, id byte) (testclient.ServerInfo, bool) {
	for _, s := range list.Servers {
		if s.ID == id {
			return s, true
		}
	}
	return testclient.ServerInfo{}, false
}

// poll calls cond every pollInterval until it returns true. An error from
// cond, or pollTimeout passing, fails the test.
func poll(t *testing.T, ctx context.Context, what string, cond func() (bool, error)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()
	for {
		ok, err := cond()
		if err != nil {
			t.Fatalf("waiting for %s: %v", what, err)
		}
		if ok {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", what)
		case <-tick.C:
		}
	}
}
