package testclient

// The server packets of Milestone 4: NPCs, targets, the HP bar, dialogs,
// social actions and NPC speech. Every reader follows the writeImpl of the
// Mobius class of the same name, field by field.

// NpcInfo shows an NPC, a monster or a summon (0x16, Mobius
// AbstractNpcInfo.NpcInfo).
type NpcInfo struct {
	ObjectID int32
	// NpcTypeID is the template's display ID plus 1,000,000.
	NpcTypeID  int32
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

	RHand int32
	Chest int32
	LHand int32
	// ShowName is 1 in Mobius.
	ShowName  uint8
	Running   bool
	InCombat  bool
	AlikeDead bool
	// Summoned is 2 while the NPC shows its summon animation, else 0.
	Summoned uint8
	Name     string
	Title    string
	// TitleColor is 0, the client's default colour, for NPCs.
	TitleColor      int32
	PvpFlag         int32
	Karma           int32
	AbnormalEffects int32
	ClanID          int32
	ClanCrestID     int32
	AllyID          int32
	AllyCrestID     int32
	// Environment is 1 in water, 2 flying, else 0.
	Environment uint8
	Team        uint8
	// CollisionRadius2 and CollisionHeight2 repeat the collision size.
	CollisionRadius2 float64
	CollisionHeight2 float64
	EnchantEffect    int32
	Flying           bool
}

// PacketName implements Packet.
func (NpcInfo) PacketName() string { return "NpcInfo" }

// NpcID returns the template's display ID.
func (n NpcInfo) NpcID() int32 { return n.NpcTypeID - npcTypeOffset }

// Position returns where the NPC stands.
func (n NpcInfo) Position() Position { return Position{n.X, n.Y, n.Z} }

func readNpcInfo(f *fields) Packet {
	var p NpcInfo
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
	p.ShowName = f.u8()
	p.Running = f.flag8("running flag")
	p.InCombat = f.flag8("combat flag")
	p.AlikeDead = f.flag8("dead flag")
	p.Summoned = f.u8()
	p.Name = f.str()
	p.Title = f.str()
	p.TitleColor = f.i32()
	p.PvpFlag = f.i32()
	p.Karma = f.i32()
	p.AbnormalEffects = f.i32()
	p.ClanID = f.i32()
	p.ClanCrestID = f.i32()
	p.AllyID = f.i32()
	p.AllyCrestID = f.i32()
	p.Environment = f.u8()
	p.Team = f.u8()
	p.CollisionRadius2 = f.f64()
	p.CollisionHeight2 = f.f64()
	p.EnchantEffect = f.i32()
	p.Flying = f.flag32("flying flag")
	return p
}

// ServerObjectInfo shows an NPC drawn as a static object (0x8c), such as
// some town decorations.
type ServerObjectInfo struct {
	ObjectID int32
	// NpcTypeID is the template's display ID plus 1,000,000.
	NpcTypeID int32
	// Name is set only for templates with server-side names.
	Name       string
	Attackable bool
	X, Y, Z    int32
	Heading    int32
	// MoveMultiplier and AttackSpeedMultiplier are 1.0 in Mobius.
	MoveMultiplier        float64
	AttackSpeedMultiplier float64
	CollisionRadius       float64
	CollisionHeight       float64
	// HP and MaxHP are 0 unless the object is attackable.
	HP    int32
	MaxHP int32
	// ObjectType is 1 and SpecialEffects 0 in Mobius.
	ObjectType     int32
	SpecialEffects int32
}

// PacketName implements Packet.
func (ServerObjectInfo) PacketName() string { return "ServerObjectInfo" }

// NpcID returns the template's display ID.
func (s ServerObjectInfo) NpcID() int32 { return s.NpcTypeID - npcTypeOffset }

// Position returns where the object stands.
func (s ServerObjectInfo) Position() Position { return Position{s.X, s.Y, s.Z} }

func readServerObjectInfo(f *fields) Packet {
	var p ServerObjectInfo
	p.ObjectID = f.i32()
	p.NpcTypeID = f.i32()
	p.Name = f.str()
	p.Attackable = f.flag32("attackable flag")
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Heading = f.i32()
	p.MoveMultiplier = f.f64()
	p.AttackSpeedMultiplier = f.f64()
	p.CollisionRadius = f.f64()
	p.CollisionHeight = f.f64()
	p.HP = f.i32()
	p.MaxHP = f.i32()
	p.ObjectType = f.i32()
	p.SpecialEffects = f.i32()
	return p
}

// MyTargetSelected confirms the player's new target (0xa6).
type MyTargetSelected struct {
	ObjectID int32
	// Color is the level difference (player minus target) for targets the
	// player can attack, else 0. The client colours the target's name by
	// it.
	Color int16
	// Unknown is a word Mobius writes as 0.
	Unknown int32
}

// PacketName implements Packet.
func (MyTargetSelected) PacketName() string { return "MyTargetSelected" }

func readMyTargetSelected(f *fields) Packet {
	var p MyTargetSelected
	p.ObjectID = f.i32()
	p.Color = f.i16()
	p.Unknown = f.i32()
	return p
}

// TargetSelected tells the viewers that an object selected a target
// (0x29).
type TargetSelected struct {
	ObjectID int32
	TargetID int32
	X, Y, Z  int32
	// Unknown is a word Mobius writes as 0.
	Unknown int32
}

// PacketName implements Packet.
func (TargetSelected) PacketName() string { return "TargetSelected" }

func readTargetSelected(f *fields) Packet {
	var p TargetSelected
	p.ObjectID = f.i32()
	p.TargetID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Unknown = f.i32()
	return p
}

