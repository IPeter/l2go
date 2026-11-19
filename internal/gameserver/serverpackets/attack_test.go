package serverpackets

import "testing"

// Mobius Attack.writeImpl:
//
//	c 0x05
//	d attackerObjId
//	first hit: d targetId, d damage, c flags
//	d attacker x, y, z
//	h hits.size() - 1
//	every other hit: d targetId, d damage, c flags
//	d target x, y, z
//
// Hit flags (Mobius Hit): soulshot ? 0x10 | ssGrade : 0, | 0x20 crit,
// | 0x40 shield, | 0x80 miss.

// m67Swing is a Human Fighter attacking a Gremlin near Talking Island.
func m67Swing() Attack {
	return Attack{
		AttackerID: 0x10000001,
		X:          -71338, Y: 258271, Z: -3104,
		TargetX: -71300, TargetY: 258300, TargetZ: -3100,
	}
}

func TestAttackPacketOneHit(t *testing.T) {
	a := m67Swing()
	a.Hits = []AttackHit{{TargetID: 0x60000001, Damage: 42}}
	var g m67Golden
	g.c(0x05).d(0x10000001)
	g.d(0x60000001).d(42).c(0x00)
	g.d(-71338).d(258271).d(-3104)
	g.h(0)
	g.d(-71300).d(258300).d(-3100)
	m67Check(t, "one hit", NewAttackPacket(a), g.b)

	// The same layout written out, with a C-grade soulshot and a critical
	// hit: flags 0x10 | 2 | 0x20.
	small := Attack{
		AttackerID: 1, X: 2, Y: 3, Z: 4, TargetX: 5, TargetY: 6, TargetZ: 7,
		Soulshot: true, Grade: 2,
		Hits: []AttackHit{{TargetID: 8, Damage: 9, Critical: true}},
	}
	m67Check(t, "written out", NewAttackPacket(small), m67Hex(t, `
		05
		01000000
		08000000 09000000 32
		02000000 03000000 04000000
		0000
		05000000 06000000 07000000`))
}

// Dual weapons hit twice: the second hit comes after the attacker's
// position and the count of the other hits.
func TestAttackPacketDualHits(t *testing.T) {
	a := m67Swing()
	a.Soulshot, a.Grade = true, 5
	a.Hits = []AttackHit{
		{TargetID: 0x60000001, Damage: 1234, Critical: true},
		{TargetID: 0x60000001, Damage: 0, Miss: true},
	}
	var g m67Golden
	g.c(0x05).d(0x10000001)
	g.d(0x60000001).d(1234).c(0x35) // soulshot | S grade | critical
	g.d(-71338).d(258271).d(-3104)
	g.h(1)
	g.d(0x60000001).d(0).c(0x95) // soulshot | S grade | miss
	g.d(-71300).d(258300).d(-3100)
	m67Check(t, "dual hits", NewAttackPacket(a), g.b)
}

// A pole hits the target and the creatures around it, in one Attack.
func TestAttackPacketPoleHits(t *testing.T) {
	a := m67Swing()
	a.Hits = []AttackHit{
		{TargetID: 0x60000001, Damage: 100},
		{TargetID: 0x60000002, Damage: 80, Shield: true},
		{TargetID: 0x60000003, Damage: 0, Miss: true},
	}
	var g m67Golden
	g.c(0x05).d(0x10000001)
	g.d(0x60000001).d(100).c(0x00)
	g.d(-71338).d(258271).d(-3104)
	g.h(2)
	g.d(0x60000002).d(80).c(0x40)
	g.d(0x60000003).d(0).c(0x80)
	g.d(-71300).d(258300).d(-3100)
	m67Check(t, "pole hits", NewAttackPacket(a), g.b)
}

// The flags byte of a hit, at offset 13 (opcode, attacker, target, damage).
func TestAttackHitFlags(t *testing.T) {
	tests := []struct {
		name     string
		soulshot bool
		grade    int32
		hit      AttackHit
		want     byte
	}{
		{"plain hit", false, 0, AttackHit{}, 0x00},
		{"miss", false, 0, AttackHit{Miss: true}, 0x80},
		{"critical", false, 0, AttackHit{Critical: true}, 0x20},
		{"shield", false, 0, AttackHit{Shield: true}, 0x40},
		{"critical, shield", false, 0, AttackHit{Critical: true, Shield: true}, 0x60},
		{"soulshot, no grade", true, 0, AttackHit{}, 0x10},
		{"soulshot, D grade", true, 1, AttackHit{}, 0x11},
		{"soulshot, A grade, critical", true, 4, AttackHit{Critical: true}, 0x34},
		{"everything", true, 5, AttackHit{Miss: true, Critical: true, Shield: true}, 0xf5},
		{"grade without a soulshot is not sent", false, 4, AttackHit{Critical: true}, 0x20},
	}
	for _, tt := range tests {
		a := m67Swing()
		a.Soulshot, a.Grade = tt.soulshot, tt.grade
		a.Hits = []AttackHit{tt.hit}
		b := NewAttackPacket(a)
		if len(b) != 40 {
			t.Fatalf("%s: %d bytes, want 40", tt.name, len(b))
		}
		if b[13] != tt.want {
			t.Errorf("%s: flags %#02x, want %#02x", tt.name, b[13], tt.want)
		}
	}
}

// Mobius sends Attack only with a hit; without hits there is nothing to
// send.
func TestAttackPacketNoHits(t *testing.T) {
	if b := NewAttackPacket(m67Swing()); b != nil {
		t.Errorf("no hits: got % x, want nil", b)
	}
	a := m67Swing()
	a.Hits = []AttackHit{}
	if b := NewAttackPacket(a); b != nil {
		t.Errorf("empty hits: got % x, want nil", b)
	}
}
