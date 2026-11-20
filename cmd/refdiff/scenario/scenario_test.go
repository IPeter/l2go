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
	want := []string{"login-create-enter", "enter-two-players", "chat", "restart", "npc-talk"}
	if !slices.Equal(names, want) {
		t.Errorf("scenarios = %v, want %v", names, want)
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
