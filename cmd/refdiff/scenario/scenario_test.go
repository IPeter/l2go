package scenario

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

func TestScenariosAreRegistered(t *testing.T) {
	var names []string
	for _, sc := range All() {
		names = append(names, sc.Name)
		if sc.Run == nil || len(sc.Roles) == 0 || sc.Description == "" {
			t.Errorf("scenario %q is incomplete: %+v", sc.Name, sc)
		}
	}
	want := []string{
		"login-create-enter", "enter-two-players", "chat", "restart", "npc-talk",
		"items-equip", "items-bow", "items-drop-pickup", "shop-buy-sell", "multisell-galladucci", "warehouse-private",
		"kill-monster", "pvp-death-restart", "sample-sword", "sample-unarmed",
		"cast-wind-strike", "learn-skill", "buff-stack", "potion", "soulshot", "newbie-buffs",
		"private-store",
	}
	if !slices.Equal(names, want) {
		t.Errorf("scenarios = %v, want %v", names, want)
	}
	gm := []string{
		"items-bow", "items-drop-pickup", "shop-buy-sell", "multisell-galladucci", "warehouse-private",
		"cast-wind-strike", "learn-skill", "buff-stack", "potion", "soulshot", "newbie-buffs",
		"private-store",
	}
	for _, sc := range All() {
		if sc.GM != slices.Contains(gm, sc.Name) {
			t.Errorf("scenario %s: GM = %v", sc.Name, sc.GM)
		}
		if sc.Sample != strings.HasPrefix(sc.Name, "sample-") {
			t.Errorf("scenario %s: Sample = %v", sc.Name, sc.Sample)
		}
		if len(sc.Ignore) == 0 {
			t.Errorf("scenario %s has no default ignores", sc.Name)
		}
		// The NPC noise is ignored everywhere; a fight scenario keeps
		// SocialAction, the level-up animation.
		fight := slices.Contains(sc.Ignore, "Fight.Swings")
		if !fight && !slices.Contains(sc.Ignore, "NpcInfo.Heading") {
			t.Errorf("scenario %s does not ignore the NPC noise: %q", sc.Name, sc.Ignore)
		}
		if fight && slices.Contains(sc.Ignore, "SocialAction") {
			t.Errorf("fight scenario %s ignores SocialAction", sc.Name)
		}
	}
	if sc, ok := Find("chat"); !ok || sc.Name != "chat" || len(sc.Roles) != 2 {
		t.Errorf("Find(chat) = %+v, %v", sc, ok)
	}
	if _, ok := Find("nope"); ok {
		t.Error("Find found an unknown scenario")
	}
}

func TestNewIdentity(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	seen := map[string]bool{}
	for range 200 {
		id := newIdentity(rng, "alice")
		for _, name := range []string{id.Account, id.Char} {
			if seen[name] {
				t.Fatalf("name %q twice", name)
			}
			seen[name] = true
			if strings.Contains(strings.ToLower(name), "a") {
				t.Errorf("%q contains an a, which the forbidden words need", name)
			}
		}
		if len(id.Account) > 14 || !strings.HasPrefix(id.Account, "rd") || id.Account != strings.ToLower(id.Account) {
			t.Errorf("account %q: want rd..., lower case, at most 14 characters", id.Account)
		}
		if len(id.Char) > 16 || !strings.HasPrefix(id.Char, "Rd") {
			t.Errorf("character %q: want Rd..., at most 16 characters", id.Char)
		}
		if strings.EqualFold(id.Account, id.Char) {
			t.Errorf("account and character share their name %q", id.Account)
		}
		for _, c := range id.Account + id.Char {
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
				t.Errorf("name character %q is not a letter or digit", c)
			}
		}
	}
}

func TestSessionPlayersAndStreams(t *testing.T) {
	s := newSession(Target{Label: "l2go"}, Options{Rand: rand.New(rand.NewPCG(3, 4))})
	a, b := s.Player("alice"), s.Player("bob")
	if s.Player("alice") != a || a == b || a.Account == b.Account {
		t.Fatal("Player does not keep one player per role")
	}
	var names []string
	for _, st := range []*stream{s.newStream("login", "alice"), s.newStream("game", "alice"), s.newStream("game", "alice"), s.newStream("game", "bob")} {
		names = append(names, st.name)
	}
	if want := []string{"login alice", "game alice", "game alice 2", "game bob"}; !slices.Equal(names, want) {
		t.Errorf("stream names = %v, want %v", names, want)
	}
	s.Note("talked to %d", 7)
	c := s.capture("x", errors.New("boom"))
	if len(c.Players) != 2 || c.Players[0].Role != "alice" || len(c.Streams) != 4 || c.Notes[0] != "talked to 7" || c.Err == nil {
		t.Errorf("capture = %+v", c)
	}
}

