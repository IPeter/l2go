package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The packets of pets, servitors and mounts (M11f): the owner's pet window
// (PetInfo, PetStatusShow, PetStatusUpdate, PetDelete, SetSummonRemainTime),
// the pet's inventory (PetItemList, PetInventoryUpdate), what other players
// see of a summon (SummonInfo, sent with the NpcInfo opcode) and Ride.

// Opcodes of the summon packets (Mobius ServerPackets).
const (
	OpRide                = 0x86
	OpPetStatusShow       = 0xb0
	OpPetInfo             = 0xb1
	OpPetItemList         = 0xb2
	OpPetInventoryUpdate  = 0xb3
	OpPetStatusUpdate     = 0xb5
	OpPetDelete           = 0xb6
	OpSetSummonRemainTime = 0xd1
)

// SummonType is what Mobius Summon.getSummonType returns.
type SummonType int32

// The summon types.
const (
	SummonTypeServitor SummonType = 1 // Mobius Servitor.getSummonType
	SummonTypePet      SummonType = 2 // Mobius Pet.getSummonType
)

// Values of the PetInfo and SummonInfo appearance byte (Mobius _value). The
// builders send SummonAppearanceSummoned instead whenever Summoned is set.
const (
	SummonAppearanceTeleported uint8 = 0 // the summon appears without animation
	SummonAppearanceDefault    uint8 = 1 // an update of a summon already shown
	SummonAppearanceSummoned   uint8 = 2 // play the summon animation
)

// Mount types of Ride (the Mobius MountType ordinals).
const (
	MountNone    = 0
	MountStrider = 1
	MountWyvern  = 2
)

// summonEnvironment returns the environment byte of PetInfo and SummonInfo:
// Mobius checks water first.
func summonEnvironment(inWater, flying bool) uint8 {
	switch {
	case inWater:
		return npcEnvironmentSwim
	case flying:
		return npcEnvironmentFly
	}
	return npcEnvironmentLand
}

// summonAppearance returns the appearance byte: 2 while the summon shows its
// summon animation (Mobius isShowSummonAnimation), else the value given.
func summonAppearance(summoned bool, value uint8) uint8 {
	if summoned {
		return SummonAppearanceSummoned
	}
	return value
}

