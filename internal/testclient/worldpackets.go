package testclient

// The server packets of Milestone 3: entering the world, the lists sent on
// entering, status, movement, chat and leaving. Every reader follows the
// writeImpl of the Mobius class of the same name, field by field.

import "fmt"

// Position is a point in the world.
type Position struct {
	X, Y, Z int32
}

// CharSelected answers CharacterSelect (0x15).
type CharSelected struct {
	Name     string
	ObjectID int32
	Title    string
	// SessionID is the PlayOK1 session key.
	SessionID int32
	ClanID    int32
	// Unknown is a word Mobius writes as 0.
	Unknown int32
	Sex     int32
	Race    int32
	ClassID int32
	// Active is a word Mobius writes as 1.
	Active    int32
	X, Y, Z   int32
	CurrentHP float64
	CurrentMP float64
	SP        int32
	Exp       int64
	Level     int32
	Karma     int32
	PkKills   int32
	// Stats in packet order: INT, STR, CON, MEN, DEX, WIT.
	INT, STR, CON, MEN, DEX, WIT int32
	// Reserved holds the 30 words Mobius writes as zero after the stats,
	// Reserved2 the two zero words after them.
	Reserved  [30]int32
	Reserved2 [2]int32
	// GameTime is the game time in minutes since midnight.
	GameTime int32
	// Reserved3 is a zero word.
	Reserved3 int32
	// ClassID2 repeats the class ID.
	ClassID2 int32
	// Reserved4 holds the 12 trailing zero words.
	Reserved4 [12]int32
}

// PacketName implements Packet.
func (CharSelected) PacketName() string { return "CharSelected" }

func readCharSelected(f *fields) Packet {
	var p CharSelected
	p.Name = f.str()
	p.ObjectID = f.i32()
	p.Title = f.str()
	p.SessionID = f.i32()
	p.ClanID = f.i32()
	p.Unknown = f.i32()
	p.Sex = f.i32()
	p.Race = f.i32()
	p.ClassID = f.i32()
	p.Active = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.CurrentHP = f.f64()
	p.CurrentMP = f.f64()
	p.SP = f.i32()
	p.Exp = f.i64()
	p.Level = f.i32()
	p.Karma = f.i32()
	p.PkKills = f.i32()
	p.INT = f.i32()
	p.STR = f.i32()
	p.CON = f.i32()
	p.MEN = f.i32()
	p.DEX = f.i32()
	p.WIT = f.i32()
	f.i32s(p.Reserved[:])
	f.i32s(p.Reserved2[:])
	p.GameTime = f.i32()
	p.Reserved3 = f.i32()
	p.ClassID2 = f.i32()
	f.i32s(p.Reserved4[:])
	return p
}

// UserInfo is the full state of the player's own character (0x04). Mobius
// sends it on entering, after a teleport and whenever something in it
// changes.
type UserInfo struct {
	X, Y, Z     int32
	VehicleID   int32
	ObjectID    int32
	Name        string
	Race        int32
	Sex         int32
	BaseClassID int32
	Level       int32
	Exp         int64

	STR, DEX, CON, INT, WIT, MEN int32

	MaxHP, HP        int32
	MaxMP, MP        int32
	SP               int32
	CurLoad, MaxLoad int32
	// WeaponFlag is 40 with a weapon equipped, 20 without.
	WeaponFlag int32

	// PaperdollObjectIDs and PaperdollItemIDs are indexed by the Slot
	// constants. The last slot is SlotFace: UserInfo lists the face slot
	// where CharSelectionInfo lists HAIR2.
	PaperdollObjectIDs [NumEquipSlots]int32
	PaperdollItemIDs   [NumEquipSlots]int32

	// Reserved1, Reserved2 and Reserved3 are the zero shorts Mobius writes
	// around the two copies of the weapon's augmentation ID.
	Reserved1       [14]uint16
	AugmentationID  int32
	Reserved2       [12]uint16
	AugmentationID2 int32
	Reserved3       [4]uint16

	PAtk     int32
	PAtkSpd  int32
	PDef     int32
	Evasion  int32
	Accuracy int32
	Critical int32
	MAtk     int32
	CastSpd  int32
	// PAtkSpd2 repeats PAtkSpd.
	PAtkSpd2 int32
	MDef     int32
	PvpFlag  int32
	Karma    int32

	RunSpd, WalkSpd         int32
	SwimRunSpd, SwimWalkSpd int32
	FlyRunSpd, FlyWalkSpd   int32
	// FlyRunSpd2 and FlyWalkSpd2 repeat the flying speeds.
	FlyRunSpd2, FlyWalkSpd2 int32

	MoveMultiplier        float64
	AttackSpeedMultiplier float64
	CollisionRadius       float64
	CollisionHeight       float64

	HairStyle int32
	HairColor int32
	Face      int32
	// GM is the "builder level" word: 1 for a GM.
	GM    bool
	Title string

	ClanID      int32
	ClanCrestID int32
	AllyID      int32
	AllyCrestID int32
	// Relation holds the clan leader and siege flags.
	Relation         int32
	MountType        uint8
	PrivateStoreType uint8
	DwarvenCraft     bool
	PkKills          int32
	PvpKills         int32
	Cubics           []uint16
	PartyMatchRoom   bool
	AbnormalEffects  int32
	InWater          bool
	ClanPrivileges   int32
	RecomLeft        uint16
	RecomHave        uint16
	// MountNpcID is the mount's NPC ID plus 1,000,000, or 0.
	MountNpcID     int32
	InventoryLimit uint16
	ClassID        int32
	// SpecialEffects is a word Mobius writes as 0.
	SpecialEffects   int32
	MaxCP, CP        int32
	EnchantEffect    uint8
	Team             uint8
	ClanCrestLargeID int32
	Noble            bool
	Hero             bool
	Fishing          bool
	FishX            int32
	FishY            int32
	FishZ            int32
	NameColor        int32
	Running          bool
	PledgeClass      int32
	PledgeType       int32
	TitleColor       int32
	// CursedWeaponLevel is the level of an equipped cursed weapon, or 0.
	CursedWeaponLevel int32
}