func TestStepsStopAtTheFirstError(t *testing.T) {
	s := newSession(Target{}, Options{StepTimeout: time.Second})
	var ran []string
	step := func(name string, err error) Step {
		return Step{name, func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Errorf("step %s has no deadline", name)
			}
			ran = append(ran, name)
			return err
		}}
	}
	boom := errors.New("boom")
	err := s.Steps(t.Context(), step("one", nil), step("two", boom), step("three", nil))
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), `step "two"`) {
		t.Errorf("Steps = %v, want boom from step two", err)
	}
	if !slices.Equal(ran, []string{"one", "two"}) {
		t.Errorf("ran %v", ran)
	}
}

func TestSetupStreamsAreLeftOut(t *testing.T) {
	s := newSession(Target{Label: "l2go"}, Options{})
	s.setup = true
	setupLogin, setupGame := s.newStream("login", "alice"), s.newStream("game", "alice")
	s.setup = false
	login, game := s.newStream("login", "alice"), s.newStream("game", "alice")
	if setupLogin.name != "setup login alice" || setupGame.name != "setup game alice" {
		t.Errorf("setup stream names %q, %q", setupLogin.name, setupGame.name)
	}
	// The scenario proper keeps the names it has without a setup.
	if login.name != "login alice" || game.name != "game alice" {
		t.Errorf("stream names after the setup %q, %q", login.name, game.name)
	}
	c := s.capture("x", nil)
	var names []string
	for _, st := range c.Streams {
		names = append(names, st.Name)
	}
	if want := []string{"login alice", "game alice"}; !slices.Equal(names, want) {
		t.Errorf("captured streams %q, want %q", names, want)
	}
}

func TestPrepareFailsLikeASetup(t *testing.T) {
	// Nothing listens on port 1: the setup's login fails, and the error
	// says it was the setup.
	s := newSession(Target{Label: "l2go", LoginAddr: "127.0.0.1:1", Password: "x"}, Options{StepTimeout: time.Second})
	err := s.Prepare(t.Context(), s.Player("alice"))
	if err == nil || !strings.HasPrefix(err.Error(), "setup: ") {
		t.Errorf("Prepare = %v, want a setup error", err)
	}
	if s.setup {
		t.Error("the session stays in setup after Prepare")
	}
	if c := s.capture("x", err); len(c.Streams) != 0 {
		t.Errorf("the setup's login stream was captured: %+v", c.Streams)
	}
}

func TestLongStep(t *testing.T) {
	s := newSession(Target{}, Options{StepTimeout: 20 * time.Millisecond})
	slow := Step{"slow", func(ctx context.Context) error { return sleep(ctx, 100*time.Millisecond) }}
	if err := s.Do(t.Context(), slow); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a slow step = %v, want the step timeout", err)
	}
	if err := s.Do(t.Context(), Long(time.Second, slow)); err != nil {
		t.Errorf("a long slow step = %v", err)
	}
	if err := s.Do(t.Context(), Long(50*time.Millisecond, slow)); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a long step over its own limit = %v, want a timeout", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.Do(ctx, Long(time.Second, slow)); !errors.Is(err, context.Canceled) {
		t.Errorf("a long step of a cancelled run = %v, want context.Canceled", err)
	}
}

func TestSessionSwings(t *testing.T) {
	if n := newSession(Target{}, Options{}).Swings(); n != DefaultSwings {
		t.Errorf("default swings %d, want %d", n, DefaultSwings)
	}
	if n := newSession(Target{}, Options{Swings: 7}).Swings(); n != 7 {
		t.Errorf("swings %d, want 7", n)
	}
}

func TestFromRecordKeepsTheTime(t *testing.T) {
	now := time.Now()
	p := fromRecord(testclient.Record{Time: now, Dir: testclient.FromServer, Name: "Attack", Packet: testclient.Attack{AttackerID: 5}})
	if !p.Time.Equal(now) || p.Name != "Attack" || p.Value.(testclient.Attack).AttackerID != 5 {
		t.Errorf("packet = %+v", p)
	}
}

