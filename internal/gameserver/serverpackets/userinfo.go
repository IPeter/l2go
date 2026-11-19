package serverpackets

import (
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
)

// OpUserInfo is the opcode of UserInfo.
const OpUserInfo = 0x04

// AbnormalStealth is the abnormal visual effect bit of an invisible
// character (Mobius AbnormalVisualEffect.STEALTH). Mobius adds it to the
// effects of UserInfo for an invisible player, and of CharInfo for a GM
// who sees an invisible player.
const AbnormalStealth = 0x00100000

// Relation bits of UserInfo (the player's own siege and clan state). They
// differ from the RelationChanged bits.
const (
	UserRelationLeader   = 0x40  // clan leader rights
	UserRelationDefender = 0x80  // siege defender: shield
	UserRelationAttacker = 0x180 // siege attacker: sword
)

// EquippedItem is the item in one paperdoll slot. Zero values are an empty
// slot.
type EquippedItem struct {
	ObjectID int32
	ItemID   int32 // the display ID of the item template
}

// Paperdoll is a character's equipment, indexed by storage.PaperdollSlot.
// Each packet picks the slots it shows in its own order.
type Paperdoll [storage.PaperdollTotalSlots]EquippedItem

// userInfoSlots is the order of the equipment lists in UserInfo. The right
// hand is listed twice (where older clients had the two-handed slot), and
// the last entry is the face slot, which Mobius Interlude never fills, not
// the second hair slot.
var userInfoSlots = [...]storage.PaperdollSlot{
	storage.PaperdollUnder,
	storage.PaperdollREar,
	storage.PaperdollLEar,
	storage.PaperdollNeck,
	storage.PaperdollRFinger,
	storage.PaperdollLFinger,
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
	storage.PaperdollFace,
}

// UserInfo is everything the client shows about the player's own
// character: the character window, the status bars and the model.
//
// The fields hold the values as Mobius writes them. The builder only does
// the conversions Mobius UserInfo.writeImpl does itself: rounding the
// current HP and MP, cutting the current CP to an int, the weapon flag, and
// the NPC ID offset of the mount.
type UserInfo struct {
	X, Y, Z     int32
	VehicleID   int32 // the boat the player is on, or 0
	ObjectID    int32
	Name        string // the visible name
	Race        int32
	Female      bool
	BaseClassID int32
	Level       int32
	Exp         int64
	STR         int32
	DEX         int32
	CON         int32
	INT         int32
	WIT         int32
	MEN         int32
	MaxHP       int32
	CurHP       float64 // sent rounded to the nearest int (halves up)
	MaxMP       int32
	CurMP       float64 // sent rounded to the nearest int (halves up)
	SP          int32
	CurLoad     int32
	MaxLoad     int32
	// WeaponEquipped is true with a weapon in the right hand; Mobius sends
	// 40 then and 20 without.
	WeaponEquipped bool

	Paperdoll Paperdoll
	// WeaponAugmentationID is the augmentation of the right-hand item, or 0.
	WeaponAugmentationID int32

	// Combat stats as the character window shows them (Mobius casts the
	// doubles to int).
	PAtk     int32
	PAtkSpd  int32 // sent twice: as attack speed and after the casting speed
	PDef     int32
	Evasion  int32
	Accuracy int32
	Critical int32
	MAtk     int32
	MAtkSpd  int32
	MDef     int32
	PvPFlag  int32
	Karma    int32

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
	// AttackSpeedMultiplier is 1.1 × attack speed ÷ base attack speed.
	// Mobius computes it as a Java float, so it has float precision.
	AttackSpeedMultiplier float32
	CollisionRadius       float64
	CollisionHeight       float64
	HairStyle             int32
	HairColor             int32
	Face                  int32
	GM                    bool   // the builder level: 1 for a GM
	Title                 string // Mobius sends "[Invisible]" for an invisible GM

	ClanID      int32
	ClanCrestID int32
	AllyID      int32
	AllyCrestID int32
	// Relation is a combination of the UserRelation bits.
	Relation         int32
	MountType        uint8 // 0 none, 1 strider, 2 wyvern, 3 great wolf
	PrivateStoreType uint8
	DwarvenCraft     bool
	PKKills          int32
	PvPKills         int32
	Cubics           []int16 // cubic IDs
	InPartyMatchRoom bool
	// AbnormalEffects is the abnormal visual effect mask; Mobius adds
	// AbnormalStealth for an invisible player.
	AbnormalEffects int32
	InWater         bool
	ClanPrivileges  int32
	RecomLeft       int16 // recommendations the player can still give
	RecomHave       int16 // recommendations the player received
	// MountNpcID is the NPC ID of the mount, or 0. The builder adds the
	// 1,000,000 offset the client expects.
	MountNpcID     int32
	InventoryLimit int16
	ClassID        int32
	MaxCP          int32
	CurCP          float64 // sent cut to an int (toward zero)
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
	Running           bool // changes the speed the status window shows
	PledgeClass       int32
	PledgeType        int32
	TitleColor        int32
	CursedWeaponLevel int32 // 0 without a cursed weapon
}

