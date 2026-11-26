package summon

// ActionKind is what a summon action of the action window does (Mobius
// RequestActionUse).
type ActionKind uint8

// The summon action kinds.
const (
	// ActFollow toggles following the owner (15 pets, 21 servitors).
	ActFollow ActionKind = iota + 1
	// ActAttack attacks the owner's target (16, 22).
	ActAttack
	// ActStop stops what the summon does (17, 23).
	ActStop
	// ActUnsummonPet sends a pet back (19).
	ActUnsummonPet
	// ActUnsummonServitor sends a servitor back (52).
	ActUnsummonServitor
	// ActMoveTo walks to the owner's target (53, 54).
	ActMoveTo
	// ActMount mounts or dismounts a strider or wyvern (38).
	ActMount
	// ActSkillByName casts the skill the template's parameters name.
	ActSkillByName
	// ActSkillByID casts a skill by ID at the level the summon has.
	ActSkillByID
	// ActSinEaterTalk makes the Sin Eater say something (1001).
	ActSinEaterTalk
	// ActDoorSkill casts a skill by ID at a door target only (41, 1000).
	ActDoorSkill
)

// ActionDef is a summon action: its Mobius name, its kind, whether it
// validates a pet (true) or a servitor (false), and the skill it casts (by
// template parameter name or by ID), at the owner rather than the owner's
// target when OnOwner is set.
type ActionDef struct {
	Name      string
	Kind      ActionKind
	Pet       bool
	SkillName string
	SkillID   int
	OnOwner   bool
}

