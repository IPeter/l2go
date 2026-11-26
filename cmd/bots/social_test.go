package main

import (
	"io"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

func TestPartyGroup(t *testing.T) {
	tests := []struct {
		id, size, n int
		leader      int
		members     []int
	}{
		{0, 3, 10, 0, []int{1, 2}},
		{1, 3, 10, 0, []int{1, 2}},
		{5, 3, 10, 3, []int{4, 5}},
		{9, 3, 10, 9, nil}, // the last bot leads a party of one
		{7, 3, 8, 6, []int{7}},
		{4, 2, 10, 4, []int{5}},
	}
	for _, tt := range tests {
		leader, members := partyGroup(tt.id, tt.size, tt.n)
		if leader != tt.leader || !slices.Equal(members, tt.members) {
			t.Errorf("partyGroup(%d, %d, %d) = %d, %v; want %d, %v", tt.id, tt.size, tt.n, leader, members, tt.leader, tt.members)
		}
	}
}

func TestInitSocial(t *testing.T) {
	cfg := config{prefix: "bot", n: 6, mode: modeHunt, class: classMystic, social: socialConfig{party: 3}}
	stats := NewStats(6, time.Now(), 1)
	leader := newBot(3, cfg, stats, slog.New(slog.DiscardHandler))
	leader.initSocial()
	if leader.soc.leader != "" || !slices.Equal(leader.soc.members, []string{"Bot0004M", "Bot0005M"}) {
		t.Errorf("leader: %+v", leader.soc)
	}
	member := newBot(4, cfg, stats, slog.New(slog.DiscardHandler))
	member.initSocial()
	if member.soc.leader != "Bot0003M" || len(member.soc.members) != 0 {
		t.Errorf("member: %+v", member.soc)
	}
	cfg.social.party = 0
	solo := newBot(4, cfg, stats, slog.New(slog.DiscardHandler))
	solo.initSocial()
	if solo.soc.leader != "" || len(solo.soc.members) != 0 {
		t.Errorf("without parties: %+v", solo.soc)
	}
}

func TestAssistTarget(t *testing.T) {
	monster := testclient.Object{ObjectID: 10, Kind: testclient.KindNpc, Attackable: true}
	dead := testclient.Object{ObjectID: 11, Kind: testclient.KindNpc, Attackable: true, Dead: true}
	npc := testclient.Object{ObjectID: 12, Kind: testclient.KindNpc}
	objs := []testclient.Object{monster, dead, npc}
	leader := func(target int32) testclient.Object {
		return testclient.Object{ObjectID: 1, Kind: testclient.KindPlayer, Motion: testclient.Motion{TargetID: target}}
	}
	if o, ok := assistTarget(leader(10), objs); !ok || o.ObjectID != 10 {
		t.Errorf("leader on the monster: %+v, %v", o, ok)
	}
	for _, target := range []int32{0, 11, 12, 99} {
		if o, ok := assistTarget(leader(target), objs); ok {
			t.Errorf("leader targeting %d: assisted on %+v", target, o)
		}
	}

	// A member that fights nothing takes the leader's target; one that
	// fights keeps its own.
	w := testclient.NewWorld()
	w.Apply(testclient.CharInfoPacket{ObjectID: 1, Name: "Bot0000M"})
	w.Apply(testclient.NpcInfo{ObjectID: 10, Attackable: true})
	w.Apply(testclient.TargetSelected{ObjectID: 1, TargetID: 10})
	b := newBot(1, config{prefix: "bot", n: 3, mode: modeHunt, class: classMystic, social: socialConfig{party: 3}},
		NewStats(3, time.Now(), 1), slog.New(slog.DiscardHandler))
	b.initSocial()
	now := time.Now()
	b.assist(w, now)
	if b.target != 10 || b.attacking != 0 || !b.fightStart.Equal(now) {
		t.Errorf("member target %d attacking %d", b.target, b.attacking)
	}
	b.target = 20
	b.assist(w, now)
	if b.target != 20 {
		t.Errorf("a fighting member switched to %d", b.target)
	}
}

func TestTradePartner(t *testing.T) {
	player := func(id int32, x int32, store uint8) testclient.Object {
		return testclient.Object{ObjectID: id, Kind: testclient.KindPlayer, Motion: testclient.Motion{Pos: testclient.Position{X: x}},
			Info: testclient.CharInfoPacket{ObjectID: id, PrivateStoreType: store}}
	}
	me := testclient.Position{}
	objs := []testclient.Object{
		player(1, 300, 1), // a store
		player(2, 500, 0),
		player(3, -500, 0),
		{ObjectID: 4, Kind: testclient.KindNpc, Motion: testclient.Motion{Pos: testclient.Position{X: 10}}},
		player(5, 2000, 0),
	}
	o, d, ok := tradePartner(me, objs, 1500)
	if !ok || o.ObjectID != 2 || d != 500 {
		t.Errorf("tradePartner = %d at %v, %v; want 2 at 500 (the lower of two)", o.ObjectID, d, ok)
	}
	if _, _, ok := tradePartner(me, objs[:1], 1500); ok {
		t.Error("a player in store mode was taken as a trade partner")
	}
}

func TestTradeOffer(t *testing.T) {
	if _, ok := tradeOffer(nil); ok {
		t.Error("an offer from an empty window")
	}
	items := []testclient.TradeItem{{ObjectID: 1, Count: 0}, {ObjectID: 2, Count: 20}, {ObjectID: 3, Count: 1}}
	if o, ok := tradeOffer(items); !ok || o != (testclient.TradeOffer{ObjectID: 2, Count: 1}) {
		t.Errorf("tradeOffer = %+v, %v", o, ok)
	}
}

func TestFriendPicks(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	friends := []testclient.Friend{{Name: "Away"}, {Name: "Here", Online: true}}
	for range 10 {
		if f, ok := pickOnlineFriend(friends, rng); !ok || f.Name != "Here" {
			t.Fatalf("pickOnlineFriend = %+v, %v", f, ok)
		}
	}
	if _, ok := pickOnlineFriend(friends[:1], rng); ok {
		t.Error("an offline friend was picked")
	}
	player := func(id int32, name string, x int32) testclient.Object {
		return testclient.Object{ObjectID: id, Name: name, Kind: testclient.KindPlayer, Motion: testclient.Motion{Pos: testclient.Position{X: x}}}
	}
	objs := []testclient.Object{player(1, "Here", 10), player(2, "Far", 3000), player(3, "New", 400)}
	if o, ok := friendCandidate(testclient.Position{}, objs, friends, 1500); !ok || o.Name != "New" {
		t.Errorf("friendCandidate = %+v, %v; want New (Here is a friend, Far out of range)", o, ok)
	}
	if _, ok := friendCandidate(testclient.Position{}, objs[:2], friends, 1500); ok {
		t.Error("a candidate among friends and far players")
	}
}

func TestParseFlagsSocial(t *testing.T) {
	cfg, err := parseFlags(nil, io.Discard)
	if err != nil || cfg.social != (socialConfig{}) || cfg.social.enabled() {
		t.Fatalf("defaults: %+v, %v; want everything off", cfg.social, err)
	}
	cfg, err = parseFlags([]string{"-mode", "hunt", "-party", "3", "-ping", "1s"}, io.Discard)
	if err != nil || cfg.social.party != 3 || cfg.social.ping != time.Second || !cfg.social.enabled() {
		t.Errorf("hunt in parties: %+v, %v", cfg.social, err)
	}
	cfg, err = parseFlags([]string{"-trade", "0.1", "-friend", "0.2"}, io.Discard)
	if err != nil || cfg.social.trade != 0.1 || cfg.social.friend != 0.2 {
		t.Errorf("trades and friends: %+v, %v", cfg.social, err)
	}
	for _, args := range [][]string{
		{"-party", "3"},                  // not hunting
		{"-mode", "hunt", "-party", "1"}, // too small
		{"-mode", "hunt", "-party", "10"},
		{"-trade", "-0.1"},
		{"-trade", "0.6", "-friend", "0.6"},
		{"-mode", "hunt", "-trade", "0.1"},
		{"-ping", "-1s"},
	} {
		if _, err := parseFlags(args, io.Discard); err == nil {
			t.Errorf("parseFlags(%q) accepted it", args)
		}
	}
}

func TestSocialDescribe(t *testing.T) {
	var b strings.Builder
	socialConfig{}.describe(&b)
	if b.Len() != 0 {
		t.Errorf("describe without social flags: %q", b.String())
	}
	socialConfig{party: 3, trade: 0.1, ping: time.Second}.describe(&b)
	for _, want := range []string{"parties of 3", "trade with probability 0.1", "every 1s"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("describe %q lacks %q", b.String(), want)
		}
	}
}

func TestReportTickLine(t *testing.T) {
	s := NewStats(500, time.Now(), 1)
	for range 499 {
		s.Move(stateIdle, stateInWorld)
	}
	for i := range 100 {
		s.Observe("ping", time.Duration(i+1)*time.Millisecond)
	}
	var b strings.Builder
	if err := s.Report(time.Now(), false).Write(&b); err != nil {
		t.Fatal(err)
	}
	want := "world loop round trip at 499 bots in the world: p50 50.0ms, p90 90.0ms, p99 99.0ms, max 100.0ms (100 pings)"
	if !strings.Contains(b.String(), want) {
		t.Errorf("report:\n%s\nwant the line %q", b.String(), want)
	}
	// Without pings there is no such line.
	b.Reset()
	s.Observe("move", time.Millisecond)
	if err := s.Report(time.Now(), false).Write(&b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "world loop") {
		t.Errorf("a report without pings has the world loop line:\n%s", b.String())
	}
}
