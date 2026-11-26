package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// cubicIDs returns p's cubic IDs, read on the loop.
func cubicIDsOf(e *testEnv, p *Player) []int16 {
	var ids []int16
	e.onLoop(func() { ids = e.srv.CubicIDs(p) })
	return ids
}

// The Storm Cubic of Summon Storm Cubic shows in UserInfo; while its owner
// is in its combat stance it casts Cubic DD (4049) at the owner's enemy on
// its 10 s timer, never more than 30 times; its owner's death clears it.
func TestCubicStormActsAndDeathClears(t *testing.T) {
	e, _ := newSummonEnv(t)
	sess, p := summonEnter(t, e, "cubic", classHumanFighter, 40)
	summonApplySkill(t, e, p, summonStormCubic, 1)
	if ids := cubicIDsOf(e, p); !slices.Equal(ids, []int16{1}) {
		t.Fatalf("cubics %v", ids)
	}
	e.advance(200 * time.Millisecond)
	ps := sess.take(t)
	var shown bool
	for _, u := range only[testclient.UserInfo](ps) {
		shown = shown || slices.Equal(u.Cubics, []uint16{1})
	}
	if !shown {
		t.Fatalf("UserInfo does not show the cubic: %v", testclient.Names(ps))
	}

	// Out of combat it waits.
	e.advance(30 * time.Second)
	for _, m := range only[testclient.MagicSkillUse](sess.take(t)) {
		if m.SkillID == 4049 {
			t.Fatal("the cubic acted out of combat")
		}
	}

	keltir := summonKeltir(t, e)
	var keltirID int32
	e.onLoop(func() {
		keltirID = int32(keltir.ID())
		e.srv.SetTarget(p, keltir)
		keltir.AddDamageHate(p, 0, 1)
		keltir.script.invul = true // it outlives the test
		e.srv.StartAutoAttack(p)
	})
	casts := 0
	for range 400 {
		e.onLoop(func() {
			if keltir.Dead {
				return
			}
			e.srv.StartAutoAttack(p) // stay in the stance
		})
		e.advance(10 * time.Second)
		for _, m := range only[testclient.MagicSkillUse](sess.take(t)) {
			if m.SkillID == 4049 && m.TargetID == keltirID && m.CasterID == p.ObjectID() {
				casts++
			}
		}
		if len(cubicIDsOf(e, p)) == 0 {
			break
		}
	}
	if casts == 0 || casts > 30 {
		t.Fatalf("the cubic cast %d times", casts)
	}

	// A new cubic; the owner dies: no more cubics.
	summonApplySkill(t, e, p, summonStormCubic, 1)
	if len(cubicIDsOf(e, p)) != 1 {
		t.Fatal("no new cubic")
	}
	e.onLoop(func() { e.srv.Kill(p, nil) })
	if ids := cubicIDsOf(e, p); len(ids) != 0 {
		t.Fatalf("cubics after death %v", ids)
	}
}

// A cubic leaves after its duration (900 s); a second cubic of the same
// kind replaces the first; without Cubic Mastery a second kind replaces the
// first.
func TestCubicDurationAndLimit(t *testing.T) {
	e, _ := newSummonEnv(t)
	_, p := summonEnter(t, e, "cubic2", classHumanFighter, 40)
	summonApplySkill(t, e, p, summonStormCubic, 1)
	summonApplySkill(t, e, p, summonStormCubic, 1)
	if ids := cubicIDsOf(e, p); !slices.Equal(ids, []int16{1}) {
		t.Fatalf("cubics %v", ids)
	}
	summonApplySkill(t, e, p, 22, 1) // Summon Vampiric Cubic
	if ids := cubicIDsOf(e, p); !slices.Equal(ids, []int16{2}) {
		t.Fatalf("cubics after a second kind %v", ids)
	}
	e.onLoop(func() { e.srv.summonOwnerOf(p).maxCubics = 2 })
	summonApplySkill(t, e, p, summonStormCubic, 1)
	if ids := cubicIDsOf(e, p); !slices.Equal(ids, []int16{2, 1}) {
		t.Fatalf("cubics with mastery %v", ids)
	}
	e.advance(901 * time.Second)
	if ids := cubicIDsOf(e, p); len(ids) != 0 {
		t.Fatalf("cubics after 900 s %v", ids)
	}
}
