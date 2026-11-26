package testclient

import (
	"context"
	"slices"
)

// The pet, servitor and mount packets (M11f), in the Mobius Interlude
// layouts. Every reader follows the writeImpl of the Mobius class of the
// same name, field by field. What other players see of a summon is NpcInfo
// (Mobius AbstractNpcInfo.SummonInfo has the NpcInfo layout): its
// TitleColor is 1, Running always true, and Summoned 0, 1 or 2.

// Game server opcodes of the summon packets (Mobius ServerPackets).
const (
	OpRide                byte = 0x86
	OpPetStatusShow       byte = 0xb0
	OpPetInfo             byte = 0xb1
	OpPetItemList         byte = 0xb2
	OpPetInventoryUpdate  byte = 0xb3
	OpPetStatusUpdate     byte = 0xb5
	OpPetDelete           byte = 0xb6
	OpSetSummonRemainTime byte = 0xd1
)

// Game client opcodes of the pet requests (Mobius ClientPackets).
const (
	OpRequestChangePetName  byte = 0x89
	OpRequestPetUseItem     byte = 0x8a
	OpRequestGiveItemToPet  byte = 0x8b
	OpRequestGetItemFromPet byte = 0x8c
	OpRequestPetGetItem     byte = 0x8f
)

// Summon types of PetInfo, PetStatusShow, PetStatusUpdate and PetDelete
// (Mobius getSummonType).
const (
	SummonServitor int32 = 1
	SummonPet      int32 = 2
)

// Mount types of Ride (Mobius MountType ordinals).
const (
	MountNone    int32 = 0
	MountStrider int32 = 1
	MountWyvern  int32 = 2
)

// Summon action IDs of RequestActionUse (Mobius RequestActionUse).
const (
	ActionPetFollow            int32 = 15
	ActionPetAttack            int32 = 16
	ActionPetStop              int32 = 17
	ActionUnsummonPet          int32 = 19
	ActionServitorFollow       int32 = 21
	ActionServitorAttack       int32 = 22
	ActionServitorStop         int32 = 23
	ActionMountDismount        int32 = 38
	ActionUnsummonServitor     int32 = 52
	ActionServitorMoveToTarget int32 = 53
	ActionPetMoveToTarget      int32 = 54
)

// The decoder and recorder entries are added here rather than in the shared
// tables, so the summon packets stay in one file.
func init() {
	for op, p := range map[byte]serverPacket{
		OpRide:                {"Ride", readRide},
		OpPetStatusShow:       {"PetStatusShow", readPetStatusShow},
		OpPetInfo:             {"PetInfo", readPetInfo},
		OpPetItemList:         {"PetItemList", readPetItemList},
		OpPetInventoryUpdate:  {"PetInventoryUpdate", readPetInventoryUpdate},
		OpPetStatusUpdate:     {"PetStatusUpdate", readPetStatusUpdate},
		OpPetDelete:           {"PetDelete", readPetDelete},
		OpSetSummonRemainTime: {"SetSummonRemainTime", readSetSummonRemainTime},
	} {
		serverPackets[op] = p
	}
	for op, name := range map[byte]string{
		OpRequestChangePetName:  "RequestChangePetName",
		OpRequestPetUseItem:     "RequestPetUseItem",
		OpRequestGiveItemToPet:  "RequestGiveItemToPet",
		OpRequestGetItemFromPet: "RequestGetItemFromPet",
		OpRequestPetGetItem:     "RequestPetGetItem",
	} {
		clientPacketNames[op] = name
	}
}