// PacketName implements Packet.
func (UserInfo) PacketName() string { return "UserInfo" }

// Position returns where the character stands.
func (u UserInfo) Position() Position { return Position{u.X, u.Y, u.Z} }

func readUserInfo(f *fields) Packet {
	var p UserInfo
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.VehicleID = f.i32()
	p.ObjectID = f.i32()
	p.Name = f.str()
	p.Race = f.i32()
	p.Sex = f.i32()
	p.BaseClassID = f.i32()
	p.Level = f.i32()
	p.Exp = f.i64()
	p.STR = f.i32()
	p.DEX = f.i32()
	p.CON = f.i32()
	p.INT = f.i32()
	p.WIT = f.i32()
	p.MEN = f.i32()
	p.MaxHP = f.i32()
	p.HP = f.i32()
	p.MaxMP = f.i32()
	p.MP = f.i32()
	p.SP = f.i32()
	p.CurLoad = f.i32()
	p.MaxLoad = f.i32()
	p.WeaponFlag = f.i32()
	f.i32s(p.PaperdollObjectIDs[:])
	f.i32s(p.PaperdollItemIDs[:])
	f.u16s(p.Reserved1[:])
	p.AugmentationID = f.i32()
	f.u16s(p.Reserved2[:])
	p.AugmentationID2 = f.i32()
	f.u16s(p.Reserved3[:])
	p.PAtk = f.i32()
	p.PAtkSpd = f.i32()
	p.PDef = f.i32()
	p.Evasion = f.i32()
	p.Accuracy = f.i32()
	p.Critical = f.i32()
	p.MAtk = f.i32()
	p.CastSpd = f.i32()
	p.PAtkSpd2 = f.i32()
	p.MDef = f.i32()
	p.PvpFlag = f.i32()
	p.Karma = f.i32()
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
	p.HairStyle = f.i32()
	p.HairColor = f.i32()
	p.Face = f.i32()
	p.GM = f.flag32("GM flag")
	p.Title = f.str()
	p.ClanID = f.i32()
	p.ClanCrestID = f.i32()
	p.AllyID = f.i32()
	p.AllyCrestID = f.i32()
	p.Relation = f.i32()
	p.MountType = f.u8()
	p.PrivateStoreType = f.u8()
	p.DwarvenCraft = f.flag8("dwarven craft flag")
	p.PkKills = f.i32()
	p.PvpKills = f.i32()
	p.Cubics = f.cubics()
	p.PartyMatchRoom = f.flag8("party match room flag")
	p.AbnormalEffects = f.i32()
	p.InWater = f.flag8("water flag")
	p.ClanPrivileges = f.i32()
	p.RecomLeft = f.u16()
	p.RecomHave = f.u16()
	p.MountNpcID = f.i32()
	p.InventoryLimit = f.u16()
	p.ClassID = f.i32()
	p.SpecialEffects = f.i32()
	p.MaxCP = f.i32()
	p.CP = f.i32()
	p.EnchantEffect = f.u8()
	p.Team = f.u8()
	p.ClanCrestLargeID = f.i32()
	p.Noble = f.flag8("noble flag")
	p.Hero = f.flag8("hero flag")
	p.Fishing = f.flag8("fishing flag")
	p.FishX = f.i32()
	p.FishY = f.i32()
	p.FishZ = f.i32()
	p.NameColor = f.i32()
	p.Running = f.flag8("running flag")
	p.PledgeClass = f.i32()
	p.PledgeType = f.i32()
	p.TitleColor = f.i32()
	p.CursedWeaponLevel = f.i32()
	return p
}

// Visible equipment slots of CharInfo, in packet order. CharInfo lists the
// item IDs of these 12 slots only, without object IDs.
const (
	VisibleUnder = iota
	VisibleHead
	VisibleRHand
	VisibleLHand
	VisibleGloves
	VisibleChest
	VisibleLegs
	VisibleFeet
	VisibleCloak
	// VisibleRHand2 repeats the right hand.
	VisibleRHand2
	VisibleHair
	VisibleHair2

	// NumVisibleSlots is the number of item IDs in CharInfo.
	NumVisibleSlots
)

