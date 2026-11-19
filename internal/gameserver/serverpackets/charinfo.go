package serverpackets

import (
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
)

// OpCharInfo is the opcode of CharInfo.
const OpCharInfo = 0x03

// charInfoSlots is the order of the equipment list in CharInfo: only the
// slots other players can see, the right hand twice.
var charInfoSlots = [...]storage.PaperdollSlot{
	storage.PaperdollUnder,
	storage.PaperdollHead,
	storage.PaperdollRHand,
	storage.PaperdollLHand,
	storage.PaperdollGloves,
	storage.PaperdollChest,
	storage.PaperdollLegs,
	storage.PaperdollFeet,
	storage.PaperdollCloak,
	storage.PaperdollRHand,
	storage.PaperdollHair,
	storage.PaperdollHair2,
}

// CharInfo is what other players see of a player: the model, the name and
// title, and the movement state. Clan and siege relations follow in
// RelationChanged.
//
// The fields hold the values as Mobius writes them. The builder only does
// the conversions Mobius CharInfo.writeImpl does itself: cutting the
// current CP to an int and inverting Sitting.
//
// For a GM who can see an invisible player, Mobius sends Title "Invisible",
// Invisible false and AbnormalStealth added to AbnormalEffects. With a
// cursed weapon equipped it sends zero clan and alliance IDs and crests.
type CharInfo struct {
	X, Y, Z     int32 // the position on the boat when on one
	VehicleID   int32 // the boat the player is on, or 0
	ObjectID    int32
	Name        string // the visible name
	Race        int32
	Female      bool
	BaseClassID int32
	// Paperdoll is the equipment; only the item IDs are sent.
	Paperdoll Paperdoll
	// WeaponAugmentationID is the augmentation of the right-hand item, or 0.
	WeaponAugmentationID int32
	PvPFlag              int32 // sent twice, with Karma
	Karma                int32
	MAtkSpd              int32
	PAtkSpd              int32
	// Speeds as Mobius sends them: the real speed divided by MoveMultiplier
	// and rounded. Each fly speed is sent twice.
	RunSpd      int32
	WalkSpd     int32
	SwimRunSpd  int32
	SwimWalkSpd int32
	FlyRunSpd   int32
	FlyWalkSpd  int32
	// MoveMultiplier is the movement speed multiplier.
	MoveMultiplier float64
	// AttackSpeedMultiplier has Java float precision, as in UserInfo.
	AttackSpeedMultiplier float32
	CollisionRadius       float64
	CollisionHeight       float64
	HairStyle             int32
	HairColor             int32
	Face                  int32
	Title                 string // the visible title
	ClanID                int32
	ClanCrestID           int32
	AllyID                int32
	AllyCrestID           int32
	Sitting               bool // sent inverted: 1 standing, 0 sitting
	Running               bool
	InCombat              bool
	// Dead is set for a dead or fake-dead player outside the Olympiad.
	Dead             bool
	Invisible        bool
	MountType        uint8 // 0 none, 1 strider, 2 wyvern, 3 great wolf
	PrivateStoreType uint8
	Cubics           []int16 // cubic IDs
	InPartyMatchRoom bool
	AbnormalEffects  int32 // abnormal visual effect mask
	RecomLeft        int16 // sent as one byte here
	RecomHave        int16 // the blue tint of the name
	ClassID          int32
	MaxCP            int32
	CurCP            float64 // sent cut to an int (toward zero)
	// EnchantEffect is the weapon enchant glow level; Mobius sends 0 while
	// mounted.
	EnchantEffect    uint8
	Team             uint8
	ClanCrestLargeID int32
	Noble            bool
	// HeroAura is set for heroes, and for GMs when Mobius GMHeroAura is on.
	HeroAura          bool
	Fishing           bool
	FishX             int32
	FishY             int32
	FishZ             int32
	NameColor         int32
	Heading           int32
	PledgeClass       int32
	PledgeType        int32
	TitleColor        int32
	CursedWeaponLevel int32 // 0 without a cursed weapon
}

