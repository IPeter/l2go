package serverpackets

// The sub-opcodes of the fishing packets (Plan 14, M11g; Mobius
// ServerPackets EX_FISHING_*).
const (
	ExOpFishingStart       = 0x13
	ExOpFishingEnd         = 0x14
	ExOpFishingStartCombat = 0x15
	ExOpFishingHpRegen     = 0x16
)

// ExFishingStart shows the cast: the float lands at X, Y, Z. Mobius
// broadcasts it to the fisher and everyone around.
type ExFishingStart struct {
	ObjectID int32
	// FishType is the hooked fish's group, -1 when the lure does not fit
	// the time of day (that fish never bites).
	FishType int32
	X, Y, Z  int32
	Night    bool // a night lure
	// RankButton shows the fish ranking button; Mobius always sends false.
	RankButton bool
}

// NewExFishingStartPacket builds ExFishingStart (Mobius
// ExFishingStart.writeImpl): d object, d fish type, d x, d y, d z, c night
// lure, c rank button.
func NewExFishingStartPacket(e ExFishingStart) []byte {
	w := newExWriter(ExOpFishingStart, 22)
	w.WriteInt32(e.ObjectID)
	w.WriteInt32(e.FishType)
	w.WriteInt32(e.X)
	w.WriteInt32(e.Y)
	w.WriteInt32(e.Z)
	putBool8(w, e.Night)
	putBool8(w, e.RankButton)
	return w.Bytes()
}

// ExFishingEnd ends the fishing; Win is set when the fish was landed.
type ExFishingEnd struct {
	ObjectID int32
	Win      bool
}

// NewExFishingEndPacket builds ExFishingEnd (Mobius ExFishingEnd.writeImpl):
// d object, c win.
func NewExFishingEndPacket(e ExFishingEnd) []byte {
	w := newExWriter(ExOpFishingEnd, 5)
	w.WriteInt32(e.ObjectID)
	putBool8(w, e.Win)
	return w.Bytes()
}

// ExFishingStartCombat opens the fight window when a fish bites.
type ExFishingStartCombat struct {
	ObjectID int32
	Time     int32 // seconds
	HP       int32
	Mode     uint8 // 0 resting, 1 fighting
	// LureType is 0 for a beginners' lure, 1 for a normal one, 2 for a
	// night lure or an upper-grade fish.
	LureType  uint8
	Deceptive uint8 // 1: pumping and reeling work the other way round
}

// NewExFishingStartCombatPacket builds ExFishingStartCombat (Mobius
// ExFishingStartCombat.writeImpl): d object, d time, d HP, c mode, c lure
// type, c deceptive mode.
func NewExFishingStartCombatPacket(e ExFishingStartCombat) []byte {
	w := newExWriter(ExOpFishingStartCombat, 15)
	w.WriteInt32(e.ObjectID)
	w.WriteInt32(e.Time)
	w.WriteInt32(e.HP)
	w.WriteUint8(e.Mode)
	w.WriteUint8(e.LureType)
	w.WriteUint8(e.Deceptive)
	return w.Bytes()
}

// ExFishingHpRegen updates the fight window: every second, and after each
// pumping or reeling.
type ExFishingHpRegen struct {
	ObjectID int32
	Time     int32
	HP       int32
	Mode     uint8 // 0 the HP stays, 1 it rises
	GoodUse  uint8 // 0 none, 1 success, 2 failure
	Anim     uint8 // the skill's animation (Mobius: 1 pumping, 2 reeling)
	Penalty  int32
	// BarColor is 1 for the purple bar of a deceptive fish.
	BarColor uint8
}

// NewExFishingHpRegenPacket builds ExFishingHpRegen (Mobius
// ExFishingHpRegen.writeImpl): d object, d time, d HP, c mode, c good use,
// c anim, d penalty, c bar color.
func NewExFishingHpRegenPacket(e ExFishingHpRegen) []byte {
	w := newExWriter(ExOpFishingHpRegen, 20)
	w.WriteInt32(e.ObjectID)
	w.WriteInt32(e.Time)
	w.WriteInt32(e.HP)
	w.WriteUint8(e.Mode)
	w.WriteUint8(e.GoodUse)
	w.WriteUint8(e.Anim)
	w.WriteInt32(e.Penalty)
	w.WriteUint8(e.BarColor)
	return w.Bytes()
}