// PetInfo is the owner's full view of a pet or servitor: its model, its
// status window values and its stats.
//
// The fields hold the values as Mobius writes them. Mobius always writes 0
// for the attackable flag and the left hand, so PetInfo has no fields for
// them.
type PetInfo struct {
	Type     SummonType
	ObjectID int32
	// DisplayID is the template's display ID. The builder adds the 1,000,000
	// offset the client expects.
	DisplayID int32
	X, Y, Z   int32
	Heading   int32
	MAtkSpd   int32 // casting speed; also sent as the casting speed stat
	PAtkSpd   int32 // attack speed, cut to an int; also sent as the stat
	// Speeds as Mobius sends them: the real speed divided by MoveMultiplier
	// and rounded. The fly speeds are the run and walk speeds while flying,
	// else 0; each is sent twice.
	RunSpd      int32
	WalkSpd     int32
	SwimRunSpd  int32
	SwimWalkSpd int32
	FlyRunSpd   int32
	FlyWalkSpd  int32
	// MoveMultiplier is the movement speed multiplier (for a pet, halved
	// while it is uncontrollable).
	MoveMultiplier float64
	// AttackSpeedMultiplier is a Java float in Mobius.
	AttackSpeedMultiplier float32
	// CollisionRadius and CollisionHeight are the template's collision size.
	CollisionRadius float32
	CollisionHeight float32
	// Weapon and Armor are the item IDs of the summon's weapon and armor
	// (Mobius getWeapon, getArmor), 0 for none.
	Weapon int32
	Armor  int32
	// ShowName is Mobius getOwner() != null: when a dead pet's owner has
	// left, the client hides the owner's name.
	ShowName bool
	Running  bool
	InCombat bool
	// Dead is set for a dead or fake-dead summon (Mobius isAlikeDead).
	Dead bool
	// Summoned plays the summon animation (Mobius isShowSummonAnimation);
	// the builder then sends SummonAppearanceSummoned instead of Appearance.
	Summoned   bool
	Appearance uint8
	// Name is the pet's name; for a servitor, its name only for templates
	// with usingServerSideName, else "".
	Name string
	// Title is the owner's name (Mobius Summon.getTitle).
	Title   string
	PvPFlag int32
	Karma   int32
	// CurFed and MaxFed are the pet's food meter. For a servitor Mobius
	// sends the life time remaining and the life time instead.
	CurFed int32
	MaxFed int32
	CurHP  int32 // cut to an int
	MaxHP  int32
	CurMP  int32 // cut to an int
	MaxMP  int32
	SP     int32
	Level  int32
	Exp    int64
	// ExpThisLevel is the experience at the start of Level. The builder
	// sends Exp instead when ExpThisLevel is greater, as Mobius does.
	ExpThisLevel int64
	ExpNextLevel int64
	// CurLoad is the weight of the pet's inventory; Mobius sends 0 for a
	// servitor.
	CurLoad  int32
	MaxLoad  int32
	PAtk     int32
	PDef     int32
	MAtk     int32
	MDef     int32
	Accuracy int32
	Evasion  int32
	Critical int32
	// Speed is the current movement speed, cut to an int (Mobius
	// getMoveSpeed: the run or walk speed, without the multiplier division).
	Speed           int32
	AbnormalEffects int32
	// Mountable shows the ride button (striders and wyverns).
	Mountable bool
	// InWater and Flying choose the movement environment; water wins.
	InWater bool
	Flying  bool
	// Team is the team circle: 0 none, 1 blue, 2 red.
	Team uint8
	// SoulShotsPerHit and SpiritShotsPerHit are how many shots the summon
	// uses per hit (Java shorts sent as ints).
	SoulShotsPerHit   int32
	SpiritShotsPerHit int32
}

// NewPetInfoPacket builds PetInfo (Mobius PetInfo.writeImpl).
func NewPetInfoPacket(p PetInfo) []byte {
	expThisLevel := p.ExpThisLevel
	if expThisLevel > p.Exp {
		expThisLevel = p.Exp
	}

	w := packets.NewWriter(260 + 2*(len(p.Name)+len(p.Title)))
	w.WriteUint8(OpPetInfo)
	w.WriteInt32(int32(p.Type))
	w.WriteInt32(p.ObjectID)
	w.WriteInt32(p.DisplayID + clientNpcIDBase)
	w.WriteInt32(0) // attackable: always 0
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteInt32(p.Heading)
	w.WriteInt32(0)
	w.WriteInt32(p.MAtkSpd)
	w.WriteInt32(p.PAtkSpd)
	w.WriteInt32(p.RunSpd)
	w.WriteInt32(p.WalkSpd)
	w.WriteInt32(p.SwimRunSpd)
	w.WriteInt32(p.SwimWalkSpd)
	w.WriteInt32(p.FlyRunSpd)
	w.WriteInt32(p.FlyWalkSpd)
	w.WriteInt32(p.FlyRunSpd)
	w.WriteInt32(p.FlyWalkSpd)
	w.WriteFloat64(p.MoveMultiplier)
	w.WriteFloat64(float64(p.AttackSpeedMultiplier))
	w.WriteFloat64(float64(p.CollisionRadius))
	w.WriteFloat64(float64(p.CollisionHeight))
	w.WriteInt32(p.Weapon)
	w.WriteInt32(p.Armor)
	w.WriteInt32(0) // left hand
	putBool8(w, p.ShowName)
	putBool8(w, p.Running)
	putBool8(w, p.InCombat)
	putBool8(w, p.Dead)
	w.WriteUint8(summonAppearance(p.Summoned, p.Appearance))
	w.WriteString(p.Name)
	w.WriteString(p.Title)
	w.WriteInt32(1)
	w.WriteInt32(p.PvPFlag)
	w.WriteInt32(p.Karma)
	w.WriteInt32(p.CurFed)
	w.WriteInt32(p.MaxFed)
	w.WriteInt32(p.CurHP)
	w.WriteInt32(p.MaxHP)
	w.WriteInt32(p.CurMP)
	w.WriteInt32(p.MaxMP)
	w.WriteInt32(p.SP)
	w.WriteInt32(p.Level)
	putInt64(w, p.Exp)
	putInt64(w, expThisLevel)
	putInt64(w, p.ExpNextLevel)
	w.WriteInt32(p.CurLoad)
	w.WriteInt32(p.MaxLoad)
	w.WriteInt32(p.PAtk)
	w.WriteInt32(p.PDef)
	w.WriteInt32(p.MAtk)
	w.WriteInt32(p.MDef)
	w.WriteInt32(p.Accuracy)
	w.WriteInt32(p.Evasion)
	w.WriteInt32(p.Critical)
	w.WriteInt32(p.Speed)
	w.WriteInt32(p.PAtkSpd)
	w.WriteInt32(p.MAtkSpd)
	w.WriteInt32(p.AbnormalEffects)
	putBool16(w, p.Mountable)
	w.WriteUint8(summonEnvironment(p.InWater, p.Flying))
	w.WriteUint16(0)
	w.WriteUint8(p.Team)
	w.WriteInt32(p.SoulShotsPerHit)
	w.WriteInt32(p.SpiritShotsPerHit)
	return w.Bytes()
}