// CharInfoPacket is the CharInfo server packet (0x03): another player as
// the viewer sees it. The Go name keeps CharInfo for the entries of the
// character list.
type CharInfoPacket struct {
	X, Y, Z     int32
	VehicleID   int32
	ObjectID    int32
	Name        string
	Race        int32
	Sex         int32
	BaseClassID int32
	// ItemIDs are indexed by the Visible constants.
	ItemIDs [NumVisibleSlots]int32

	// Reserved1, Reserved2 and Reserved3 are the zero shorts around the two
	// copies of the weapon's augmentation ID.
	Reserved1       [4]uint16
	AugmentationID  int32
	Reserved2       [12]uint16
	AugmentationID2 int32
	Reserved3       [4]uint16

	PvpFlag int32
	Karma   int32
	MAtkSpd int32
	PAtkSpd int32
	// PvpFlag2 and Karma2 repeat PvpFlag and Karma.
	PvpFlag2 int32
	Karma2   int32

	RunSpd, WalkSpd         int32
	SwimRunSpd, SwimWalkSpd int32
	FlyRunSpd, FlyWalkSpd   int32
	FlyRunSpd2, FlyWalkSpd2 int32

	MoveMultiplier        float64
	AttackSpeedMultiplier float64
	CollisionRadius       float64
	CollisionHeight       float64

	HairStyle   int32
	HairColor   int32
	Face        int32
	Title       string
	ClanID      int32
	ClanCrestID int32
	AllyID      int32
	AllyCrestID int32
	// Relation is a word Mobius writes as 0: RelationChanged carries the
	// relation.
	Relation         int32
	Standing         bool
	Running          bool
	InCombat         bool
	AlikeDead        bool
	Invisible        bool
	MountType        uint8
	PrivateStoreType uint8
	Cubics           []uint16
	PartyMatchRoom   bool
	AbnormalEffects  int32
	RecomLeft        uint8
	RecomHave        uint16
	ClassID          int32
	MaxCP, CP        int32
	EnchantEffect    uint8
	Team             uint8
	ClanCrestLargeID int32
	Noble            bool
	Hero             bool
	Fishing          bool
	FishX            int32
	FishY            int32
	FishZ            int32
	NameColor        int32
	Heading          int32
	PledgeClass      int32
	PledgeType       int32
	TitleColor       int32
	// CursedWeaponLevel is the level of an equipped cursed weapon, or 0.
	CursedWeaponLevel int32
}

// PacketName implements Packet.
func (CharInfoPacket) PacketName() string { return "CharInfo" }

// Position returns where the player stands.
func (c CharInfoPacket) Position() Position { return Position{c.X, c.Y, c.Z} }

func readCharInfoPacket(f *fields) Packet {
	var p CharInfoPacket
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.VehicleID = f.i32()
	p.ObjectID = f.i32()
	p.Name = f.str()
	p.Race = f.i32()
	p.Sex = f.i32()
	p.BaseClassID = f.i32()
	f.i32s(p.ItemIDs[:])
	f.u16s(p.Reserved1[:])
	p.AugmentationID = f.i32()
	f.u16s(p.Reserved2[:])
	p.AugmentationID2 = f.i32()
	f.u16s(p.Reserved3[:])
	p.PvpFlag = f.i32()
	p.Karma = f.i32()
	p.MAtkSpd = f.i32()
	p.PAtkSpd = f.i32()
	p.PvpFlag2 = f.i32()
	p.Karma2 = f.i32()
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
	p.HairStyle = f.i32()
	p.HairColor = f.i32()
	p.Face = f.i32()
	p.Title = f.str()
	p.ClanID = f.i32()
	p.ClanCrestID = f.i32()
	p.AllyID = f.i32()
	p.AllyCrestID = f.i32()
	p.Relation = f.i32()
	p.Standing = f.flag8("standing flag")
	p.Running = f.flag8("running flag")
	p.InCombat = f.flag8("combat flag")
	p.AlikeDead = f.flag8("dead flag")
	p.Invisible = f.flag8("invisible flag")
	p.MountType = f.u8()
	p.PrivateStoreType = f.u8()
	p.Cubics = f.cubics()
	p.PartyMatchRoom = f.flag8("party match room flag")
	p.AbnormalEffects = f.i32()
	p.RecomLeft = f.u8()
	p.RecomHave = f.u16()
	p.ClassID = f.i32()
	p.MaxCP = f.i32()
	p.CP = f.i32()
	p.EnchantEffect = f.u8()
	p.Team = f.u8()
	p.ClanCrestLargeID = f.i32()
	p.Noble = f.flag8("noble flag")
	p.Hero = f.flag8("hero flag")
	p.Fishing = f.flag8("fishing flag")
	p.FishX = f.i32()
	p.FishY = f.i32()
	p.FishZ = f.i32()
	p.NameColor = f.i32()
	p.Heading = f.i32()
	p.PledgeClass = f.i32()
	p.PledgeType = f.i32()
	p.TitleColor = f.i32()
	p.CursedWeaponLevel = f.i32()
	return p
}

// RelationChanged tells the viewer how it relates to a player (0xce): the
// clan, siege and war flags behind name colours and attackability.
type RelationChanged struct {
	ObjectID       int32
	Relation       int32
	AutoAttackable bool
	Karma          int32
	PvpFlag        int32
}