// PetInfo is the owner's view of a pet or servitor (0xb1): its model, its
// status window and its stats.
type PetInfo struct {
	// SummonType is SummonServitor or SummonPet.
	SummonType int32
	ObjectID   int32
	// NpcTypeID is the template's display ID plus 1,000,000.
	NpcTypeID int32
	// Attackable is always false in Mobius.
	Attackable bool
	X, Y, Z    int32
	Heading    int32
	// Unknown is a word Mobius writes as 0.
	Unknown int32
	MAtkSpd int32
	PAtkSpd int32

	RunSpd, WalkSpd         int32
	SwimRunSpd, SwimWalkSpd int32
	FlyRunSpd, FlyWalkSpd   int32
	FlyRunSpd2, FlyWalkSpd2 int32

	MoveMultiplier        float64
	AttackSpeedMultiplier float64
	CollisionRadius       float64
	CollisionHeight       float64

	// RHand is the summon's weapon, Chest its armor; LHand is 0.
	RHand int32
	Chest int32
	LHand int32
	// ShowName is false when the owner of a dead pet has left.
	ShowName  bool
	Running   bool
	InCombat  bool
	AlikeDead bool
	// Summoned is 0 (teleported), 1 (default) or 2 (summon animation).
	Summoned uint8
	Name     string
	// Title is the owner's name.
	Title string
	// TitleColor is 1 in Mobius.
	TitleColor int32
	PvpFlag    int32
	Karma      int32
	// CurFed and MaxFed are the food meter of a pet, the remaining and the
	// whole life time of a servitor.
	CurFed int32
	MaxFed int32
	CurHP  int32
	MaxHP  int32
	CurMP  int32
	MaxMP  int32
	SP     int32
	Level  int32
	Exp    int64
	// ExpThisLevel is the 0% mark of the experience bar (at most Exp),
	// ExpNextLevel the 100% mark.
	ExpThisLevel int64
	ExpNextLevel int64
	// CurLoad is 0 for a servitor.
	CurLoad  int32
	MaxLoad  int32
	PAtk     int32
	PDef     int32
	MAtk     int32
	MDef     int32
	Accuracy int32
	Evasion  int32
	Critical int32
	// Speed is the current movement speed.
	Speed           int32
	AtkSpd          int32
	CastSpd         int32
	AbnormalEffects int32
	// Mountable shows the ride button.
	Mountable bool
	// Environment is 1 in water, 2 flying, else 0.
	Environment uint8
	// Unknown2 is a short Mobius writes as 0.
	Unknown2          uint16
	Team              uint8
	SoulShotsPerHit   int32
	SpiritShotsPerHit int32
}

// PacketName implements Packet.
func (PetInfo) PacketName() string { return "PetInfo" }

// NpcID returns the template's display ID.
func (p PetInfo) NpcID() int32 { return p.NpcTypeID - npcTypeOffset }

// Position returns where the summon stands.
func (p PetInfo) Position() Position { return Position{p.X, p.Y, p.Z} }

// IsPet reports whether the summon is a pet rather than a servitor.
func (p PetInfo) IsPet() bool { return p.SummonType == SummonPet }

func readPetInfo(f *fields) Packet {
	var p PetInfo
	p.SummonType = f.i32()
	p.ObjectID = f.i32()
	p.NpcTypeID = f.i32()
	p.Attackable = f.flag32("attackable flag")
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Heading = f.i32()
	p.Unknown = f.i32()
	p.MAtkSpd = f.i32()
	p.PAtkSpd = f.i32()
	p.RunSpd = f.i32()
	p.WalkSpd = f.i32()
	p.SwimRunSpd = f.i32()
	p.SwimWalkSpd = f.i32()
	p.FlyRunSpd = f.i32()
	p.FlyWalkSpd = f.i32()
	p.FlyRunSpd2 = f.i32()
	p.FlyWalkSpd2 = f.i32()
	p.MoveMultiplier = f.f64()
	p.AttackSpeedMultiplier = f.f64()
	p.CollisionRadius = f.f64()
	p.CollisionHeight = f.f64()
	p.RHand = f.i32()
	p.Chest = f.i32()
	p.LHand = f.i32()
	p.ShowName = f.flag8("show name flag")
	p.Running = f.flag8("running flag")
	p.InCombat = f.flag8("combat flag")
	p.AlikeDead = f.flag8("dead flag")
	p.Summoned = f.u8()
	p.Name = f.str()
	p.Title = f.str()
	p.TitleColor = f.i32()
	p.PvpFlag = f.i32()
	p.Karma = f.i32()
	p.CurFed = f.i32()
	p.MaxFed = f.i32()
	p.CurHP = f.i32()
	p.MaxHP = f.i32()
	p.CurMP = f.i32()
	p.MaxMP = f.i32()
	p.SP = f.i32()
	p.Level = f.i32()
	p.Exp = f.i64()
	p.ExpThisLevel = f.i64()
	p.ExpNextLevel = f.i64()
	p.CurLoad = f.i32()
	p.MaxLoad = f.i32()
	p.PAtk = f.i32()
	p.PDef = f.i32()
	p.MAtk = f.i32()
	p.MDef = f.i32()
	p.Accuracy = f.i32()
	p.Evasion = f.i32()
	p.Critical = f.i32()
	p.Speed = f.i32()
	p.AtkSpd = f.i32()
	p.CastSpd = f.i32()
	p.AbnormalEffects = f.i32()
	p.Mountable = f.flag16("mountable flag")
	p.Environment = f.u8()
	p.Unknown2 = f.u16()
	p.Team = f.u8()
	p.SoulShotsPerHit = f.i32()
	p.SpiritShotsPerHit = f.i32()
	return p
}