// SummonInfo is a summon as the players around it see it (Mobius
// AbstractNpcInfo.SummonInfo). It has the NpcInfo opcode and layout; the
// owner gets PetInfo instead.
type SummonInfo struct {
	ObjectID int32
	// DisplayID is the template's display ID. The builder adds the 1,000,000
	// offset.
	DisplayID int32
	// Attackable is whether the viewer can attack the summon (Mobius
	// isAutoAttackable).
	Attackable bool
	X, Y, Z    int32
	Heading    int32
	MAtkSpd    int32
	PAtkSpd    int32
	// Speeds as in PetInfo.
	RunSpd                int32
	WalkSpd               int32
	SwimRunSpd            int32
	SwimWalkSpd           int32
	FlyRunSpd             int32
	FlyWalkSpd            int32
	MoveMultiplier        float64
	AttackSpeedMultiplier float32
	CollisionRadius       float32
	CollisionHeight       float32
	// Weapon is sent as the right hand, Armor as the chest.
	Weapon   int32
	Armor    int32
	InCombat bool
	Dead     bool
	// Summoned plays the summon animation; the builder then sends
	// SummonAppearanceSummoned instead of Appearance.
	Summoned   bool
	Appearance uint8
	// Name is the summon's name (Mobius sends it for every summon).
	Name string
	// Title is the owner's name while the owner is online, else "".
	Title   string
	PvPFlag int32
	Karma   int32
	// AbnormalEffects is the abnormal visual effect mask. The builder adds
	// AbnormalStealth when the summon is Invisible and the viewer a GM.
	AbnormalEffects int32
	Invisible       bool
	ViewerGM        bool
	InWater         bool
	Flying          bool
	Team            uint8
	// EnchantEffect is the template's weapon enchant.
	EnchantEffect int32
}