// PacketName implements Packet.
func (RelationChanged) PacketName() string { return "RelationChanged" }

func readRelationChanged(f *fields) Packet {
	var p RelationChanged
	p.ObjectID = f.i32()
	p.Relation = f.i32()
	p.AutoAttackable = f.flag32("auto-attackable flag")
	p.Karma = f.i32()
	p.PvpFlag = f.i32()
	return p
}

// Item is one item of ItemList and InventoryUpdate, as Mobius
// AbstractItemPacket.writeItem writes it.
type Item struct {
	Type1       uint16
	ObjectID    int32
	ItemID      int32
	Count       int32
	Type2       uint16
	CustomType1 uint16
	Equipped    bool
	// BodyPart is the slot mask of the item.
	BodyPart       int32
	Enchant        uint16
	CustomType2    uint16
	AugmentationID int32
	// Mana is the shadow item mana, -1 for other items.
	Mana int32
}

func readItem(f *fields) Item {
	var it Item
	it.Type1 = f.u16()
	it.ObjectID = f.i32()
	it.ItemID = f.i32()
	it.Count = f.i32()
	it.Type2 = f.u16()
	it.CustomType1 = f.u16()
	it.Equipped = f.flag16("equipped flag")
	it.BodyPart = f.i32()
	it.Enchant = f.u16()
	it.CustomType2 = f.u16()
	it.AugmentationID = f.i32()
	it.Mana = f.i32()
	return it
}

// ItemList is the whole inventory (0x1b).
type ItemList struct {
	// ShowWindow opens the inventory window; EnterWorld sends false.
	ShowWindow bool
	Items      []Item
}

// PacketName implements Packet.
func (ItemList) PacketName() string { return "ItemList" }

func readItemList(f *fields) Packet {
	var p ItemList
	p.ShowWindow = f.flag16("show window flag")
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		p.Items = append(p.Items, readItem(f))
	}
	return p
}

// Changes of InventoryUpdate.
const (
	ItemAdded    uint16 = 1
	ItemModified uint16 = 2
	ItemRemoved  uint16 = 3
)

// ItemChange is one entry of InventoryUpdate.
type ItemChange struct {
	// Change is ItemAdded, ItemModified or ItemRemoved.
	Change uint16
	Item
}

// InventoryUpdate changes some items of the inventory (0x27), for example
// when a teleport fee takes adena.
type InventoryUpdate struct {
	Items []ItemChange
}

// PacketName implements Packet.
func (InventoryUpdate) PacketName() string { return "InventoryUpdate" }

func readInventoryUpdate(f *fields) Packet {
	var p InventoryUpdate
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		change := f.u16()
		p.Items = append(p.Items, ItemChange{Change: change, Item: readItem(f)})
	}
	return p
}

// Shortcut types (Mobius ShortcutType ordinals).
const (
	ShortcutNone   int32 = 0
	ShortcutItem   int32 = 1
	ShortcutSkill  int32 = 2
	ShortcutAction int32 = 3
	ShortcutMacro  int32 = 4
	ShortcutRecipe int32 = 5
)

// Shortcut is one entry of ShortcutInit. What follows Slot depends on
// Type, as Mobius ShortcutInit.writeImpl writes it.
type Shortcut struct {
	Type int32
	// Slot is the slot plus 12 times the page.
	Slot int32
	// ID is the item object ID, or the skill, action, macro or recipe ID.
	// A ShortcutNone entry has no further fields.
	ID int32
	// Level is the skill level of a skill and the 1 Mobius writes for the
	// other types.
	Level int32
	// Rest holds the bytes after Level: for an item the words -1, 0, 0 and
	// two zero shorts, for a skill a zero byte and the word 1.
	Rest []byte
}

// ShortcutInit is the shortcut bar (0x45).
type ShortcutInit struct {
	Shortcuts []Shortcut
}

// PacketName implements Packet.
func (ShortcutInit) PacketName() string { return "ShortcutInit" }

func readShortcutInit(f *fields) Packet {
	var p ShortcutInit
	n := f.count32("shortcut count")
	for i := 0; i < n && f.ok(); i++ {
		s := Shortcut{Type: f.i32(), Slot: f.i32()}
		switch s.Type {
		case ShortcutNone:
		case ShortcutItem:
			s.ID = f.i32()
			s.Level = f.i32()
			s.Rest = f.r.ReadBytes(16)
		case ShortcutSkill:
			s.ID = f.i32()
			s.Level = f.i32()
			s.Rest = f.r.ReadBytes(5)
		case ShortcutAction, ShortcutMacro, ShortcutRecipe:
			s.ID = f.i32()
			s.Level = f.i32()
		default:
			f.failf("shortcut type %d", s.Type)
		}
		p.Shortcuts = append(p.Shortcuts, s)
	}
	return p
}

// Quest is one quest of QuestList.
type Quest struct {
	ID int32
	// State is the current step (cond), or a bit mask of steps with the top
	// bit set.
	State int32
}

// QuestList lists the active quests (0x80).
type QuestList struct {
	Quests []Quest
}

