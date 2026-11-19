package testclient

// The server packets of Milestone 7: casting, learning skills, shortcuts,
// soulshots and buffs. Every reader follows the writeImpl of the Mobius
// class of the same name, field by field.

// MagicSkillUse starts casting a skill (0x48). MagicSkillLaunched follows
// when the skill takes effect.
type MagicSkillUse struct {
	CasterID int32
	// TargetID is the caster itself for a skill on oneself.
	TargetID int32
	SkillID  int32
	Level    int32
	// HitTime is the casting time and ReuseDelay the reuse time, in
	// milliseconds.
	HitTime    int32
	ReuseDelay int32
	// X, Y and Z are the caster's position.
	X, Y, Z int32
	// Critical is a flag Interlude never sets. When it is set a short
	// follows it, Reserved, which Mobius writes as 0.
	Critical bool
	Reserved uint16
	// TargetX, TargetY and TargetZ are the target's position.
	TargetX, TargetY, TargetZ int32
}

// PacketName implements Packet.
func (MagicSkillUse) PacketName() string { return "MagicSkillUse" }

func readMagicSkillUse(f *fields) Packet {
	var p MagicSkillUse
	p.CasterID = f.i32()
	p.TargetID = f.i32()
	p.SkillID = f.i32()
	p.Level = f.i32()
	p.HitTime = f.i32()
	p.ReuseDelay = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Critical = f.flag32("critical flag")
	if p.Critical {
		p.Reserved = f.u16()
	}
	p.TargetX = f.i32()
	p.TargetY = f.i32()
	p.TargetZ = f.i32()
	return p
}

// MagicSkillLaunched shows a skill taking effect on its targets (0x76).
type MagicSkillLaunched struct {
	CasterID int32
	SkillID  int32
	Level    int32
	// Targets are the objects the skill hit; Mobius lists the caster for a
	// skill without targets.
	Targets []int32
}

// PacketName implements Packet.
func (MagicSkillLaunched) PacketName() string { return "MagicSkillLaunched" }

func readMagicSkillLaunched(f *fields) Packet {
	var p MagicSkillLaunched
	p.CasterID = f.i32()
	p.SkillID = f.i32()
	p.Level = f.i32()
	n := f.count32("target count")
	for i := 0; i < n && f.ok(); i++ {
		p.Targets = append(p.Targets, f.i32())
	}
	return p
}

// MagicSkillCanceled stops a cast (0x49).
type MagicSkillCanceled struct {
	ObjectID int32
}

// PacketName implements Packet.
func (MagicSkillCanceled) PacketName() string { return "MagicSkillCanceled" }

func readMagicSkillCanceled(f *fields) Packet { return MagicSkillCanceled{ObjectID: f.i32()} }

// Colours of SetupGauge.
const (
	GaugeBlue  int32 = 0
	GaugeRed   int32 = 1
	GaugeCyan  int32 = 2
	GaugeGreen int32 = 3
)

// SetupGauge shows the player's cast bar (0x6d). Interlude sends no object
// ID: the bar is always the player's own.
type SetupGauge struct {
	Color int32
	// Time is where the bar starts and MaxTime its full length, in
	// milliseconds; both are the casting time for a cast.
	Time    int32
	MaxTime int32
}

// PacketName implements Packet.
func (SetupGauge) PacketName() string { return "SetupGauge" }

func readSetupGauge(f *fields) Packet {
	var p SetupGauge
	p.Color = f.i32()
	p.Time = f.i32()
	p.MaxTime = f.i32()
	return p
}

// Types of AcquireSkillList, AcquireSkillInfo, RequestAcquireSkillInfo and
// RequestAcquireSkill (Mobius AcquireSkillType ordinals).
const (
	AcquireSkillClass   int32 = 0
	AcquireSkillFishing int32 = 1
	AcquireSkillPledge  int32 = 2
)

// LearnableSkill is one skill of AcquireSkillList.
type LearnableSkill struct {
	ID        int32
	NextLevel int32
	MaxLevel  int32
	SPCost    int32
	// Requirements is the number of required items; Mobius writes 0 for
	// class skills.
	Requirements int32
}

// AcquireSkillList is a trainer's list of skills to learn (0x8a). Mobius
// does not send an empty list: AcquireSkillDone comes instead.
type AcquireSkillList struct {
	// Type is AcquireSkillClass, AcquireSkillFishing or AcquireSkillPledge.
	Type   int32
	Skills []LearnableSkill
}

// PacketName implements Packet.
func (AcquireSkillList) PacketName() string { return "AcquireSkillList" }

// Find returns the entry of skill id.
func (l AcquireSkillList) Find(id int32) (LearnableSkill, bool) {
	for _, s := range l.Skills {
		if s.ID == id {
			return s, true
		}
	}
	return LearnableSkill{}, false
}

func readAcquireSkillList(f *fields) Packet {
	var p AcquireSkillList
	p.Type = f.i32()
	n := f.count32("skill count")
	for i := 0; i < n && f.ok(); i++ {
		var s LearnableSkill
		s.ID = f.i32()
		s.NextLevel = f.i32()
		s.MaxLevel = f.i32()
		s.SPCost = f.i32()
		s.Requirements = f.i32()
		p.Skills = append(p.Skills, s)
	}
	return p
}

