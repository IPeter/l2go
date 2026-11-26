package stats

import "testing"

// A mount's speed_on_ride replaces the base speeds (through the DEX bonus,
// as Mobius's runSpd chain); each penalty halves run and walk; the info
// packets show the ride's base speeds and its collision; on foot again the
// template's come back.
func TestMountSpeeds(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	p.SetMotion(Motion{Running: true})
	foot := p.View()
	p.SetMount(&Mount{OnRide: true, Walk: 140, Run: 140, SlowSwim: 70, FastSwim: 70, CollisionRadius: 20, CollisionHeight: 30})
	if got, want := p.RunSpeed(), 140*dex30; got != want {
		t.Errorf("RunSpeed() mounted = %v, want %v", got, want)
	}
	if got, want := p.SwimRunSpeed(), 70*dex30; got != want {
		t.Errorf("SwimRunSpeed() mounted = %v, want %v", got, want)
	}
	v := p.View()
	if v.RunSpd != 140 || v.WalkSpd != 140 || v.SwimRunSpd != 70 || v.CollisionRadius != 20 || v.CollisionHeight != 30 {
		t.Errorf("mounted view %d/%d/%d collision %v/%v", v.RunSpd, v.WalkSpd, v.SwimRunSpd, v.CollisionRadius, v.CollisionHeight)
	}
	if want := moveMul(140*dex30, 140); v.MoveMultiplier != want {
		t.Errorf("MoveMultiplier mounted = %v, want %v", v.MoveMultiplier, want)
	}

	m := p.Mount()
	m.Penalties = 2
	p.SetMount(m)
	if got, want := p.RunSpeed(), 140*dex30/4; got != want {
		t.Errorf("RunSpeed() with two penalties = %v, want %v", got, want)
	}
	if got, want := p.WalkSpeed(), 140*dex30/4; got != want {
		t.Errorf("WalkSpeed() with two penalties = %v, want %v", got, want)
	}

	// Without ride data the template's speeds stay, the collision changes.
	p.SetMount(&Mount{CollisionRadius: 9, CollisionHeight: 9})
	if got, want := p.RunSpeed(), 115*dex30; got != want {
		t.Errorf("RunSpeed() without ride data = %v, want %v", got, want)
	}

	p.SetMount(nil)
	if v := p.View(); !v.Equal(foot) {
		t.Errorf("on foot again %+v, want %+v", v, foot)
	}
}