// NewSummonInfoPacket builds SummonInfo (Mobius
// AbstractNpcInfo.SummonInfo.writeImpl).
func NewSummonInfoPacket(s SummonInfo) []byte {
	effects := s.AbnormalEffects
	if s.ViewerGM && s.Invisible {
		effects |= AbnormalStealth
	}

	w := packets.NewWriter(184 + 2*(len(s.Name)+len(s.Title)))
	w.WriteUint8(OpNpcInfo)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(s.DisplayID + clientNpcIDBase)
	putBool32(w, s.Attackable)
	w.WriteInt32(s.X)
	w.WriteInt32(s.Y)
	w.WriteInt32(s.Z)
	w.WriteInt32(s.Heading)
	w.WriteInt32(0)
	w.WriteInt32(s.MAtkSpd)
	w.WriteInt32(s.PAtkSpd)
	w.WriteInt32(s.RunSpd)
	w.WriteInt32(s.WalkSpd)
	w.WriteInt32(s.SwimRunSpd)
	w.WriteInt32(s.SwimWalkSpd)
	w.WriteInt32(s.FlyRunSpd)
	w.WriteInt32(s.FlyWalkSpd)
	w.WriteInt32(s.FlyRunSpd)
	w.WriteInt32(s.FlyWalkSpd)
	w.WriteFloat64(s.MoveMultiplier)
	w.WriteFloat64(float64(s.AttackSpeedMultiplier))
	w.WriteFloat64(float64(s.CollisionRadius))
	w.WriteFloat64(float64(s.CollisionHeight))
	w.WriteInt32(s.Weapon)
	w.WriteInt32(s.Armor)
	w.WriteInt32(0) // left hand
	w.WriteUint8(1) // show the name
	w.WriteUint8(1) // always running
	putBool8(w, s.InCombat)
	putBool8(w, s.Dead)
	w.WriteUint8(summonAppearance(s.Summoned, s.Appearance))
	w.WriteString(s.Name)
	w.WriteString(s.Title)
	w.WriteInt32(1) // title colour
	w.WriteInt32(s.PvPFlag)
	w.WriteInt32(s.Karma)
	w.WriteInt32(effects)
	w.WriteInt32(0) // clan ID
	w.WriteInt32(0) // clan crest ID
	w.WriteInt32(0) // alliance ID
	w.WriteInt32(0) // alliance crest ID
	w.WriteUint8(summonEnvironment(s.InWater, s.Flying))
	w.WriteUint8(s.Team)
	w.WriteFloat64(float64(s.CollisionRadius))
	w.WriteFloat64(float64(s.CollisionHeight))
	w.WriteInt32(s.EnchantEffect)
	w.WriteInt32(0) // flying: Mobius sends 0 for summons
	return w.Bytes()
}

// petItemSize is the size of one PetItemList item block.
const petItemSize = 28

// putPetItem writes one item of the pet inventory packets (Mobius
// PetItemList and PetInventoryUpdate). It is shorter than the player's item
// block: the custom types are sent as 0 and there is no augmentation or
// mana, so those fields of it are ignored.
func putPetItem(w *packets.Writer, it ItemInfo) {
	putInt16(w, it.Type1)
	w.WriteInt32(it.ObjectID)
	w.WriteInt32(it.ItemID)
	w.WriteInt32(it.Count)
	putInt16(w, it.Type2)
	w.WriteUint16(0)
	putBool16(w, it.Equipped)
	w.WriteInt32(it.BodyPart)
	putInt16(w, it.Enchant)
	w.WriteUint16(0)
}

// PetItemList is the pet's whole inventory. Only Type1, ObjectID, ItemID,
// Count, Type2, Equipped, BodyPart and Enchant of each item are sent.
type PetItemList struct {
	Items []ItemInfo
}

// NewPetItemListPacket builds PetItemList (Mobius PetItemList.writeImpl).
func NewPetItemListPacket(l PetItemList) []byte {
	w := packets.NewWriter(3 + len(l.Items)*petItemSize)
	w.WriteUint8(OpPetItemList)
	putCount16(w, len(l.Items))
	for _, it := range l.Items {
		putPetItem(w, it)
	}
	return w.Bytes()
}

// PetInventoryUpdate changes some items of the pet's inventory. The items
// are sent as in PetItemList.
type PetInventoryUpdate struct {
	Items []InventoryItem
}

// NewPetInventoryUpdatePacket builds PetInventoryUpdate (Mobius
// PetInventoryUpdate.writeImpl).
func NewPetInventoryUpdatePacket(u PetInventoryUpdate) []byte {
	w := packets.NewWriter(3 + len(u.Items)*(2+petItemSize))
	w.WriteUint8(OpPetInventoryUpdate)
	putCount16(w, len(u.Items))
	for _, it := range u.Items {
		putInt16(w, int16(it.Change))
		putPetItem(w, it.Item)
	}
	return w.Bytes()
}

