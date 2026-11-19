package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpNpcInfo is the opcode of NpcInfo.
const OpNpcInfo = 0x16

// NpcInfo shows an NPC or a monster to a player: its model, position,
// speeds, equipment, name and title. Mobius sends ServerObjectInfo instead
// for an NPC whose run speed is 0.
//
// The fields hold the values as Mobius writes them; the comments say where
// Mobius takes them from. The builder only does what Mobius
// AbstractNpcInfo.NpcInfo.writeImpl does itself: the NPC ID offset, the
// summon and environment codes, and the title and stealth effect of an
// invisible NPC. Mobius sends nothing for an NPC that has decayed, so don't
// build the packet for one.
type NpcInfo struct {
	ObjectID int32
	// DisplayID is the template's display ID (usually the NPC ID). The
	// builder adds the 1,000,000 offset the client expects.
	DisplayID int32
	// Attackable is whether the viewer can attack the NPC (Mobius
	// isAutoAttackable): monsters are, town NPCs are not.
	Attackable bool
	X, Y, Z    int32
	Heading    int32
	MAtkSpd    int32 // casting speed
	PAtkSpd    int32 // attack speed, cut to an int
	// Speeds as Mobius sends them: the real speed divided by MoveMultiplier
	// and rounded. The fly speeds are the run and walk speeds while flying,
	// else 0; each is sent twice.
	RunSpd      int32
	WalkSpd     int32
	SwimRunSpd  int32
	SwimWalkSpd int32
	FlyRunSpd   int32
	FlyWalkSpd  int32
	// MoveMultiplier is the movement speed multiplier.
	MoveMultiplier float64
	// AttackSpeedMultiplier is Mobius getAttackSpeedMultiplier, a Java
	// float.
	AttackSpeedMultiplier float32
	// CollisionRadius and CollisionHeight are the template's collision size,
	// Java floats in Mobius. Each is sent twice.
	CollisionRadius float32
	CollisionHeight float32
	// RHand, Chest and LHand are the item IDs of the template's equipment.
	// Mobius never sets Chest for an NPC.
	RHand int32
	Chest int32
	LHand int32
	// Running shows the NPC running rather than walking.
	Running  bool
	InCombat bool
	// Dead is set for a dead or fake-dead NPC (Mobius isAlikeDead).
	Dead bool
	// Summoned plays the summon animation when the NPC appears (Mobius
	// isShowSummonAnimation); the client shows it only if the model has one.
	Summoned bool
	// Name is the template's name for templates with usingServerSideName,
	// else "": the client then shows the name from its own data.
	Name string
	// Title is the template's title for templates with
	// usingServerSideTitle, else the NPC's own title. The builder sends
	// "Invisible" instead for an invisible NPC.
	Title string
	// Invisible marks an NPC that only GMs see.
	Invisible bool
	// AbnormalEffects is the abnormal visual effect mask. The builder adds
	// AbnormalStealth for an invisible NPC.
	AbnormalEffects int32
	// The clan and alliance of the castle owner, whose crests Mobius shows
	// over town NPCs. 0 while there are no castles.
	ClanID      int32
	ClanCrestID int32
	AllyID      int32
	AllyCrestID int32
	// InWater and Flying choose the movement environment; water wins.
	InWater bool
	Flying  bool
	// Team is the team circle under the NPC: 0 none, 1 blue, 2 red.
	Team uint8
	// EnchantEffect is the weapon enchant glow (the template's weapon
	// enchant).
	EnchantEffect int32
}

// Values of the NpcInfo summon and environment bytes.
const (
	npcSummonAnimation = 2 // the summon byte: play the summon animation
	npcEnvironmentLand = 0
	npcEnvironmentSwim = 1
	npcEnvironmentFly  = 2
)

// invisibleNpcTitle is the title Mobius shows for an invisible NPC.
const invisibleNpcTitle = "Invisible"

// environment returns the environment byte: Mobius checks water first.
func (n NpcInfo) environment() uint8 {
	switch {
	case n.InWater:
		return npcEnvironmentSwim
	case n.Flying:
		return npcEnvironmentFly
	}
	return npcEnvironmentLand
}

// NewNpcInfoPacket builds NpcInfo (Mobius AbstractNpcInfo.NpcInfo.writeImpl).
func NewNpcInfoPacket(n NpcInfo) []byte {
	title, effects := n.Title, n.AbnormalEffects
	if n.Invisible {
		title = invisibleNpcTitle
		effects |= AbnormalStealth
	}

	w := packets.NewWriter(184 + 2*(len(n.Name)+len(title)))
	w.WriteUint8(OpNpcInfo)
	w.WriteInt32(n.ObjectID)
	w.WriteInt32(n.DisplayID + clientNpcIDBase)
	putBool32(w, n.Attackable)
	w.WriteInt32(n.X)
	w.WriteInt32(n.Y)
	w.WriteInt32(n.Z)
	w.WriteInt32(n.Heading)
	w.WriteInt32(0)
	w.WriteInt32(n.MAtkSpd)
	w.WriteInt32(n.PAtkSpd)
	w.WriteInt32(n.RunSpd)
	w.WriteInt32(n.WalkSpd)
	w.WriteInt32(n.SwimRunSpd)
	w.WriteInt32(n.SwimWalkSpd)
	w.WriteInt32(n.FlyRunSpd)
	w.WriteInt32(n.FlyWalkSpd)
	w.WriteInt32(n.FlyRunSpd)
	w.WriteInt32(n.FlyWalkSpd)
	w.WriteFloat64(n.MoveMultiplier)
	w.WriteFloat64(float64(n.AttackSpeedMultiplier))
	w.WriteFloat64(float64(n.CollisionRadius))
	w.WriteFloat64(float64(n.CollisionHeight))
	w.WriteInt32(n.RHand)
	w.WriteInt32(n.Chest)
	w.WriteInt32(n.LHand)
	w.WriteUint8(1) // show the name over the NPC
	putBool8(w, n.Running)
	putBool8(w, n.InCombat)
	putBool8(w, n.Dead)
	if n.Summoned {
		w.WriteUint8(npcSummonAnimation)
	} else {
		w.WriteUint8(0)
	}
	w.WriteString(n.Name)
	w.WriteString(title)
	w.WriteInt32(0) // title colour: the client default
	w.WriteInt32(0) // PvP flag
	w.WriteInt32(0) // karma
	w.WriteInt32(effects)
	w.WriteInt32(n.ClanID)
	w.WriteInt32(n.ClanCrestID)
	w.WriteInt32(n.AllyID)
	w.WriteInt32(n.AllyCrestID)
	w.WriteUint8(n.environment())
	w.WriteUint8(n.Team)
	w.WriteFloat64(float64(n.CollisionRadius))
	w.WriteFloat64(float64(n.CollisionHeight))
	w.WriteInt32(n.EnchantEffect)
	putBool32(w, n.Flying)
	return w.Bytes()
}