// actionList is Mobius RequestActionUse's summon cases, in its order.
var actionList = []struct {
	id  int32
	def ActionDef
}{
	{15, ActionDef{Name: "Change Movement Mode (Pets)", Kind: ActFollow, Pet: true}},
	{16, ActionDef{Name: "Attack (Pets)", Kind: ActAttack, Pet: true}},
	{17, ActionDef{Name: "Stop (Pets)", Kind: ActStop, Pet: true}},
	{19, ActionDef{Name: "Unsummon Pet", Kind: ActUnsummonPet, Pet: true}},
	{21, ActionDef{Name: "Change Movement Mode (Servitors)", Kind: ActFollow}},
	{22, ActionDef{Name: "Attack (Servitors)", Kind: ActAttack}},
	{23, ActionDef{Name: "Stop (Servitors)", Kind: ActStop}},
	{32, ActionDef{Name: "Wild Hog Cannon - Wild Cannon", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{36, ActionDef{Name: "Soulless - Toxic Smoke", Kind: ActSkillByName, SkillName: "RangeDebuff"}},
	{38, ActionDef{Name: "Mount/Dismount", Kind: ActMount}},
	{39, ActionDef{Name: "Soulless - Parasite Burst", Kind: ActSkillByName, SkillName: "RangeDD"}},
	{41, ActionDef{Name: "Wild Hog Cannon - Attack", Kind: ActDoorSkill, SkillID: 4230}},
	{42, ActionDef{Name: "Kai the Cat - Self Damage Shield", Kind: ActSkillByName, SkillName: "HealMagic"}},
	{43, ActionDef{Name: "Merrow the Unicorn - Hydro Screw", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{44, ActionDef{Name: "Big Boom - Boom Attack", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{45, ActionDef{Name: "Boxer the Unicorn - Master Recharge", Kind: ActSkillByName, SkillName: "HealMagic", OnOwner: true}},
	{46, ActionDef{Name: "Mew the Cat - Mega Storm Strike", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{47, ActionDef{Name: "Silhouette - Steal Blood", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{48, ActionDef{Name: "Mechanic Golem - Mech. Cannon", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{52, ActionDef{Name: "Unsummon Servitor", Kind: ActUnsummonServitor}},
	{53, ActionDef{Name: "Move to target (Servitors)", Kind: ActMoveTo}},
	{54, ActionDef{Name: "Move to target (Pets)", Kind: ActMoveTo, Pet: true}},
	{1000, ActionDef{Name: "Siege Golem - Siege Hammer", Kind: ActDoorSkill, SkillID: 4079}},
	{1001, ActionDef{Name: "Sin Eater - Ultimate Bombastic Buster", Kind: ActSinEaterTalk, Pet: true}},
	{1003, ActionDef{Name: "Wind Hatchling/Strider - Wild Stun", Kind: ActSkillByName, SkillName: "PhysicalSpecial", Pet: true}},
	{1004, ActionDef{Name: "Wind Hatchling/Strider - Wild Defense", Kind: ActSkillByName, SkillName: "Buff", Pet: true, OnOwner: true}},
	{1005, ActionDef{Name: "Star Hatchling/Strider - Bright Burst", Kind: ActSkillByName, SkillName: "DDMagic", Pet: true}},
	{1006, ActionDef{Name: "Star Hatchling/Strider - Bright Heal", Kind: ActSkillByName, SkillName: "Heal", Pet: true, OnOwner: true}},
	{1007, ActionDef{Name: "Feline Queen - Blessing of Queen", Kind: ActSkillByName, SkillName: "Buff1", OnOwner: true}},
	{1008, ActionDef{Name: "Feline Queen - Gift of Queen", Kind: ActSkillByName, SkillName: "Buff2", OnOwner: true}},
	{1009, ActionDef{Name: "Feline Queen - Cure of Queen", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{1010, ActionDef{Name: "Unicorn Seraphim - Blessing of Seraphim", Kind: ActSkillByName, SkillName: "Buff1", OnOwner: true}},
	{1011, ActionDef{Name: "Unicorn Seraphim - Gift of Seraphim", Kind: ActSkillByName, SkillName: "Buff2", OnOwner: true}},
	{1012, ActionDef{Name: "Unicorn Seraphim - Cure of Seraphim", Kind: ActSkillByName, SkillName: "DDMagic"}},
	{1013, ActionDef{Name: "Nightshade - Curse of Shade", Kind: ActSkillByName, SkillName: "DeBuff1"}},
	{1014, ActionDef{Name: "Nightshade - Mass Curse of Shade", Kind: ActSkillByName, SkillName: "DeBuff2"}},
	{1015, ActionDef{Name: "Nightshade - Shade Sacrifice", Kind: ActSkillByName, SkillName: "Heal"}},
	{1016, ActionDef{Name: "Cursed Man - Cursed Blow", Kind: ActSkillByName, SkillName: "PhysicalSpecial1"}},
	{1017, ActionDef{Name: "Cursed Man - Cursed Strike", Kind: ActSkillByName, SkillName: "PhysicalSpecial2"}},
	{1031, ActionDef{Name: "Feline King - Slash", Kind: ActSkillByName, SkillName: "PhysicalSpecial1"}},
	{1032, ActionDef{Name: "Feline King - Spinning Slash", Kind: ActSkillByName, SkillName: "PhysicalSpecial2"}},
	{1033, ActionDef{Name: "Feline King - Hold of King", Kind: ActSkillByName, SkillName: "PhysicalSpecial3"}},
	{1034, ActionDef{Name: "Magnus the Unicorn - Whiplash", Kind: ActSkillByName, SkillName: "PhysicalSpecial1"}},
	{1035, ActionDef{Name: "Magnus the Unicorn - Tridal Wave", Kind: ActSkillByName, SkillName: "PhysicalSpecial2"}},
	{1036, ActionDef{Name: "Spectral Lord - Corpse Kaboom", Kind: ActSkillByName, SkillName: "PhysicalSpecial1"}},
	{1037, ActionDef{Name: "Spectral Lord - Dicing Death", Kind: ActSkillByName, SkillName: "PhysicalSpecial2"}},
	{1038, ActionDef{Name: "Spectral Lord - Dark Curse", Kind: ActSkillByName, SkillName: "PhysicalSpecial3"}},
	{1039, ActionDef{Name: "Swoop Cannon - Cannon Fodder", Kind: ActSkillByID, SkillID: 5110}},
	{1040, ActionDef{Name: "Swoop Cannon - Big Bang", Kind: ActSkillByID, SkillID: 5111}},
}

// Actions are the summon actions of Mobius RequestActionUse by action ID.
// The player actions (sit, run, stores, social actions) are not here.
var Actions = func() map[int32]ActionDef {
	m := make(map[int32]ActionDef, len(actionList))
	for _, a := range actionList {
		m[a.id] = a.def
	}
	return m
}()

// ValidFor is Mobius RequestActionUse.validateSummon's kind test: a pet
// action takes a pet, and every action takes a servitor (Mobius's
// "(checkPet && isPet()) || isServitor()").
func (d ActionDef) ValidFor(k Kind) bool { return (d.Pet && k.IsPet()) || k == Servitor }

// SinEaterLines are what the Sin Eater says on action 1001 (Mobius
// RequestActionUse.NPC_STRINGS).
var SinEaterLines = []string{
	"Using a special skill here could trigger a bloodbath!",
	"Hey, what do you expect of me?",
	"Ugggggh! Push! It's not coming out!",
	"Ah, I missed the mark!",
}