// PetStatusShow opens the owner's pet or servitor window.
type PetStatusShow struct {
	Type SummonType
}

// NewPetStatusShowPacket builds PetStatusShow (Mobius
// PetStatusShow.writeImpl).
func NewPetStatusShowPacket(p PetStatusShow) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpPetStatusShow)
	w.WriteInt32(int32(p.Type))
	return w.Bytes()
}

// PetStatusUpdate refreshes the bars of the owner's pet window.
type PetStatusUpdate struct {
	Type     SummonType
	ObjectID int32
	X, Y, Z  int32
	// Title is the owner's name (Mobius Summon.getTitle).
	Title string
	// CurFed and MaxFed as in PetInfo: a servitor sends its life time.
	CurFed int32
	MaxFed int32
	CurHP  int32 // cut to an int
	MaxHP  int32
	CurMP  int32 // cut to an int
	MaxMP  int32
	Level  int32
	Exp    int64
	// ExpThisLevel is sent as it is: unlike PetInfo, Mobius does not cap
	// it at Exp here.
	ExpThisLevel int64
	ExpNextLevel int64
}

// NewPetStatusUpdatePacket builds PetStatusUpdate (Mobius
// PetStatusUpdate.writeImpl).
func NewPetStatusUpdatePacket(p PetStatusUpdate) []byte {
	w := packets.NewWriter(83 + 2*len(p.Title))
	w.WriteUint8(OpPetStatusUpdate)
	w.WriteInt32(int32(p.Type))
	w.WriteInt32(p.ObjectID)
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteString(p.Title)
	w.WriteInt32(p.CurFed)
	w.WriteInt32(p.MaxFed)
	w.WriteInt32(p.CurHP)
	w.WriteInt32(p.MaxHP)
	w.WriteInt32(p.CurMP)
	w.WriteInt32(p.MaxMP)
	w.WriteInt32(p.Level)
	putInt64(w, p.Exp)
	putInt64(w, p.ExpThisLevel)
	putInt64(w, p.ExpNextLevel)
	return w.Bytes()
}

// PetDelete closes the owner's pet window when the summon goes away.
type PetDelete struct {
	Type     SummonType
	ObjectID int32
}

// NewPetDeletePacket builds PetDelete (Mobius PetDelete.writeImpl).
func NewPetDeletePacket(p PetDelete) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpPetDelete)
	w.WriteInt32(int32(p.Type))
	w.WriteInt32(p.ObjectID)
	return w.Bytes()
}

// SetSummonRemainTime sets the servitor's life time bar.
type SetSummonRemainTime struct {
	MaxTime       int32
	RemainingTime int32
}

// NewSetSummonRemainTimePacket builds SetSummonRemainTime (Mobius
// SetSummonRemainTime.writeImpl): the maximum first.
func NewSetSummonRemainTimePacket(s SetSummonRemainTime) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpSetSummonRemainTime)
	w.WriteInt32(s.MaxTime)
	w.WriteInt32(s.RemainingTime)
	return w.Bytes()
}

// Ride mounts a player on a strider or wyvern, or dismounts them.
type Ride struct {
	ObjectID int32
	Mounted  bool
	// MountType is MountNone, MountStrider or MountWyvern.
	MountType int32
	// MountNpcID is the NPC ID of the mount, 0 when dismounting. The builder
	// adds the 1,000,000 offset even to 0, as Mobius does.
	MountNpcID int32
	X, Y, Z    int32
}

// NewRidePacket builds Ride (Mobius Ride.writeImpl).
func NewRidePacket(r Ride) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpRide)
	w.WriteInt32(r.ObjectID)
	putBool32(w, r.Mounted)
	w.WriteInt32(r.MountType)
	w.WriteInt32(r.MountNpcID + clientNpcIDBase)
	w.WriteInt32(r.X)
	w.WriteInt32(r.Y)
	w.WriteInt32(r.Z)
	return w.Bytes()
}
