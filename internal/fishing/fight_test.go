package fishing

import (
	"fmt"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// record is an Output that writes down what the fight does.
type record struct{ events []string }

func (r *record) Message(m Message, arg int) {
	r.events = append(r.events, fmt.Sprintf("msg %d %d", m, arg))
}

func (r *record) Regen(g Regen, broadcast bool) {
	r.events = append(r.events, fmt.Sprintf("regen t%d hp%d m%d g%d a%d p%d d%d b%v", g.Time, g.HP, g.Mode, g.GoodUse, g.Anim, g.Penalty, g.Deceptive, broadcast))
}

func (r *record) Over(win bool) { r.events = append(r.events, fmt.Sprintf("over %v", win)) }

func (r *record) take() []string {
	e := r.events
	r.events = nil
	return e
}

func checkEvents(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n got %q\nwant %q", what, got, want)
	}
}

var greenFish = &data.Fish{ItemID: 6411, HP: 100, HPRegen: 4, CombatDuration: 3}

// TestNewFight checks the draws and the combat a bite starts.
func TestNewFight(t *testing.T) {
	r := rnd.NewScripted(t, rnd.Get(100, 80))
	f, c := NewFight(r, greenFish, false, false, 6519)
	if c != (Combat{Time: 3, HP: 100, Mode: 1, LureType: LureTypeNormal}) || f.ItemID() != 6411 || f.LureID() != 6519 {
		t.Errorf("normal: %+v", c)
	}
	r.Done()
	r = rnd.NewScripted(t, rnd.Get(100, 79))
	if _, c = NewFight(r, greenFish, true, false, 7807); c.Mode != 0 || c.LureType != LureTypeNewbie {
		t.Errorf("noob: %+v", c)
	}
	r.Done()
	// Upper grade: the deception first, then the mode.
	r = rnd.NewScripted(t, rnd.Get(100, 90), rnd.Get(100, 0))
	if f, c = NewFight(r, greenFish, true, true, 8505); c.Deceptive != 1 || c.LureType != LureTypeNight || c.Mode != 0 || !f.Deceptive() {
		t.Errorf("upper: %+v", c)
	}
	r.Done()
}

// TestFightLoop runs the one-second loop with a seeded script: a fighting
// fish regains HP, the mode switches every other second at >= 70, and the
// fish spits the hook when the time is up.
func TestFightLoop(t *testing.T) {
	var out record
	r := rnd.NewScripted(t, rnd.Get(100, 85)) // fights
	f, _ := NewFight(r, greenFish, false, false, 6519)
	r.Add(rnd.Get(100, 70)) // second 1: regains 4, then rests
	f.Tick(r, &out)
	checkEvents(t, "second 1", out.take(), "regen t2 hp104 m0 g0 a0 p0 d0 bfalse")
	f.Tick(r, &out) // second 2: no check
	checkEvents(t, "second 2", out.take(), "regen t1 hp104 m0 g0 a0 p0 d0 bfalse")
	r.Add(rnd.Get(100, 69)) // second 3: stays resting
	f.Tick(r, &out)
	checkEvents(t, "second 3", out.take(), "regen t0 hp104 m0 g0 a0 p0 d0 bfalse")
	f.Tick(r, &out)
	checkEvents(t, "time up", out.take(), "msg 1450 0", "over false")
	f.Tick(r, &out)
	checkEvents(t, "after the end", out.take())
	r.Done()
}

// TestFightPumpAndReel checks the skills on a resting and a fighting fish,
// the deceptive reversal, the resist roll and the endings.
func TestFightPumpAndReel(t *testing.T) {
	var out record
	r := rnd.NewScripted(t, rnd.Get(100, 0)) // rests
	f, _ := NewFight(r, greenFish, false, false, 6519)

	r.Add(rnd.Get(100, 90)) // not resisted (> 90 resists)
	f.Pump(r, 40, 0, &out)
	checkEvents(t, "pumping a resting fish", out.take(), "msg 1465 40", "regen t3 hp60 m0 g1 a1 p0 d0 btrue")

	r.Add(rnd.Get(100, 0))
	f.Reel(r, 30, 2, &out)
	checkEvents(t, "reeling a resting fish", out.take(), "msg 1468 30", "regen t3 hp90 m0 g2 a2 p2 d0 btrue")

	r.Add(rnd.Get(100, 91))
	f.Pump(r, 40, 0, &out)
	checkEvents(t, "resisted", out.take(), "msg 1464 0", "regen t3 hp90 m0 g0 a1 p0 d0 btrue")

	r.Add(rnd.Get(100, 5))
	f.Pump(r, 95, 3, &out)
	checkEvents(t, "landed", out.take(), "msg 1465 95", "msg 1672 3", "regen t3 hp0 m0 g1 a1 p3 d0 btrue", "over true")
	if !f.IsOver() {
		t.Error("the fight goes on after the fish was landed")
	}
	r.Done()

	// A deceptive fighting fish: pumping hurts it, reeling feeds it, and
	// above twice its HP it gets away.
	r = rnd.NewScripted(t, rnd.Get(100, 95), rnd.Get(100, 99)) // deceptive, fights
	f, _ = NewFight(r, greenFish, false, true, 8505)
	r.Add(rnd.Get(100, 0))
	f.Pump(r, 10, 0, &out)
	checkEvents(t, "pumping a deceptive fighting fish", out.take(), "msg 1465 10", "regen t3 hp90 m1 g1 a1 p0 d1 btrue")
	r.Add(rnd.Get(100, 0))
	f.Reel(r, 120, 0, &out)
	checkEvents(t, "reeling it", out.take(), "msg 1468 120", "regen t3 hp210 m1 g2 a2 p0 d1 btrue", "over false")
	r.Done()
}

// TestFightStolenBait checks that a fish at twice its HP steals the bait at
// the next second.
func TestFightStolenBait(t *testing.T) {
	var out record
	r := rnd.NewScripted(t, rnd.Get(100, 99))
	fish := &data.Fish{ItemID: 1, HP: 10, HPRegen: 10, CombatDuration: 30}
	f, _ := NewFight(r, fish, false, false, 6519)
	r.Add(rnd.Get(100, 0)) // keeps fighting: 20
	f.Tick(r, &out)
	checkEvents(t, "second 1", out.take(), "regen t29 hp20 m1 g0 a0 p0 d0 bfalse")
	f.Tick(r, &out)
	checkEvents(t, "bait stolen", out.take(), "msg 1451 0", "over false")
	r.Done()
}

// TestFightUpperGradeDeception checks that an upper-grade fish rolls its
// deception every other second.
func TestFightUpperGradeDeception(t *testing.T) {
	var out record
	r := rnd.NewScripted(t, rnd.Get(100, 0), rnd.Get(100, 0))
	f, _ := NewFight(r, greenFish, false, true, 8506)
	r.Add(rnd.Get(100, 0), rnd.Get(100, 90)) // mode stays, deception on
	f.Tick(r, &out)
	checkEvents(t, "deception on", out.take(), "regen t2 hp100 m0 g0 a0 p0 d1 bfalse")
	r.Done()
}