// NewUserInfoPacket builds UserInfo (Mobius UserInfo.writeImpl).
func NewUserInfoPacket(u UserInfo) []byte {
	w := packets.NewWriter(600 + 2*(len(u.Name)+len(u.Title)) + 2*len(u.Cubics))
	w.WriteUint8(OpUserInfo)
	w.WriteInt32(u.X)
	w.WriteInt32(u.Y)
	w.WriteInt32(u.Z)
	w.WriteInt32(u.VehicleID)
	w.WriteInt32(u.ObjectID)
	w.WriteString(u.Name)
	w.WriteInt32(u.Race)
	putBool32(w, u.Female)
	w.WriteInt32(u.BaseClassID)
	w.WriteInt32(u.Level)
	putInt64(w, u.Exp)
	w.WriteInt32(u.STR)
	w.WriteInt32(u.DEX)
	w.WriteInt32(u.CON)
	w.WriteInt32(u.INT)
	w.WriteInt32(u.WIT)
	w.WriteInt32(u.MEN)
	w.WriteInt32(u.MaxHP)
	w.WriteInt32(roundHalfUp32(u.CurHP))
	w.WriteInt32(u.MaxMP)
	w.WriteInt32(roundHalfUp32(u.CurMP))
	w.WriteInt32(u.SP)
	w.WriteInt32(u.CurLoad)
	w.WriteInt32(u.MaxLoad)
	if u.WeaponEquipped {
		w.WriteInt32(40)
	} else {
		w.WriteInt32(20)
	}

	for _, slot := range userInfoSlots {
		w.WriteInt32(u.Paperdoll[slot].ObjectID)
	}
	for _, slot := range userInfoSlots {
		w.WriteInt32(u.Paperdoll[slot].ItemID)
	}

	// The C6 additions: unknown shorts around the weapon augmentation.
	for range 14 {
		w.WriteUint16(0)
	}
	w.WriteInt32(u.WeaponAugmentationID)
	for range 12 {
		w.WriteUint16(0)
	}
	w.WriteInt32(u.WeaponAugmentationID)
	for range 4 {
		w.WriteUint16(0)
	}

	w.WriteInt32(u.PAtk)
	w.WriteInt32(u.PAtkSpd)
	w.WriteInt32(u.PDef)
	w.WriteInt32(u.Evasion)
	w.WriteInt32(u.Accuracy)
	w.WriteInt32(u.Critical)
	w.WriteInt32(u.MAtk)
	w.WriteInt32(u.MAtkSpd)
	w.WriteInt32(u.PAtkSpd)
	w.WriteInt32(u.MDef)
	w.WriteInt32(u.PvPFlag)
	w.WriteInt32(u.Karma)
	w.WriteInt32(u.RunSpd)
	w.WriteInt32(u.WalkSpd)
	w.WriteInt32(u.SwimRunSpd)
	w.WriteInt32(u.SwimWalkSpd)
	w.WriteInt32(u.FlyRunSpd)
	w.WriteInt32(u.FlyWalkSpd)
	w.WriteInt32(u.FlyRunSpd)
	w.WriteInt32(u.FlyWalkSpd)
	w.WriteFloat64(u.MoveMultiplier)
	w.WriteFloat64(float64(u.AttackSpeedMultiplier))
	w.WriteFloat64(u.CollisionRadius)
	w.WriteFloat64(u.CollisionHeight)
	w.WriteInt32(u.HairStyle)
	w.WriteInt32(u.HairColor)
	w.WriteInt32(u.Face)
	putBool32(w, u.GM)
	w.WriteString(u.Title)

	w.WriteInt32(u.ClanID)
	w.WriteInt32(u.ClanCrestID)
	w.WriteInt32(u.AllyID)
	w.WriteInt32(u.AllyCrestID)
	w.WriteInt32(u.Relation)
	w.WriteUint8(u.MountType)
	w.WriteUint8(u.PrivateStoreType)
	putBool8(w, u.DwarvenCraft)
	w.WriteInt32(u.PKKills)
	w.WriteInt32(u.PvPKills)
	putCount16(w, len(u.Cubics))
	for _, id := range u.Cubics {
		putInt16(w, id)
	}
	putBool8(w, u.InPartyMatchRoom)
	w.WriteInt32(u.AbnormalEffects)
	putBool8(w, u.InWater)
	w.WriteInt32(u.ClanPrivileges)
	putInt16(w, u.RecomLeft)
	putInt16(w, u.RecomHave)
	if u.MountNpcID > 0 {
		w.WriteInt32(u.MountNpcID + clientNpcIDBase)
	} else {
		w.WriteInt32(0)
	}
	putInt16(w, u.InventoryLimit)
	w.WriteInt32(u.ClassID)
	w.WriteInt32(0) // special effects, always 0
	w.WriteInt32(u.MaxCP)
	w.WriteInt32(truncToInt32(u.CurCP))
	w.WriteUint8(u.EnchantEffect)
	w.WriteUint8(u.Team)
	w.WriteInt32(u.ClanCrestLargeID)
	putBool8(w, u.Noble)
	putBool8(w, u.HeroAura)
	putBool8(w, u.Fishing)
	w.WriteInt32(u.FishX)
	w.WriteInt32(u.FishY)
	w.WriteInt32(u.FishZ)
	w.WriteInt32(u.NameColor)
	putBool8(w, u.Running)
	w.WriteInt32(u.PledgeClass)
	w.WriteInt32(u.PledgeType)
	w.WriteInt32(u.TitleColor)
	w.WriteInt32(u.CursedWeaponLevel)
	return w.Bytes()
}