// PacketName implements Packet.
func (QuestList) PacketName() string { return "QuestList" }

func readQuestList(f *fields) Packet {
	var p QuestList
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		p.Quests = append(p.Quests, Quest{ID: f.i32(), State: f.i32()})
	}
	return p
}

// Skill is one skill of SkillList.
type Skill struct {
	Passive  bool
	Level    int32
	ID       int32
	Disabled bool
}

// SkillList lists the known skills (0x58). Mobius sends it 300 ms after
// entering the world.
type SkillList struct {
	Skills []Skill
}

// PacketName implements Packet.
func (SkillList) PacketName() string { return "SkillList" }

func readSkillList(f *fields) Packet {
	var p SkillList
	n := f.count32("skill count")
	for i := 0; i < n && f.ok(); i++ {
		var s Skill
		s.Passive = f.flag32("passive flag")
		s.Level = f.i32()
		s.ID = f.i32()
		s.Disabled = f.flag8("disabled flag")
		p.Skills = append(p.Skills, s)
	}
	return p
}

// MacroCommand is one command of a macro.
type MacroCommand struct {
	// Index counts the commands from 1.
	Index uint8
	// Type is 1 for a skill, 3 for an action, 4 for a shortcut.
	Type    uint8
	D1      int32
	D2      uint8
	Command string
}

// Macro is the macro SendMacroList carries.
type Macro struct {
	ID          int32
	Name        string
	Description string
	Acronym     string
	Icon        uint8
	Commands    []MacroCommand
}

// SendMacroList sends one macro, or none (0xe7). Mobius sends one packet
// per macro, each with the total count, and one empty packet when there
// are no macros.
type SendMacroList struct {
	Revision int32
	// Unknown is a byte Mobius writes as 0.
	Unknown uint8
	Count   uint8
	// Macro is nil when the packet carries none.
	Macro *Macro
}

// PacketName implements Packet.
func (SendMacroList) PacketName() string { return "SendMacroList" }

func readSendMacroList(f *fields) Packet {
	var p SendMacroList
	p.Revision = f.i32()
	p.Unknown = f.u8()
	p.Count = f.u8()
	if !f.flag8("macro flag") {
		return p
	}
	m := &Macro{}
	m.ID = f.i32()
	m.Name = f.str()
	m.Description = f.str()
	m.Acronym = f.str()
	m.Icon = f.u8()
	n := f.count8()
	for i := 0; i < n && f.ok(); i++ {
		var c MacroCommand
		c.Index = f.u8()
		c.Type = f.u8()
		c.D1 = f.i32()
		c.D2 = f.u8()
		c.Command = f.str()
		m.Commands = append(m.Commands, c)
	}
	p.Macro = m
	return p
}

// Henna is one dye of HennaInfo.
type Henna struct {
	DyeID int32
	// Unknown is a word Mobius writes as 1.
	Unknown int32
}

// HennaInfo lists the dyes and their stat changes (0xe4).
type HennaInfo struct {
	// Stat changes in packet order: INT, STR, CON, MEN, DEX, WIT.
	INT, STR, CON, MEN, DEX, WIT int8
	// Slots is 3 in Mobius.
	Slots  int32
	Hennas []Henna
}

// PacketName implements Packet.
func (HennaInfo) PacketName() string { return "HennaInfo" }

func readHennaInfo(f *fields) Packet {
	var p HennaInfo
	p.INT = f.i8()
	p.STR = f.i8()
	p.CON = f.i8()
	p.MEN = f.i8()
	p.DEX = f.i8()
	p.WIT = f.i8()
	p.Slots = f.i32()
	n := f.count32("henna count")
	for i := 0; i < n && f.ok(); i++ {
		p.Hennas = append(p.Hennas, Henna{DyeID: f.i32(), Unknown: f.i32()})
	}
	return p
}

// Friend is one entry of FriendList.
type Friend struct {
	ObjectID int32
	Name     string
	Online   bool
	// OnlineObjectID is the object ID when the friend is online, else 0.
	OnlineObjectID int32
}

// FriendList lists the friends (0xfa).
type FriendList struct {
	Friends []Friend
}

// PacketName implements Packet.
func (FriendList) PacketName() string { return "FriendList" }

func readFriendList(f *fields) Packet {
	var p FriendList
	n := f.count32("friend count")
	for i := 0; i < n && f.ok(); i++ {
		var fr Friend
		fr.ObjectID = f.i32()
		fr.Name = f.str()
		fr.Online = f.flag32("online flag")
		fr.OnlineObjectID = f.i32()
		p.Friends = append(p.Friends, fr)
	}
	return p
}

// SkillCooldown is one entry of SkillCoolTime, in seconds.
type SkillCooldown struct {
	SkillID   int32
	Level     int32
	Reuse     int32
	Remaining int32
}

// SkillCoolTime lists the skills that are still cooling down (0xc1).
type SkillCoolTime struct {
	Skills []SkillCooldown
}

// PacketName implements Packet.
func (SkillCoolTime) PacketName() string { return "SkillCoolTime" }

