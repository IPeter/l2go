package testclient

// The server packets of Milestone 6: attacks, the combat stance, death and
// revival. Every reader follows the writeImpl of the Mobius class of the
// same name, field by field.

// Flags of an Attack hit (Mobius Hit.HITFLAG_*). A hit with a soulshot
// also carries the crystal grade of the weapon in its low four bits
// (HitGradeMask).
const (
	HitSoulshot  uint8 = 0x10
	HitCritical  uint8 = 0x20
	HitShield    uint8 = 0x40
	HitMiss      uint8 = 0x80
	HitGradeMask uint8 = 0x0f
)

// Hit is one hit of Attack.
type Hit struct {
	TargetID int32
	Damage   int32
	Flags    uint8
}

// Miss reports whether the hit missed.
func (h Hit) Miss() bool { return h.Flags&HitMiss != 0 }

// Critical reports whether the hit was a critical hit.
func (h Hit) Critical() bool { return h.Flags&HitCritical != 0 }

// Shield reports whether the target blocked the hit with its shield.
func (h Hit) Shield() bool { return h.Flags&HitShield != 0 }

// Soulshot reports whether the attacker used a soulshot.
func (h Hit) Soulshot() bool { return h.Flags&HitSoulshot != 0 }

// Grade returns the crystal grade sent with a soulshot (0 none, 1 D to 5
// S), 0 without a soulshot.
func (h Hit) Grade() uint8 {
	if !h.Soulshot() {
		return 0
	}
	return h.Flags & HitGradeMask
}

// Attack is one swing of a weapon with its hits (0x05). Most weapons hit
// once; dual swords and fists hit twice, a pole also hits creatures around
// the target.
type Attack struct {
	AttackerID int32
	// Hits are in packet order. Mobius writes the first hit before the
	// attacker's position and the others, with their count, after it.
	Hits []Hit
	// X, Y and Z are the attacker's position.
	X, Y, Z int32
	// TargetX, TargetY and TargetZ are the position of the creature the
	// swing is aimed at.
	TargetX, TargetY, TargetZ int32
}

// PacketName implements Packet.
func (Attack) PacketName() string { return "Attack" }

// HitOn returns the hit on targetID.
func (a Attack) HitOn(targetID int32) (Hit, bool) {
	for _, h := range a.Hits {
		if h.TargetID == targetID {
			return h, true
		}
	}
	return Hit{}, false
}

// Damage returns the damage of every hit together.
func (a Attack) Damage() int64 {
	var d int64
	for _, h := range a.Hits {
		d += int64(h.Damage)
	}
	return d
}

func readHit(f *fields) Hit {
	return Hit{TargetID: f.i32(), Damage: f.i32(), Flags: f.u8()}
}

func readAttack(f *fields) Packet {
	var p Attack
	p.AttackerID = f.i32()
	p.Hits = append(p.Hits, readHit(f))
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		p.Hits = append(p.Hits, readHit(f))
	}
	p.TargetX = f.i32()
	p.TargetY = f.i32()
	p.TargetZ = f.i32()
	return p
}

// AutoAttackStart puts an object in the combat stance (0x2b).
type AutoAttackStart struct {
	ObjectID int32
}

// PacketName implements Packet.
func (AutoAttackStart) PacketName() string { return "AutoAttackStart" }

func readAutoAttackStart(f *fields) Packet { return AutoAttackStart{ObjectID: f.i32()} }

// AutoAttackStop ends the combat stance of an object (0x2c), 15 seconds
// after its last attack in Mobius.
type AutoAttackStop struct {
	ObjectID int32
}

// PacketName implements Packet.
func (AutoAttackStop) PacketName() string { return "AutoAttackStop" }

func readAutoAttackStop(f *fields) Packet { return AutoAttackStop{ObjectID: f.i32()} }

// Die shows a creature dying (0x06). For the player's own character it
// opens the restart window; the flags show its buttons.
type Die struct {
	ObjectID int32
	// CanTeleport shows "to village". Mobius sends the clan buttons only
	// with it.
	CanTeleport bool
	ToClanHall  bool
	ToCastle    bool
	ToSiegeHQ   bool
	// Sweepable makes a dead monster glow: its spoil can be swept.
	Sweepable bool
	// FixedRes shows the button to get up where the character died.
	FixedRes bool
}

// PacketName implements Packet.
func (Die) PacketName() string { return "Die" }

func readDie(f *fields) Packet {
	var p Die
	p.ObjectID = f.i32()
	p.CanTeleport = f.flag32("to village flag")
	p.ToClanHall = f.flag32("to clan hall flag")
	p.ToCastle = f.flag32("to castle flag")
	p.ToSiegeHQ = f.flag32("to siege HQ flag")
	p.Sweepable = f.flag32("sweepable flag")
	p.FixedRes = f.flag32("fixed resurrection flag")
	return p
}

// Revive brings a dead creature back to life (0x07).
type Revive struct {
	ObjectID int32
}

// PacketName implements Packet.
func (Revive) PacketName() string { return "Revive" }

func readRevive(f *fields) Packet { return Revive{ObjectID: f.i32()} }
