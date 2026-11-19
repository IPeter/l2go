package testclient

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The Mobius server packets of Milestone 6, transcribed from their
// writeImpl line by line.

// mobiusAttack is Attack.writeImpl: the first hit, the attacker's
// position, the number of the other hits and the other hits, then the
// target's position. writeHit is d target, d damage, c flags.
func mobiusAttack(a Attack) []byte {
	w := msg(0x05)
	w.i32(a.AttackerID)
	writeHit := func(h Hit) { w.i32(h.TargetID).i32(h.Damage).u8(h.Flags) }
	writeHit(a.Hits[0])
	w.i32(a.X).i32(a.Y).i32(a.Z)
	w.u16(uint16(len(a.Hits) - 1))
	for _, h := range a.Hits[1:] {
		writeHit(h)
	}
	w.i32(a.TargetX).i32(a.TargetY).i32(a.TargetZ)
	return w.b
}

// mobiusDie is Die.writeImpl: d object, d can teleport, the three clan
// buttons, d sweepable, d fixed resurrection.
func mobiusDie(d Die) []byte {
	return msg(0x06).i32(d.ObjectID).i32(b2i(d.CanTeleport)).
		i32(b2i(d.ToClanHall)).i32(b2i(d.ToCastle)).i32(b2i(d.ToSiegeHQ)).
		i32(b2i(d.Sweepable)).i32(b2i(d.FixedRes)).b
}

func TestDecodeCombatPackets(t *testing.T) {
	one := Attack{AttackerID: 0x10000001, Hits: []Hit{{TargetID: 0x60000005, Damage: 31, Flags: HitCritical}},
		X: 1, Y: 2, Z: 3, TargetX: 4, TargetY: 5, TargetZ: 6}
	pole := Attack{AttackerID: 0x10000001, Hits: []Hit{
		{TargetID: 0x60000005, Damage: 40, Flags: HitSoulshot | 2},
		{TargetID: 0x60000006, Damage: 0, Flags: HitSoulshot | 2 | HitMiss},
		{TargetID: 0x60000007, Damage: 12, Flags: HitSoulshot | 2 | HitShield},
	}, X: -71000, Y: 258000, Z: -3100, TargetX: -70990, TargetY: 258010, TargetZ: -3100}

	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"Attack one hit", mobiusAttack(one), one},
		{"Attack three hits", mobiusAttack(pole), pole},
		{"AutoAttackStart", msg(0x2b).i32(0x10000001).b, AutoAttackStart{ObjectID: 0x10000001}},
		{"AutoAttackStop", msg(0x2c).i32(0x10000001).b, AutoAttackStop{ObjectID: 0x10000001}},
		{"Die monster", mobiusDie(Die{ObjectID: 0x60000005, Sweepable: true}), Die{ObjectID: 0x60000005, Sweepable: true}},
		{"Die player", mobiusDie(Die{ObjectID: 0x10000001, CanTeleport: true, ToCastle: true, FixedRes: true}),
			Die{ObjectID: 0x10000001, CanTeleport: true, ToCastle: true, FixedRes: true}},
		{"Revive", msg(0x07).i32(0x10000001).b, Revive{ObjectID: 0x10000001}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	if h, ok := pole.HitOn(0x60000007); !ok || h.Damage != 12 {
		t.Errorf("HitOn = %+v, %v", h, ok)
	}
	if _, ok := pole.HitOn(0x60000008); ok {
		t.Error("HitOn found a hit that is not there")
	}
	if d := pole.Damage(); d != 52 {
		t.Errorf("Damage() = %d, want 52", d)
	}
}

// TestAttackLayout writes an Attack of two hits byte by byte: the second
// hit and its count come after the attacker's position.
func TestAttackLayout(t *testing.T) {
	body := []byte{
		0x05,
		0x01, 0x00, 0x00, 0x10, // attacker 0x10000001
		0x05, 0x00, 0x00, 0x60, // first hit: target 0x60000005
		0x2a, 0x00, 0x00, 0x00, // damage 42
		0x31,                   // flags: soulshot, grade 1, critical
		0x0a, 0x00, 0x00, 0x00, // x 10
		0x14, 0x00, 0x00, 0x00, // y 20
		0xe2, 0xff, 0xff, 0xff, // z -30
		0x01, 0x00, // one more hit
		0x05, 0x00, 0x00, 0x60, // second hit: target 0x60000005
		0x00, 0x00, 0x00, 0x00, // damage 0
		0x91,                   // flags: soulshot, grade 1, miss
		0x0b, 0x00, 0x00, 0x00, // target x 11
		0x15, 0x00, 0x00, 0x00, // target y 21
		0xe2, 0xff, 0xff, 0xff, // target z -30
	}
	want := Attack{AttackerID: 0x10000001, Hits: []Hit{
		{TargetID: 0x60000005, Damage: 42, Flags: 0x31},
		{TargetID: 0x60000005, Damage: 0, Flags: 0x91},
	}, X: 10, Y: 20, Z: -30, TargetX: 11, TargetY: 21, TargetZ: -30}
	checkDecode(t, "Attack", body, want)
	if !bytes.Equal(mobiusAttack(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusAttack(want), body)
	}
	if n := len(mobiusAttack(Attack{Hits: []Hit{{}}})) - 1; n != 39 {
		t.Errorf("Attack of one hit is %d bytes, Mobius writes 39", n)
	}
}

func TestHitFlags(t *testing.T) {
	tests := []struct {
		flags                  uint8
		miss, crit, shield, ss bool
		grade                  uint8
	}{
		{0, false, false, false, false, 0},
		{HitMiss, true, false, false, false, 0},
		{HitCritical | HitShield, false, true, true, false, 0},
		{HitSoulshot | 5 | HitCritical, false, true, false, true, 5},
		{HitSoulshot, false, false, false, true, 0},
		// The grade bits mean nothing without the soulshot flag.
		{3, false, false, false, false, 0},
	}
	for _, tt := range tests {
		h := Hit{Flags: tt.flags}
		if h.Miss() != tt.miss || h.Critical() != tt.crit || h.Shield() != tt.shield || h.Soulshot() != tt.ss || h.Grade() != tt.grade {
			t.Errorf("flags 0x%02x: miss %v, critical %v, shield %v, soulshot %v, grade %d", tt.flags,
				h.Miss(), h.Critical(), h.Shield(), h.Soulshot(), h.Grade())
		}
	}
}

func TestDecodeCombatPacketErrors(t *testing.T) {
	die := mobiusDie(Die{})
	die[1+5*4] = 2 // the sweepable word
	one := mobiusAttack(Attack{Hits: []Hit{{}}})

	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"Attack without a hit", msg(0x05).i32(1).b, "Attack: packets: short packet"},
		{"Attack without the target position", one[:len(one)-12], "Attack: packets: short packet"},
		{"Attack hit count too high", append(one[:1+4+9+12:1+4+9+12], 0x01, 0x00), "Attack: packets: short packet"},
		{"Attack extra byte", append(append([]byte(nil), one...), 0), "Attack: 1 bytes after the last field"},
		{"AutoAttackStart short", msg(0x2b).u16(1).b, "AutoAttackStart: packets: short packet"},
		{"AutoAttackStop extra", msg(0x2c).i32(1).u8(0).b, "AutoAttackStop: 1 bytes after the last field"},
		{"Die sweepable 2", die, "Die: sweepable flag is 2"},
		{"Die short", mobiusDie(Die{})[:25], "Die: packets: short packet"},
		{"Revive short", msg(0x07).b, "Revive: packets: short packet"},
	}
	for _, tt := range tests {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}