func readSkillCoolTime(f *fields) Packet {
	var p SkillCoolTime
	n := f.count32("skill count")
	for i := 0; i < n && f.ok(); i++ {
		var s SkillCooldown
		s.SkillID = f.i32()
		s.Level = f.i32()
		s.Reuse = f.i32()
		s.Remaining = f.i32()
		p.Skills = append(p.Skills, s)
	}
	return p
}

// Effect is one effect icon of AbnormalStatusUpdate.
type Effect struct {
	SkillID int32
	Level   uint16
	// Duration is the remaining time in seconds.
	Duration int32
}

// AbnormalStatusUpdate lists the buff and debuff icons (0x7f).
type AbnormalStatusUpdate struct {
	Effects []Effect
}

// PacketName implements Packet.
func (AbnormalStatusUpdate) PacketName() string { return "AbnormalStatusUpdate" }

func readAbnormalStatusUpdate(f *fields) Packet {
	var p AbnormalStatusUpdate
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		var e Effect
		e.SkillID = f.i32()
		e.Level = f.u16()
		e.Duration = f.i32()
		p.Effects = append(p.Effects, e)
	}
	return p
}

// ExStorageMaxCount gives the storage sizes (Ex 0x2e).
type ExStorageMaxCount struct {
	Inventory     int32
	Warehouse     int32
	ClanWarehouse int32
	PrivateSell   int32
	PrivateBuy    int32
	DwarfRecipe   int32
	CommonRecipe  int32
	// InventoryExtra is the extra slots from the INV_LIM stat.
	InventoryExtra int32
}

// PacketName implements Packet.
func (ExStorageMaxCount) PacketName() string { return "ExStorageMaxCount" }

func readExStorageMaxCount(f *fields) Packet {
	var p ExStorageMaxCount
	p.Inventory = f.i32()
	p.Warehouse = f.i32()
	p.ClanWarehouse = f.i32()
	p.PrivateSell = f.i32()
	p.PrivateBuy = f.i32()
	p.DwarfRecipe = f.i32()
	p.CommonRecipe = f.i32()
	p.InventoryExtra = f.i32()
	return p
}

// EtcStatusUpdate carries the status icons of the character (0xf3).
type EtcStatusUpdate struct {
	// Charges is the force level, 0-7.
	Charges int32
	// WeightPenalty is the weight penalty level, 0-4.
	WeightPenalty  int32
	ChatBlocked    bool
	DangerArea     bool
	GradePenalty   bool
	CharmOfCourage bool
	// DeathPenalty is the death penalty level, 0-15.
	DeathPenalty int32
}

// PacketName implements Packet.
func (EtcStatusUpdate) PacketName() string { return "EtcStatusUpdate" }

func readEtcStatusUpdate(f *fields) Packet {
	var p EtcStatusUpdate
	p.Charges = f.i32()
	p.WeightPenalty = f.i32()
	p.ChatBlocked = f.flag32("chat block flag")
	p.DangerArea = f.flag32("danger area flag")
	p.GradePenalty = f.flag32("grade penalty flag")
	p.CharmOfCourage = f.flag32("charm of courage flag")
	p.DeathPenalty = f.i32()
	return p
}

// ValidateLocation corrects where an object stands (0x61).
type ValidateLocation struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// PacketName implements Packet.
func (ValidateLocation) PacketName() string { return "ValidateLocation" }

// Position returns the corrected position.
func (v ValidateLocation) Position() Position { return Position{v.X, v.Y, v.Z} }

func readValidateLocation(f *fields) Packet {
	var p ValidateLocation
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Heading = f.i32()
	return p
}

// ActionFailed ends a request the server did not carry out, or a group of
// answers (0x25). It has no fields.
type ActionFailed struct{}

// PacketName implements Packet.
func (ActionFailed) PacketName() string { return "ActionFailed" }

func readActionFailed(*fields) Packet { return ActionFailed{} }

// MoveToLocation starts a movement (server 0x01). Mobius writes the
// destination before the current position.
type MoveToLocation struct {
	ObjectID            int32
	DestX, DestY, DestZ int32
	X, Y, Z             int32
}

// PacketName implements Packet.
func (MoveToLocation) PacketName() string { return "MoveToLocation" }

// Origin returns where the movement starts.
func (m MoveToLocation) Origin() Position { return Position{m.X, m.Y, m.Z} }

// Destination returns where the movement ends.
func (m MoveToLocation) Destination() Position { return Position{m.DestX, m.DestY, m.DestZ} }

func readMoveToLocation(f *fields) Packet {
	var p MoveToLocation
	p.ObjectID = f.i32()
	p.DestX = f.i32()
	p.DestY = f.i32()
	p.DestZ = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	return p
}

// StopMove stops a movement (0x47).
type StopMove struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// PacketName implements Packet.
func (StopMove) PacketName() string { return "StopMove" }

// Position returns where the object stopped.
func (s StopMove) Position() Position { return Position{s.X, s.Y, s.Z} }

func readStopMove(f *fields) Packet {
	var p StopMove
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Heading = f.i32()
	return p
}

// DeleteObject removes an object from the client's view (0x12).
type DeleteObject struct {
	ObjectID int32
	// Disappear is 1 in Mobius: a mounted object disappears instead of
	// dismounting.
	Disappear int32
}

