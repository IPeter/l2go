package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAttack is the opcode of Attack.
const OpAttack = 0x05

// The flags of an Attack hit (Mobius Hit.HITFLAG_*). A hit with a soulshot
// also carries the crystal grade of the weapon in its low four bits.
const (
	HitFlagSoulshot = 0x10
	HitFlagCritical = 0x20
	HitFlagShield   = 0x40
	HitFlagMiss     = 0x80
)

// AttackHit is one hit of an Attack.
type AttackHit struct {
	TargetID int32
	Damage   int32
	Miss     bool
	Critical bool
	// Shield is set when the target blocked the hit with its shield (Mobius
	// shld > 0: a block or a perfect block).
	Shield bool
}

// Attack is one swing of a creature's weapon with its hits, shown to
// everyone who sees the attacker. Most weapons hit once; dual weapons and a
// player's fists hit twice, and a pole also hits creatures around the
// target.
type Attack struct {
	AttackerID int32
	X, Y, Z    int32 // the attacker's position
	// TargetX, TargetY and TargetZ are the position of the creature the
	// attack is aimed at.
	TargetX, TargetY, TargetZ int32
	// Soulshot is set when the attacker has a soulshot charged. Every hit
	// then shows it, with Grade: the crystal grade level of the weapon, 0
	// for none, 1 (D) to 5 (S), as data.CrystalType.Level. Without a
	// soulshot Grade is not sent.
	Soulshot bool
	Grade    int32
	// Hits are in the order Mobius adds them. The first one is written
	// before the attacker's position, the others after it.
	Hits []AttackHit
}

// NewAttackPacket builds Attack (Mobius Attack.writeImpl).
//
// Mobius sends an Attack only when it has a hit (Attack.hasHits). Without
// hits this returns nil: there is nothing to send, and the game server
// connection drops an empty body as Mobius does.
func NewAttackPacket(a Attack) []byte {
	if len(a.Hits) == 0 {
		return nil
	}
	w := packets.NewWriter(31 + 9*len(a.Hits))
	w.WriteUint8(OpAttack)
	w.WriteInt32(a.AttackerID)
	putAttackHit(w, a, a.Hits[0])
	w.WriteInt32(a.X)
	w.WriteInt32(a.Y)
	w.WriteInt32(a.Z)
	putCount16(w, len(a.Hits)-1)
	for _, h := range a.Hits[1:] {
		putAttackHit(w, a, h)
	}
	w.WriteInt32(a.TargetX)
	w.WriteInt32(a.TargetY)
	w.WriteInt32(a.TargetZ)
	return w.Bytes()
}

// putAttackHit writes one hit: target, damage and the flags byte (Mobius
// Attack.writeHit and the Hit constructor).
func putAttackHit(w *packets.Writer, a Attack, h AttackHit) {
	var flags int32
	if a.Soulshot {
		flags |= HitFlagSoulshot | a.Grade
	}
	if h.Critical {
		flags |= HitFlagCritical
	}
	if h.Shield {
		flags |= HitFlagShield
	}
	if h.Miss {
		flags |= HitFlagMiss
	}
	w.WriteInt32(h.TargetID)
	w.WriteInt32(h.Damage)
	w.WriteUint8(uint8(flags)) //nolint:gosec // writeByte(int) keeps the low byte
}
