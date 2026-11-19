package main

import (
	"math/rand/v2"
	"net/netip"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

func TestBotNames(t *testing.T) {
	tests := []struct {
		prefix        string
		id, n         int
		account, char string
	}{
		{"bot", 7, 50, "bot0007", "Bot0007"},
		{"bot", 0, 1, "bot0000", "Bot0000"},
		{"x", 12345, 20000, "x12345", "X12345"},
		{"load2", 3, 500, "load20003", "Load20003"},
	}
	for _, tt := range tests {
		account, char := botNames(tt.prefix, tt.id, tt.n)
		if account != tt.account || char != tt.char {
			t.Errorf("botNames(%q, %d, %d) = %q, %q; want %q, %q", tt.prefix, tt.id, tt.n, account, char, tt.account, tt.char)
		}
	}
}

func TestEstimate(t *testing.T) {
	from, to := testclient.Position{X: 0, Y: 0, Z: -100}, testclient.Position{X: 1000, Y: 0, Z: -100}
	tests := []struct {
		elapsed time.Duration
		want    testclient.Position
	}{
		{0, from},
		{-time.Second, from},
		{5 * time.Second, testclient.Position{X: 500, Y: 0, Z: -100}},
		{2500 * time.Millisecond, testclient.Position{X: 250, Y: 0, Z: -100}},
		{20 * time.Second, to},
	}
	for _, tt := range tests {
		if got := estimate(from, to, tt.elapsed, 100); got != tt.want {
			t.Errorf("estimate after %v = %+v, want %+v", tt.elapsed, got, tt.want)
		}
	}
	if got := estimate(from, from, time.Second, 100); got != from {
		t.Errorf("standing still: %+v", got)
	}
	if got := estimate(from, to, time.Second, 0); got != to {
		t.Errorf("without a speed the estimate is the destination, got %+v", got)
	}
}

func TestHeading(t *testing.T) {
	o := testclient.Position{}
	tests := []struct {
		to   testclient.Position
		want int32
	}{
		{testclient.Position{X: 10}, 0},
		{testclient.Position{Y: 10}, 16384},
		{testclient.Position{X: -10}, 32768},
		{testclient.Position{Y: -10}, 49152},
		{testclient.Position{X: 10, Y: 10}, 8192},
	}
	for _, tt := range tests {
		// Mobius truncates, so the axis values may come out one lower.
		if got := heading(o, tt.to); got < tt.want-1 || got > tt.want {
			t.Errorf("heading to %+v = %d, want %d (or one less)", tt.to, got, tt.want)
		}
	}
}

func TestRandomPoint(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	center := testclient.Position{X: -71338, Y: 258271, Z: -3104}
	const radius = 300
	far := 0
	for range 2000 {
		p := randomPoint(rng, center, radius)
		d := p.DistanceTo(center)
		if p.Z != center.Z || d > radius+1 {
			t.Fatalf("point %+v is %v from the center, Z %d", p, d, p.Z)
		}
		if d > radius/2 {
			far++
		}
	}
	// Uniform in the disc: three quarters of the points lie in the outer
	// ring between r/2 and r.
	if far < 1300 || far > 1700 {
		t.Errorf("%d of 2000 points in the outer ring, want about 1500", far)
	}
	if p := randomPoint(rng, center, 0); p != center {
		t.Errorf("radius 0 gives %+v", p)
	}
}

func TestNearestNpc(t *testing.T) {
	me := testclient.Position{}
	at := func(x int32) testclient.Motion { return testclient.Motion{Pos: testclient.Position{X: x}} }
	objs := []testclient.Object{
		{ObjectID: 1, Kind: testclient.KindNpc, Attackable: true, Motion: at(10)},
		{ObjectID: 2, Kind: testclient.KindStatic, Motion: at(20)},
		{ObjectID: 3, Kind: testclient.KindPlayer, Motion: at(30)},
		{ObjectID: 5, Kind: testclient.KindNpc, Motion: at(400)},
		{ObjectID: 4, Kind: testclient.KindNpc, Motion: at(-400)},
		{ObjectID: 6, Kind: testclient.KindNpc, Motion: at(2000)},
	}
	if o, ok := nearestNpc(me, objs, 1500); !ok || o.ObjectID != 4 {
		t.Errorf("nearestNpc = %d, %v; want 4 (the lower ID of two at 400)", o.ObjectID, ok)
	}
	if _, ok := nearestNpc(me, objs, 100); ok {
		t.Error("found an NPC beyond the range")
	}
}

func TestMoveSpeed(t *testing.T) {
	tests := []struct {
		u    testclient.UserInfo
		want float64
	}{
		{testclient.UserInfo{RunSpd: 115, WalkSpd: 80, MoveMultiplier: 1, Running: true}, 115},
		{testclient.UserInfo{RunSpd: 115, WalkSpd: 80, MoveMultiplier: 1.5}, 120},
		{testclient.UserInfo{RunSpd: 100, Running: true}, 100},
		{testclient.UserInfo{}, defaultSpeed},
	}
	for _, tt := range tests {
		if got := moveSpeed(tt.u); got != tt.want {
			t.Errorf("moveSpeed(%+v) = %v, want %v", tt.u, got, tt.want)
		}
	}
}

func TestJitter(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	for range 1000 {
		if d := jitter(rng, 2*time.Second); d < time.Second || d > 3*time.Second {
			t.Fatalf("jitter(2s) = %v, want 1s-3s", d)
		}
	}
}

func TestServerAddr(t *testing.T) {
	ip := netip.MustParseAddr("192.168.0.5")
	list := testclient.ServerListResult{Servers: []testclient.ServerInfo{
		{ID: 1, IP: ip, Port: 7777, Up: true},
		{ID: 2, IP: ip, Port: 7778},
	}}
	if addr, err := serverAddr(list, 1); err != nil || addr != "192.168.0.5:7777" {
		t.Errorf("server 1: %q, %v", addr, err)
	}
	if _, err := serverAddr(list, 2); err == nil || errorKind(staged("login", err)) != "login:refused" {
		t.Errorf("a server that is down: %v", err)
	}
	if _, err := serverAddr(list, 3); err == nil {
		t.Error("a missing server was found")
	}
}