// PacketName implements Packet.
func (DeleteObject) PacketName() string { return "DeleteObject" }

func readDeleteObject(f *fields) Packet {
	var p DeleteObject
	p.ObjectID = f.i32()
	p.Disappear = f.i32()
	return p
}

// TeleportToLocation moves an object at once (0x28). For the player's own
// character the client loads the area and answers with Appearing.
type TeleportToLocation struct {
	ObjectID int32
	X, Y, Z  int32
	// Fade is 0 (fade) in Mobius; 1 would be instant.
	Fade    int32
	Heading int32
}

// PacketName implements Packet.
func (TeleportToLocation) PacketName() string { return "TeleportToLocation" }

// Position returns the teleport destination.
func (t TeleportToLocation) Position() Position { return Position{t.X, t.Y, t.Z} }

func readTeleportToLocation(f *fields) Packet {
	var p TeleportToLocation
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Fade = f.i32()
	p.Heading = f.i32()
	return p
}

// ChangeMoveType switches between running and walking (0x2e).
type ChangeMoveType struct {
	ObjectID int32
	Running  bool
	// Unknown is a word Mobius writes as 0.
	Unknown int32
}

// PacketName implements Packet.
func (ChangeMoveType) PacketName() string { return "ChangeMoveType" }

func readChangeMoveType(f *fields) Packet {
	var p ChangeMoveType
	p.ObjectID = f.i32()
	p.Running = f.flag32("running flag")
	p.Unknown = f.i32()
	return p
}

// Wait types of ChangeWaitType.
const (
	WaitSitting        int32 = 0
	WaitStanding       int32 = 1
	WaitStartFakeDeath int32 = 2
	WaitStopFakeDeath  int32 = 3
)

// ChangeWaitType makes an object sit down, stand up or fake death (0x2f).
type ChangeWaitType struct {
	ObjectID int32
	WaitType int32
	X, Y, Z  int32
}

// PacketName implements Packet.
func (ChangeWaitType) PacketName() string { return "ChangeWaitType" }

func readChangeWaitType(f *fields) Packet {
	var p ChangeWaitType
	p.ObjectID = f.i32()
	p.WaitType = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	return p
}

// StartRotation starts turning an object (Mobius BEGIN_ROTATION, 0x62).
type StartRotation struct {
	ObjectID int32
	Degree   int32
	Side     int32
	Speed    int32
}

// PacketName implements Packet.
func (StartRotation) PacketName() string { return "StartRotation" }

func readStartRotation(f *fields) Packet {
	var p StartRotation
	p.ObjectID = f.i32()
	p.Degree = f.i32()
	p.Side = f.i32()
	p.Speed = f.i32()
	return p
}

// StopRotation ends turning an object at Degree (0x63).
type StopRotation struct {
	ObjectID int32
	Degree   int32
	Speed    int32
	// Unknown is a byte Mobius writes as 0.
	Unknown uint8
}

// PacketName implements Packet.
func (StopRotation) PacketName() string { return "StopRotation" }

func readStopRotation(f *fields) Packet {
	var p StopRotation
	p.ObjectID = f.i32()
	p.Degree = f.i32()
	p.Speed = f.i32()
	p.Unknown = f.u8()
	return p
}

// CreatureSay is a chat message (0x4a).
//
// Mobius writes the sender name and the text. Its boat announcements
// (ChatBoat) carry a name ID and a system message ID instead: a CreatureSay
// of type ChatBoat with exactly two words after the type is read that way.
type CreatureSay struct {
	// ObjectID is the sender, 0 for messages without one.
	ObjectID int32
	Type     ChatType
	Name     string
	Text     string
	// NameID and MessageID are set for the boat announcements.
	NameID    int32
	MessageID int32
}

// PacketName implements Packet.
func (CreatureSay) PacketName() string { return "CreatureSay" }

func readCreatureSay(f *fields) Packet {
	var p CreatureSay
	p.ObjectID = f.i32()
	p.Type = ChatType(f.i32())
	if p.Type == ChatBoat && f.r.Len() == 8 {
		p.NameID = f.i32()
		p.MessageID = f.i32()
		return p
	}
	p.Name = f.str()
	p.Text = f.str()
	return p
}

// System message parameter types (Mobius SystemMessage TYPE_*).
const (
	SMParamText         int32 = 0
	SMParamInt          int32 = 1
	SMParamNpcName      int32 = 2
	SMParamItemName     int32 = 3
	SMParamSkillName    int32 = 4
	SMParamCastleName   int32 = 5
	SMParamLong         int32 = 6
	SMParamZoneName     int32 = 7
	SMParamElementName  int32 = 9
	SMParamInstanceName int32 = 10
	SMParamDoorName     int32 = 11
	SMParamPlayerName   int32 = 12
	SMParamSystemString int32 = 13
)