// SkillRequirement is an item that learning a skill takes, such as a
// spellbook.
type SkillRequirement struct {
	// Type is 99 in Mobius.
	Type   int32
	ItemID int32
	// Count is written as a long.
	Count int64
	// Unknown is a word Mobius writes as 50.
	Unknown int32
}

// AcquireSkillInfo is what learning one skill level costs (0x8b), the
// answer to RequestAcquireSkillInfo.
type AcquireSkillInfo struct {
	SkillID      int32
	Level        int32
	SPCost       int32
	Type         int32
	Requirements []SkillRequirement
}

// PacketName implements Packet.
func (AcquireSkillInfo) PacketName() string { return "AcquireSkillInfo" }

func readAcquireSkillInfo(f *fields) Packet {
	var p AcquireSkillInfo
	p.SkillID = f.i32()
	p.Level = f.i32()
	p.SPCost = f.i32()
	p.Type = f.i32()
	n := f.count32("requirement count")
	for i := 0; i < n && f.ok(); i++ {
		var r SkillRequirement
		r.Type = f.i32()
		r.ItemID = f.i32()
		r.Count = f.i64()
		r.Unknown = f.i32()
		p.Requirements = append(p.Requirements, r)
	}
	return p
}

// AcquireSkillDone tells that the trainer has nothing more to teach
// (0x8e). It has no fields.
type AcquireSkillDone struct{}

// PacketName implements Packet.
func (AcquireSkillDone) PacketName() string { return "AcquireSkillDone" }

func readAcquireSkillDone(*fields) Packet { return AcquireSkillDone{} }

// ShortcutRegister sets one slot of the shortcut bar (0x44), the answer to
// RequestShortcutReg.
//
// Its layout is shorter than a ShortcutInit entry. After the type and slot
// Mobius writes the ID; for a skill the level and a zero byte; and then a
// word 1 for every type. The reader puts that word in Level for the types
// without a level, where ShortcutInit has the same 1, and the zero byte and
// the word of a skill in Rest, as ShortcutInit does. An item shortcut has
// no Rest here.
type ShortcutRegister struct {
	Shortcut
}

// PacketName implements Packet.
func (ShortcutRegister) PacketName() string { return "ShortcutRegister" }

func readShortcutRegister(f *fields) Packet {
	s := Shortcut{Type: f.i32(), Slot: f.i32()}
	switch s.Type {
	case ShortcutItem, ShortcutAction, ShortcutMacro, ShortcutRecipe:
		s.ID = f.i32()
		s.Level = f.i32()
	case ShortcutSkill:
		s.ID = f.i32()
		s.Level = f.i32()
		s.Rest = f.r.ReadBytes(5)
	default:
		f.failf("shortcut type %d", s.Type)
	}
	return ShortcutRegister{Shortcut: s}
}

// ExAutoSoulShot turns the automatic use of a shot item on or off (Ex
// 0x12), the answer to RequestAutoSoulShot.
type ExAutoSoulShot struct {
	ItemID  int32
	Enabled bool
}

// PacketName implements Packet.
func (ExAutoSoulShot) PacketName() string { return "ExAutoSoulShot" }

func readExAutoSoulShot(f *fields) Packet {
	var p ExAutoSoulShot
	p.ItemID = f.i32()
	p.Enabled = f.flag32("enabled flag")
	return p
}

// Kinds of the owner of PartySpelled.
const (
	SpelledPlayer   int32 = 0
	SpelledPet      int32 = 1
	SpelledServitor int32 = 2
)

// PartySpelled lists the buffs of a party member or of the player's pet or
// servitor (0xee), for the icons of the party window.
type PartySpelled struct {
	// Kind is SpelledPlayer, SpelledPet or SpelledServitor.
	Kind     int32
	ObjectID int32
	// Effects are laid out as in AbnormalStatusUpdate: d skill, h level and
	// d remaining seconds.
	Effects []Effect
}

// PacketName implements Packet.
func (PartySpelled) PacketName() string { return "PartySpelled" }

func readPartySpelled(f *fields) Packet {
	var p PartySpelled
	p.Kind = f.i32()
	p.ObjectID = f.i32()
	n := f.count32("effect count")
	for i := 0; i < n && f.ok(); i++ {
		var e Effect
		e.SkillID = f.i32()
		e.Level = f.u16()
		e.Duration = f.i32()
		p.Effects = append(p.Effects, e)
	}
	return p
}

// ShortBuffStatusUpdate shows the short buff icon (0xf4), such as a potion
// effect. SkillID 0 removes it.
type ShortBuffStatusUpdate struct {
	SkillID int32
	Level   int32
	// Duration is the remaining time in seconds.
	Duration int32
}

// PacketName implements Packet.
func (ShortBuffStatusUpdate) PacketName() string { return "ShortBuffStatusUpdate" }

func readShortBuffStatusUpdate(f *fields) Packet {
	var p ShortBuffStatusUpdate
	p.SkillID = f.i32()
	p.Level = f.i32()
	p.Duration = f.i32()
	return p
}