func TestFindLink(t *testing.T) {
	html := `<a action="bypass -h npc_268435460_Buy 3000400">Buy</a>` +
		`<a action="bypass -h npc_268435460_Sell">Sell</a>` +
		`<a action="link merchant/30004-1.htm">More</a>` +
		`<a action="bypass -h npc_268435460_multisell 300974001">Weapons</a>`
	tests := []struct {
		part string
		want Link
		ok   bool
	}{
		{"_Buy", Link{Bypass: true, Target: "npc_268435460_Buy 3000400"}, true},
		{"_sell", Link{Bypass: true, Target: "npc_268435460_Sell"}, true},
		{"30004-1", Link{Target: "merchant/30004-1.htm"}, true},
		{"multisell", Link{Bypass: true, Target: "npc_268435460_multisell 300974001"}, true},
		{"_DepositP", Link{}, false},
	}
	for _, tt := range tests {
		if got, ok := FindLink(html, tt.part); ok != tt.ok || got != tt.want {
			t.Errorf("FindLink(%q) = %+v, %v; want %+v, %v", tt.part, got, ok, tt.want, tt.ok)
		}
	}
}

func TestPlayerTemplate(t *testing.T) {
	s := newSession(Target{}, Options{Rand: rand.New(rand.NewPCG(5, 6))})
	a, b := s.Player("alice"), s.Player("bob")
	b.Template = HumanMystic
	if r := a.createRequest(); r.ClassID != 0 || r.Name != a.Char || r.STR != 40 {
		t.Errorf("default template = %+v, want a Human Fighter named %s", r, a.Char)
	}
	r := b.createRequest()
	want := testclient.CreateRequest{Name: b.Char, ClassID: 10, INT: 41, STR: 22, CON: 27, MEN: 39, DEX: 21, WIT: 20,
		HairStyle: 1, HairColor: 2, Face: 1}
	if r != want {
		t.Errorf("mystic template = %+v, want %+v", r, want)
	}
}

func TestReuseWait(t *testing.T) {
	c := testclient.Cast{Use: testclient.MagicSkillUse{ReuseDelay: 4000}}
	if got := reuseWait(c); got != 4*time.Second+settle {
		t.Errorf("reuseWait(4000 ms) = %v", got)
	}
	if got := reuseWait(testclient.Cast{Use: testclient.MagicSkillUse{ReuseDelay: -1}}); got != settle {
		t.Errorf("reuseWait(-1) = %v, want settle", got)
	}
}

// TestSkillScenariosUseFixedSpots checks the spots next to the NPCs of the
// skill scenarios against their Mobius spawns: a few steps away, within
// talking reach after the walk.
func TestSkillScenariosUseFixedSpots(t *testing.T) {
	spawns := []struct {
		name  string
		spot  testclient.Position
		spawn testclient.Position
	}{
		{"Baulro", spotBaulro, testclient.Position{X: -84729, Y: 245001, Z: -3720}},
		{"Newbie Guide", spotGuide, testclient.Position{X: -84081, Y: 243227, Z: -3728}},
	}
	for _, sp := range spawns {
		if d := sp.spot.DistanceTo(sp.spawn); d > 2*talkRange {
			t.Errorf("the spot of %s is %.0f units from its spawn", sp.name, d)
		}
	}
}

func TestRunWithoutRunFunction(t *testing.T) {
	c := Run(t.Context(), Scenario{Name: "empty", Roles: []string{"alice"}}, Target{Label: "l2go"}, Options{})
	if c.Err == nil || c.Scenario != "empty" || len(c.Players) != 1 || len(c.Streams) != 0 {
		t.Errorf("capture = %+v", c)
	}
}

func TestFirstLink(t *testing.T) {
	tests := []struct {
		html string
		want Link
		ok   bool
	}{
		{`<a action="bypass -h npc_268435457_Chat 1">Talk</a>`, Link{Bypass: true, Target: "npc_268435457_Chat 1"}, true},
		{`<a action="link default/30001-1.htm">More</a>`, Link{Target: "default/30001-1.htm"}, true},
		{`<a ACTION = "bypass npc_5_Quest" >Quest</a>`, Link{Bypass: true, Target: "npc_5_Quest"}, true},
		{`<a action="bypass -h npc_1_Chat 2">A</a><a action="link x.htm">B</a>`, Link{Bypass: true, Target: "npc_1_Chat 2"}, true},
		{`<html><body>No links here.</body></html>`, Link{}, false},
	}
	for _, tt := range tests {
		got, ok := FirstLink(tt.html)
		if ok != tt.ok || got != tt.want {
			t.Errorf("FirstLink(%s) = %+v, %v; want %+v, %v", tt.html, got, ok, tt.want, tt.ok)
		}
	}
	if s := (Link{Bypass: true, Target: "npc_1_Chat 1"}).String(); s != "bypass npc_1_Chat 1" {
		t.Errorf("String = %q", s)
	}
}