// System message IDs that the M3 and M4 tests look for (Mobius
// SystemMessageId).
const (
	// SysMsgS1IsNotCurrentlyLoggedIn: "$s1 is not currently logged in."
	SysMsgS1IsNotCurrentlyLoggedIn int32 = 3
	// SysMsgWelcome: "Welcome to the World of Lineage II."
	SysMsgWelcome int32 = 34
	// SysMsgIncorrectTarget: "That is the incorrect target."
	SysMsgIncorrectTarget int32 = 144
	// SysMsgPlayerNotOnline: "That player is not online."
	SysMsgPlayerNotOnline int32 = 145
	// SysMsgNotEnoughAdena: "You do not have enough adena."
	SysMsgNotEnoughAdena int32 = 279
	// SysMsgLocationTalkingIsland: "Current location : $s1, $s2, $s3 (Near
	// Talking Island Village)", the first of the /loc messages.
	SysMsgLocationTalkingIsland int32 = 910
	// SysMsgText is "$s1" (S1_2): the text of Player.sendMessage.
	SysMsgText int32 = 1983
)

// SMParam is one parameter of a system message.
type SMParam struct {
	Type int32
	// Text is set for SMParamText and SMParamPlayerName.
	Text string
	// Number is set for the number, item, NPC, castle, element, instance,
	// door and system string parameters.
	Number int64
	// Numbers holds the skill ID and level of SMParamSkillName, or x, y, z
	// of SMParamZoneName.
	Numbers []int32
}

// SystemMessage is a message from the client's message table, with
// parameters (0x64).
type SystemMessage struct {
	ID     int32
	Params []SMParam
}

// PacketName implements Packet.
func (SystemMessage) PacketName() string { return "SystemMessage" }

// Texts returns the text parameters, in order.
func (m SystemMessage) Texts() []string {
	var s []string
	for _, p := range m.Params {
		if p.Type == SMParamText || p.Type == SMParamPlayerName {
			s = append(s, p.Text)
		}
	}
	return s
}

func readSystemMessage(f *fields) Packet {
	var p SystemMessage
	p.ID = f.i32()
	n := f.count32("parameter count")
	for i := 0; i < n && f.ok(); i++ {
		p.Params = append(p.Params, readSMParam(f))
	}
	return p
}

func readSMParam(f *fields) SMParam {
	prm := SMParam{Type: f.i32()}
	switch prm.Type {
	case SMParamText, SMParamPlayerName:
		prm.Text = f.str()
	case SMParamLong:
		prm.Number = f.i64()
	case SMParamItemName, SMParamCastleName, SMParamInt, SMParamNpcName,
		SMParamElementName, SMParamSystemString, SMParamInstanceName, SMParamDoorName:
		prm.Number = int64(f.i32())
	case SMParamSkillName:
		prm.Numbers = make([]int32, 2)
		f.i32s(prm.Numbers)
	case SMParamZoneName:
		prm.Numbers = make([]int32, 3)
		f.i32s(prm.Numbers)
	default:
		f.failf("parameter type %d", prm.Type)
	}
	return prm
}

// RestartResponse answers RequestRestart (0x5f).
type RestartResponse struct {
	// OK is true when the character leaves the world; the character list
	// follows.
	OK bool
}

// PacketName implements Packet.
func (RestartResponse) PacketName() string { return "RestartResponse" }

func readRestartResponse(f *fields) Packet {
	return RestartResponse{OK: f.flag32("result")}
}

// LeaveWorld answers Logout before the server closes the connection
// (0x7e). It has no fields.
type LeaveWorld struct{}

// PacketName implements Packet.
func (LeaveWorld) PacketName() string { return "LeaveWorld" }

func readLeaveWorld(*fields) Packet { return LeaveWorld{} }

// ServerClose tells the client the server closes the connection (0x26),
// for example when a banned character is selected. It has no fields.
type ServerClose struct{}

// PacketName implements Packet.
func (ServerClose) PacketName() string { return "ServerClose" }

func readServerClose(*fields) Packet { return ServerClose{} }

// CharDeleteSuccess accepts CharacterDelete (0x23). It has no fields.
type CharDeleteSuccess struct{}

// PacketName implements Packet.
func (CharDeleteSuccess) PacketName() string { return "CharDeleteSuccess" }

func readCharDeleteSuccess(*fields) Packet { return CharDeleteSuccess{} }

// DeleteFailReason is the reason of CharDeleteFail.
type DeleteFailReason int32

// CharDeleteFail reasons.
const (
	DeleteFailFailed      DeleteFailReason = 1
	DeleteFailClanMember  DeleteFailReason = 2
	DeleteFailClanLeader  DeleteFailReason = 3
	deleteFailReasonCount                  = 4
)

func (r DeleteFailReason) String() string {
	names := [deleteFailReasonCount]string{"", "deletion failed", "clan member", "clan leader"}
	if r > 0 && r < deleteFailReasonCount {
		return fmt.Sprintf("%d (%s)", int32(r), names[r])
	}
	return fmt.Sprintf("%d", int32(r))
}

// CharDeleteFail refuses CharacterDelete (0x24).
type CharDeleteFail struct {
	Reason DeleteFailReason
}

// PacketName implements Packet.
func (CharDeleteFail) PacketName() string { return "CharDeleteFail" }

func readCharDeleteFail(f *fields) Packet {
	return CharDeleteFail{Reason: DeleteFailReason(f.i32())}
}