// TargetUnselected tells the viewers that an object dropped its target
// (0x2a).
type TargetUnselected struct {
	ObjectID int32
	X, Y, Z  int32
	// Unknown is a word Mobius writes as 0.
	Unknown int32
}

// PacketName implements Packet.
func (TargetUnselected) PacketName() string { return "TargetUnselected" }

func readTargetUnselected(f *fields) Packet {
	var p TargetUnselected
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Unknown = f.i32()
	return p
}

// StatusUpdate attribute IDs (Mobius StatusUpdate).
const (
	StatusLevel    int32 = 0x01
	StatusSTR      int32 = 0x03
	StatusDEX      int32 = 0x04
	StatusCON      int32 = 0x05
	StatusINT      int32 = 0x06
	StatusWIT      int32 = 0x07
	StatusMEN      int32 = 0x08
	StatusCurHP    int32 = 0x09
	StatusMaxHP    int32 = 0x0a
	StatusCurMP    int32 = 0x0b
	StatusMaxMP    int32 = 0x0c
	StatusCurLoad  int32 = 0x0e
	StatusMaxLoad  int32 = 0x0f
	StatusPAtk     int32 = 0x11
	StatusAtkSpd   int32 = 0x12
	StatusPDef     int32 = 0x13
	StatusEvasion  int32 = 0x14
	StatusAccuracy int32 = 0x15
	StatusCritical int32 = 0x16
	StatusMAtk     int32 = 0x17
	StatusCastSpd  int32 = 0x18
	StatusMDef     int32 = 0x19
	StatusPvpFlag  int32 = 0x1a
	StatusKarma    int32 = 0x1b
	StatusCurCP    int32 = 0x21
	StatusMaxCP    int32 = 0x22
)

// StatusAttribute is one value of StatusUpdate.
type StatusAttribute struct {
	ID    int32
	Value int32
}

// StatusUpdate changes some status values of an object (0x0e), for example
// the HP bar of a targeted NPC.
type StatusUpdate struct {
	ObjectID   int32
	Attributes []StatusAttribute
}

// PacketName implements Packet.
func (StatusUpdate) PacketName() string { return "StatusUpdate" }

// Value returns the last value of attribute id.
func (s StatusUpdate) Value(id int32) (int32, bool) {
	v, ok := int32(0), false
	for _, a := range s.Attributes {
		if a.ID == id {
			v, ok = a.Value, true
		}
	}
	return v, ok
}

func readStatusUpdate(f *fields) Packet {
	var p StatusUpdate
	p.ObjectID = f.i32()
	n := f.count32("attribute count")
	for i := 0; i < n && f.ok(); i++ {
		p.Attributes = append(p.Attributes, StatusAttribute{ID: f.i32(), Value: f.i32()})
	}
	return p
}

// NpcHtmlMessage opens a dialog window (0x0f).
type NpcHtmlMessage struct {
	// ObjectID is the NPC the page belongs to, 0 for pages without one.
	ObjectID int32
	HTML     string
	// ItemID is the item of an item page, else 0.
	ItemID int32
}

// PacketName implements Packet.
func (NpcHtmlMessage) PacketName() string { return "NpcHtmlMessage" }

func readNpcHtmlMessage(f *fields) Packet {
	var p NpcHtmlMessage
	p.ObjectID = f.i32()
	p.HTML = f.str()
	p.ItemID = f.i32()
	return p
}

// MoveToPawn moves an object toward another one until it is Distance away
// (0x60).
type MoveToPawn struct {
	ObjectID                  int32
	TargetID                  int32
	Distance                  int32
	X, Y, Z                   int32
	TargetX, TargetY, TargetZ int32
}

// PacketName implements Packet.
func (MoveToPawn) PacketName() string { return "MoveToPawn" }

func readMoveToPawn(f *fields) Packet {
	var p MoveToPawn
	p.ObjectID = f.i32()
	p.TargetID = f.i32()
	p.Distance = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.TargetX = f.i32()
	p.TargetY = f.i32()
	p.TargetZ = f.i32()
	return p
}

// SocialAction plays an animation (0x2d): a player's social action
// (2-13) or an NPC's.
type SocialAction struct {
	ObjectID int32
	ActionID int32
}

// PacketName implements Packet.
func (SocialAction) PacketName() string { return "SocialAction" }

func readSocialAction(f *fields) Packet {
	var p SocialAction
	p.ObjectID = f.i32()
	p.ActionID = f.i32()
	return p
}

// NpcSay is an NPC speaking in the chat (0x02).
type NpcSay struct {
	ObjectID int32
	Type     ChatType
	// NpcTypeID is the template's display ID plus 1,000,000.
	NpcTypeID int32
	Text      string
}

// PacketName implements Packet.
func (NpcSay) PacketName() string { return "NpcSay" }

// NpcID returns the template's display ID.
func (n NpcSay) NpcID() int32 { return n.NpcTypeID - npcTypeOffset }

func readNpcSay(f *fields) Packet {
	var p NpcSay
	p.ObjectID = f.i32()
	p.Type = ChatType(f.i32())
	p.NpcTypeID = f.i32()
	p.Text = f.str()
	return p
}

// ShowMiniMap opens the map window (0x9d).
type ShowMiniMap struct {
	MapID int32
	// Period is the Seven Signs period.
	Period uint8
}

// PacketName implements Packet.
func (ShowMiniMap) PacketName() string { return "ShowMiniMap" }

func readShowMiniMap(f *fields) Packet {
	var p ShowMiniMap
	p.MapID = f.i32()
	p.Period = f.u8()
	return p
}