// PetItem is one item of PetItemList and PetInventoryUpdate. The pet
// packets have no custom types, augmentation or mana.
type PetItem struct {
	Type1    uint16
	ObjectID int32
	ItemID   int32
	Count    int32
	Type2    uint16
	// Reserved1 and Reserved2 are shorts Mobius writes as 0.
	Reserved1 uint16
	Equipped  bool
	// BodyPart is the slot mask of the item.
	BodyPart  int32
	Enchant   uint16
	Reserved2 uint16
}

func readPetItem(f *fields) PetItem {
	var it PetItem
	it.Type1 = f.u16()
	it.ObjectID = f.i32()
	it.ItemID = f.i32()
	it.Count = f.i32()
	it.Type2 = f.u16()
	it.Reserved1 = f.u16()
	it.Equipped = f.flag16("equipped flag")
	it.BodyPart = f.i32()
	it.Enchant = f.u16()
	it.Reserved2 = f.u16()
	return it
}

// PetItemList is the pet's whole inventory (0xb2).
type PetItemList struct {
	Items []PetItem
}

// PacketName implements Packet.
func (PetItemList) PacketName() string { return "PetItemList" }

// Find returns the item objectID.
func (l PetItemList) Find(objectID int32) (PetItem, bool) {
	i := slices.IndexFunc(l.Items, func(it PetItem) bool { return it.ObjectID == objectID })
	if i < 0 {
		return PetItem{}, false
	}
	return l.Items[i], true
}

// FindItem returns the first item with template itemID.
func (l PetItemList) FindItem(itemID int32) (PetItem, bool) {
	i := slices.IndexFunc(l.Items, func(it PetItem) bool { return it.ItemID == itemID })
	if i < 0 {
		return PetItem{}, false
	}
	return l.Items[i], true
}

func readPetItemList(f *fields) Packet {
	var p PetItemList
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		p.Items = append(p.Items, readPetItem(f))
	}
	return p
}

// PetItemChange is one entry of PetInventoryUpdate.
type PetItemChange struct {
	// Change is ItemAdded, ItemModified or ItemRemoved.
	Change uint16
	PetItem
}

// PetInventoryUpdate changes some items of the pet's inventory (0xb3).
type PetInventoryUpdate struct {
	Items []PetItemChange
}

// PacketName implements Packet.
func (PetInventoryUpdate) PacketName() string { return "PetInventoryUpdate" }

// Find returns the entry of the item objectID.
func (u PetInventoryUpdate) Find(objectID int32) (PetItemChange, bool) {
	i := slices.IndexFunc(u.Items, func(it PetItemChange) bool { return it.ObjectID == objectID })
	if i < 0 {
		return PetItemChange{}, false
	}
	return u.Items[i], true
}

func readPetInventoryUpdate(f *fields) Packet {
	var p PetInventoryUpdate
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		change := f.u16()
		p.Items = append(p.Items, PetItemChange{Change: change, PetItem: readPetItem(f)})
	}
	return p
}

// PetStatusShow opens the owner's summon window (0xb0).
type PetStatusShow struct {
	SummonType int32
}

// PacketName implements Packet.
func (PetStatusShow) PacketName() string { return "PetStatusShow" }

func readPetStatusShow(f *fields) Packet { return PetStatusShow{SummonType: f.i32()} }

// PetStatusUpdate refreshes the owner's summon window (0xb5).
type PetStatusUpdate struct {
	SummonType int32
	ObjectID   int32
	X, Y, Z    int32
	// Title is the owner's name.
	Title  string
	CurFed int32
	MaxFed int32
	CurHP  int32
	MaxHP  int32
	CurMP  int32
	MaxMP  int32
	Level  int32
	Exp    int64
	// ExpThisLevel is not capped at Exp here, unlike PetInfo.
	ExpThisLevel int64
	ExpNextLevel int64
}