// NewCharInfoPacket builds CharInfo (Mobius CharInfo.writeImpl).
func NewCharInfoPacket(c CharInfo) []byte {
	w := packets.NewWriter(400 + 2*(len(c.Name)+len(c.Title)) + 2*len(c.Cubics))
	w.WriteUint8(OpCharInfo)
	w.WriteInt32(c.X)
	w.WriteInt32(c.Y)
	w.WriteInt32(c.Z)
	w.WriteInt32(c.VehicleID)
	w.WriteInt32(c.ObjectID)
	w.WriteString(c.Name)
	w.WriteInt32(c.Race)
	putBool32(w, c.Female)
	w.WriteInt32(c.BaseClassID)
	for _, slot := range charInfoSlots {
		w.WriteInt32(c.Paperdoll[slot].ItemID)
	}

	// The C6 additions: unknown shorts around the weapon augmentation.
	for range 4 {
		w.WriteUint16(0)
	}
	w.WriteInt32(c.WeaponAugmentationID)
	for range 12 {
		w.WriteUint16(0)
	}
	w.WriteInt32(c.WeaponAugmentationID)
	for range 4 {
		w.WriteUint16(0)
	}

	w.WriteInt32(c.PvPFlag)
	w.WriteInt32(c.Karma)
	w.WriteInt32(c.MAtkSpd)
	w.WriteInt32(c.PAtkSpd)
	w.WriteInt32(c.PvPFlag)
	w.WriteInt32(c.Karma)
	w.WriteInt32(c.RunSpd)
	w.WriteInt32(c.WalkSpd)
	w.WriteInt32(c.SwimRunSpd)
	w.WriteInt32(c.SwimWalkSpd)
	w.WriteInt32(c.FlyRunSpd)
	w.WriteInt32(c.FlyWalkSpd)
	w.WriteInt32(c.FlyRunSpd)
	w.WriteInt32(c.FlyWalkSpd)
	w.WriteFloat64(c.MoveMultiplier)
	w.WriteFloat64(float64(c.AttackSpeedMultiplier))
	w.WriteFloat64(c.CollisionRadius)
	w.WriteFloat64(c.CollisionHeight)
	w.WriteInt32(c.HairStyle)
	w.WriteInt32(c.HairColor)
	w.WriteInt32(c.Face)
	w.WriteString(c.Title)
	w.WriteInt32(c.ClanID)
	w.WriteInt32(c.ClanCrestID)
	w.WriteInt32(c.AllyID)
	w.WriteInt32(c.AllyCrestID)
	w.WriteInt32(0) // relation: sent in RelationChanged instead
	putBool8(w, !c.Sitting)
	putBool8(w, c.Running)
	putBool8(w, c.InCombat)
	putBool8(w, c.Dead)
	putBool8(w, c.Invisible)
	w.WriteUint8(c.MountType)
	w.WriteUint8(c.PrivateStoreType)
	putCount16(w, len(c.Cubics))
	for _, id := range c.Cubics {
		putInt16(w, id)
	}
	putBool8(w, c.InPartyMatchRoom)
	w.WriteInt32(c.AbnormalEffects)
	w.WriteUint8(uint8(c.RecomLeft)) //nolint:gosec // Mobius writes the low byte
	putInt16(w, c.RecomHave)
	w.WriteInt32(c.ClassID)
	w.WriteInt32(c.MaxCP)
	w.WriteInt32(truncToInt32(c.CurCP))
	w.WriteUint8(c.EnchantEffect)
	w.WriteUint8(c.Team)
	w.WriteInt32(c.ClanCrestLargeID)
	putBool8(w, c.Noble)
	putBool8(w, c.HeroAura)
	putBool8(w, c.Fishing)
	w.WriteInt32(c.FishX)
	w.WriteInt32(c.FishY)
	w.WriteInt32(c.FishZ)
	w.WriteInt32(c.NameColor)
	w.WriteInt32(c.Heading)
	w.WriteInt32(c.PledgeClass)
	w.WriteInt32(c.PledgeType)
	w.WriteInt32(c.TitleColor)
	w.WriteInt32(c.CursedWeaponLevel)
	return w.Bytes()
}