func TestPickTalkable(t *testing.T) {
	me := testclient.Position{X: 0, Y: 0, Z: 0}
	npc := func(id, npcID int32, x, y int32) testclient.Object {
		return testclient.Object{ObjectID: id, Kind: testclient.KindNpc, NpcID: npcID, Motion: testclient.Motion{Pos: testclient.Position{X: x, Y: y}}}
	}
	monster := npc(1, 20001, 10, 0)
	monster.Attackable = true
	static := testclient.Object{ObjectID: 2, Kind: testclient.KindStatic, NpcID: 30001, Motion: testclient.Motion{Pos: testclient.Position{X: 20}}}
	far := []testclient.Object{
		monster, static,
		npc(3, 30500, 300, 0),
		npc(4, 30400, 0, -300),
		npc(5, 30600, 900, 0),
	}
	o, d, ok := PickTalkable(me, far)
	if !ok || o.ObjectID != 4 || d != 300 {
		t.Errorf("PickTalkable(out of reach) = %d at %v, %v; want object 4 (the lower template of the two nearest)", o.ObjectID, d, ok)
	}
	near := slices.Concat(far, []testclient.Object{npc(6, 30009, 40, 0), npc(7, 30008, 0, 70)})
	o, d, ok = PickTalkable(me, near)
	if !ok || o.ObjectID != 7 || d != 70 {
		t.Errorf("PickTalkable(in reach) = %d at %v, %v; want object 7 (the lowest template in reach)", o.ObjectID, d, ok)
	}
	if _, _, ok := PickTalkable(me, far[:2]); ok {
		t.Error("a monster or a static object was taken as talkable")
	}
}

func TestWalkTime(t *testing.T) {
	tests := []struct {
		u    testclient.UserInfo
		dist float64
		want time.Duration
	}{
		{testclient.UserInfo{RunSpd: 100, MoveMultiplier: 1, Running: true}, 500, 6 * time.Second},
		{testclient.UserInfo{RunSpd: 100, WalkSpd: 50, MoveMultiplier: 2}, 500, 6 * time.Second},
		{testclient.UserInfo{}, 500, 6 * time.Second},
		{testclient.UserInfo{RunSpd: 1, MoveMultiplier: 1, Running: true}, 500, 15 * time.Second},
	}
	for _, tt := range tests {
		if got := walkTime(tt.u, tt.dist); got != tt.want {
			t.Errorf("walkTime(%+v, %v) = %v, want %v", tt.u, tt.dist, got, tt.want)
		}
	}
}

func TestAuthGameGuardBody(t *testing.T) {
	b := authGameGuardBody(0x01020304)
	want := []byte{4, 3, 2, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if !slices.Equal(b, want) {
		t.Errorf("body = % x, want % x", b, want)
	}
}

func TestGameAddress(t *testing.T) {
	ip := netip.MustParseAddr("127.0.0.1")
	list := testclient.ServerListResult{Servers: []testclient.ServerInfo{
		{ID: 1, IP: ip, Port: 7777, Up: true},
		{ID: 2, IP: ip, Port: 17777, Up: false},
	}}
	if addr, err := gameAddress(list, Target{ServerID: 1}); err != nil || addr != "127.0.0.1:7777" {
		t.Errorf("server 1: %q, %v", addr, err)
	}
	if _, err := gameAddress(list, Target{ServerID: 2}); err == nil {
		t.Error("a server that is down was accepted")
	}
	if _, err := gameAddress(list, Target{ServerID: 3}); err == nil {
		t.Error("a server not in the list was accepted")
	}
	if addr, err := gameAddress(list, Target{ServerID: 3, GameAddr: "10.0.0.1:7777"}); err != nil || addr != "10.0.0.1:7777" {
		t.Errorf("override: %q, %v", addr, err)
	}
}