// PacketName implements Packet.
func (PetStatusUpdate) PacketName() string { return "PetStatusUpdate" }

// Position returns where the summon stands.
func (p PetStatusUpdate) Position() Position { return Position{p.X, p.Y, p.Z} }

func readPetStatusUpdate(f *fields) Packet {
	var p PetStatusUpdate
	p.SummonType = f.i32()
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Title = f.str()
	p.CurFed = f.i32()
	p.MaxFed = f.i32()
	p.CurHP = f.i32()
	p.MaxHP = f.i32()
	p.CurMP = f.i32()
	p.MaxMP = f.i32()
	p.Level = f.i32()
	p.Exp = f.i64()
	p.ExpThisLevel = f.i64()
	p.ExpNextLevel = f.i64()
	return p
}

// PetDelete closes the owner's summon window (0xb6).
type PetDelete struct {
	SummonType int32
	ObjectID   int32
}

// PacketName implements Packet.
func (PetDelete) PacketName() string { return "PetDelete" }

func readPetDelete(f *fields) Packet {
	return PetDelete{SummonType: f.i32(), ObjectID: f.i32()}
}

// SetSummonRemainTime sets the servitor's life time bar (0xd1).
type SetSummonRemainTime struct {
	MaxTime       int32
	RemainingTime int32
}

// PacketName implements Packet.
func (SetSummonRemainTime) PacketName() string { return "SetSummonRemainTime" }

func readSetSummonRemainTime(f *fields) Packet {
	return SetSummonRemainTime{MaxTime: f.i32(), RemainingTime: f.i32()}
}

// Ride mounts or dismounts a player (0x86).
type Ride struct {
	ObjectID int32
	Mounted  bool
	// MountType is MountNone, MountStrider or MountWyvern.
	MountType int32
	// MountNpcTypeID is the mount's NPC ID plus 1,000,000; Mobius sends
	// 1,000,000 when dismounting.
	MountNpcTypeID int32
	X, Y, Z        int32
}

// PacketName implements Packet.
func (Ride) PacketName() string { return "Ride" }

// MountNpcID returns the mount's NPC ID, 0 when dismounting.
func (r Ride) MountNpcID() int32 { return r.MountNpcTypeID - npcTypeOffset }

// Position returns where the player is.
func (r Ride) Position() Position { return Position{r.X, r.Y, r.Z} }

func readRide(f *fields) Packet {
	var p Ride
	p.ObjectID = f.i32()
	p.Mounted = f.flag32("mounted flag")
	p.MountType = f.i32()
	p.MountNpcTypeID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	return p
}

// encodeRequestGetItemFromPet builds RequestGetItemFromPet: d object ID,
// d count, and a word Mobius reads and ignores (0).
func encodeRequestGetItemFromPet(objectID, count int32) []byte {
	return encodeInts(OpRequestGetItemFromPet, objectID, count, 0)
}

// ChangePetName names the player's pet (RequestChangePetName). Mobius
// answers with PetInfo and a PetStatusUpdate, or a system message; nothing
// is awaited.
func (c *GameConn) ChangePetName(ctx context.Context, name string) error {
	return c.send(ctx, encodeString(OpRequestChangePetName, name))
}

// PetUseItem uses the pet's item objectID like a double click
// (RequestPetUseItem): it equips or unequips pet gear and feeds the pet.
// Mobius reads only the object ID. Nothing is awaited.
func (c *GameConn) PetUseItem(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeInts(OpRequestPetUseItem, objectID))
}

// GiveItemToPet moves count of the inventory item objectID to the pet
// (RequestGiveItemToPet). Nothing is awaited: the answers are an
// InventoryUpdate and a PetInventoryUpdate.
func (c *GameConn) GiveItemToPet(ctx context.Context, objectID, count int32) error {
	return c.send(ctx, encodeInts(OpRequestGiveItemToPet, objectID, count))
}

// GetItemFromPet moves count of the pet's item objectID to the inventory
// (RequestGetItemFromPet). Nothing is awaited.
func (c *GameConn) GetItemFromPet(ctx context.Context, objectID, count int32) error {
	return c.send(ctx, encodeRequestGetItemFromPet(objectID, count))
}

// PetPickUp sends the pet to pick up the ground item objectID
// (RequestPetGetItem). Nothing is awaited.
func (c *GameConn) PetPickUp(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeInts(OpRequestPetGetItem, objectID))
}
