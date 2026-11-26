package scripting

import "time"

// Quest states, as Mobius's State class numbers them.
const (
	StateCreated   = 0
	StateStarted   = 1
	StateCompleted = 2
)

// QuestState is one player's state of one quest (Mobius QuestState without
// the parts that need the world). internal/quest implements it; the game
// hands it out through QuestWorld. Scripts reach it through the QuestState
// class of types/l2go.lua, whose methods add the world side (sounds, quest
// items) on top of these.
//
// Every method runs on the world loop. Values are L2J's strings.
type QuestState interface {
	// Quest returns the quest's name, e.g. "Q00001_LettersOfLove".
	Quest() string
	// State returns StateCreated, StateStarted or StateCompleted.
	State() int
	// SetState is Mobius setState: nothing when the state is unchanged,
	// otherwise the new state is saved and QuestList is sent.
	SetState(state int)
	// Get returns a variable; false when it is not set.
	Get(name string) (string, bool)
	// Set is Mobius set(name, value), including what it does for "cond":
	// the __compltdStateFlags arithmetic, QuestList and, for cond > 0 on a
	// quest that is not custom, ExShowQuestMark.
	Set(name, value string)
	// Unset is Mobius unset: it removes a variable ("cond" resets the cond
	// to 0).
	Unset(name string)
	// Cond is the cond as last set, whatever the state (Mobius _cond, which
	// isCond reads).
	Cond() int
	// Exit is Mobius exitQuest(repeatable) after the quest items are gone:
	// a repeatable quest's state is deleted (and QuestList sent), any other
	// becomes COMPLETED with its variables cleared. The caller checks that
	// the state is STARTED.
	Exit(repeatable bool)
}

// QuestWorld is the game world as quest scripts see it: quest states,
// time, items, sounds and pages. internal/game implements it and passes it
// in Options.World. The engine calls it only on the world loop, while a
// handler runs or a quest timer fires.
//
// Item IDs and counts are passed as scripts give them; the world ignores
// unknown item IDs, as Mobius does.
type QuestWorld interface {
	// QuestState returns the player's state of the quest, or nil.
	QuestState(p PlayerRef, quest string) QuestState
	// NewQuestState creates a CREATED state of the quest for the player
	// (Mobius newQuestState) and returns it. A CREATED state lives in memory
	// until a variable is set or its state changes.
	NewQuestState(p PlayerRef, quest string) QuestState

	// Now is the loop clock.
	Now() time.Time
	// After runs fn on the loop once, d from now; Every runs it every d
	// until stopped. stop cancels the timer; it may be called more than once
	// and after the timer ran.
	After(d time.Duration, fn func()) (stop func())
	Every(d time.Duration, fn func()) (stop func())

	// ItemCount is Mobius getQuestItemsCount(player, id): the count of the
	// item in the inventory, equipped ones included.
	ItemCount(p PlayerRef, itemID int) int64
	// EnchantLevel is Mobius getEnchantLevel: the enchant level of the
	// first item with this ID, 0 if none.
	EnchantLevel(p PlayerRef, itemID int) int
	// EquippedItemID is the item ID in a paperdoll slot (Mobius
	// getPaperdollItemId, the Paperdoll constants), 0 if empty.
	EquippedItemID(p PlayerRef, slot int) int
	// GiveItems is Mobius giveItems(player, id, count, enchant): the quest
	// item drop amount multiplier for quest items, the enchant level for
	// everything but adena, and the "You have earned" messages. count > 0.
	GiveItems(p PlayerRef, itemID int, count int64, enchant int)
	// RewardItems is Mobius rewardItems: GiveItems with the quest reward
	// rates instead of the multiplier. count > 0.
	RewardItems(p PlayerRef, itemID int, count int64)
	// AddItem is Mobius player.addItem(QUEST, id, count, npc, true), as
	// giveItemRandomly uses it, and reports whether the item was added.
	AddItem(p PlayerRef, itemID int, count int64, npc NpcRef) bool
	// CanAddItem is Mobius inventory.validateCapacityByItemId.
	CanAddItem(p PlayerRef, itemID int) bool
	// TakeItems is Mobius takeItems(player, id, count): count items, or
	// all of them for a negative count; an equipped item is unequipped
	// first.
	TakeItems(p PlayerRef, itemID int, count int64)
	// AddExpAndSp is Mobius addExpAndSp, with the quest rates.
	AddExpAndSp(p PlayerRef, exp, sp int64)

	// PlaySound sends PlaySound with a sound name (QuestSound values).
	PlaySound(p PlayerRef, sound string)
	// ShowReply shows a reply that has no caller to return to: a quest
	// timer's and q:show_html_file's. npc may be nil.
	ShowReply(p PlayerRef, npc NpcRef, r Reply)
	// HTML returns a page from the server's HTML folder (path relative to
	// it, e.g. "noquest.htm"), false if there is none.
	HTML(p PlayerRef, path string) (string, bool)

	// FindObject returns the PlayerRef or NpcRef with this object ID, or
	// nil (Mobius World.findObject).
	FindObject(objectID int32) any
	// GlobalVar and SetGlobalVar are Mobius GlobalVariablesManager.
	GlobalVar(name string) (string, bool)
	SetGlobalVar(name, value string)
}

// QuestActions are what quests make players and NPCs do. A QuestWorld may
// implement it; without it these calls raise a Lua error saying so.
type QuestActions interface {
	// SocialAction broadcasts SocialAction for a player or an NPC.
	SocialAction(obj any, action int)
	// Teleport moves a player (Mobius teleToLocation).
	Teleport(p PlayerRef, loc Location)
	// SendMessage sends a plain text system message.
	SendMessage(p PlayerRef, text string)
	// SendSystemMessage sends a SystemMessage with string parameters.
	SendSystemMessage(p PlayerRef, id int, params []string)
	// ShowOnScreenMsg sends ExShowScreenMessage (Mobius showOnScreenMsg
	// with the default position).
	ShowOnScreenMsg(p PlayerRef, text string, ms int)
	// Radar sends RadarControl: op 0 adds, 1 removes, 2 clears (Mobius
	// addRadar/removeRadar/clearRadar).
	Radar(p PlayerRef, op int, x, y, z int)
	// Tutorial sends a tutorial packet: TutorialShowQuestionMark (arg is
	// the number), the tutorial voice (PlaySound type 2), TutorialShowHtml
	// (arg is the page of Q00255_Tutorial), TutorialCloseHtml, or
	// TutorialEnableClientEvent (arg is the number).
	Tutorial(p PlayerRef, op TutorialOp, arg string)
	// NpcBroadcastSay is Mobius npc.broadcastSay(chatType, text).
	NpcBroadcastSay(npc NpcRef, chatType int, text string)
	// ShowChatWindow shows the NPC's own dialog (Mobius showChatWindow).
	ShowChatWindow(npc NpcRef, p PlayerRef)
	// DeleteNpc removes an NPC from the world (Mobius deleteMe).
	DeleteNpc(npc NpcRef)
	// AddSpawn spawns an NPC (Mobius addSpawn) and returns it, or nil.
	AddSpawn(npcID int, loc Location, randomOffset bool, despawn time.Duration, summon bool) NpcRef
}

// NpcActions are what quests make NPCs do beyond QuestActions: their AI
// and hate list, movement, death, spawn and skills (Mobius Npc, Attackable
// and their AI). A QuestWorld may implement it; without it these calls
// raise a Lua error saying so. A target is a PlayerRef or an NpcRef.
type NpcActions interface {
	// NpcAttack is Mobius npc.getAI().setIntention(ATTACK, target).
	NpcAttack(npc NpcRef, target any)
	// NpcAddDamageHate is Mobius Attackable.addDamageHate(target, damage,
	// hate).
	NpcAddDamageHate(npc NpcRef, target any, damage, hate int64)
	// NpcRemoveAttacker is Mobius npc.getAttackByList().remove(target).
	NpcRemoveAttacker(npc NpcRef, target any)
	// NpcAbortAttack is Mobius abortAttack.
	NpcAbortAttack(npc NpcRef)
	// NpcMoveTo is Mobius npc.getAI().setIntention(MOVE_TO, loc).
	NpcMoveTo(npc NpcRef, loc Location)
	// NpcSetRunning is Mobius setRunning.
	NpcSetRunning(npc NpcRef)
	// NpcDie is Mobius doDie(killer); killer is a PlayerRef or NpcRef.
	NpcDie(npc NpcRef, killer any)
	// NpcSetRespawn is Mobius npc.getSpawn().startRespawn() (on) and
	// stopRespawn() (off).
	NpcSetRespawn(npc NpcRef, on bool)
	// NpcCast is Mobius doCast(skill): at target, or at the NPC's current
	// target when target is nil.
	NpcCast(npc NpcRef, skill Skill, target any)
	// MagicSkillUse broadcasts MagicSkillUse of caster on target (Mobius
	// broadcastPacket(new MagicSkillUse(caster, target, id, level, hitTime,
	// reuseDelay))).
	MagicSkillUse(caster, target any, skillID, level int, hitTime, reuseDelay time.Duration)
	// NpcSetTarget is Mobius npc.setTarget(target); target may be nil.
	NpcSetTarget(npc NpcRef, target any)
	// NpcTeleport is Mobius npc.teleToLocation(x, y, z).
	NpcTeleport(npc NpcRef, loc Location)
	// NpcDropItem is Mobius npc.dropItem(player, id, count): the item falls
	// around the NPC, kept for the player for a while.
	NpcDropItem(npc NpcRef, p PlayerRef, itemID int, count int64)
	// NpcExists reports whether an NPC of the template is in the world
	// (Mobius World.getVisibleObjects() anyMatch getId() == npcID).
	NpcExists(npcID int) bool
}

// QuestSkills is the skill data and the effects as quests see them. A
// QuestWorld may implement it.
type QuestSkills interface {
	// SkillExists reports whether the level of the skill exists (Mobius
	// SkillData.getSkill(id, level) != null).
	SkillExists(id, level int) bool
	// IsAffectedBySkill is Mobius isAffectedBySkill on a PlayerRef or
	// NpcRef.
	IsAffectedBySkill(obj any, skillID int) bool
	// ApplyEffects is Mobius skill.applyEffects(effector, effected).
	ApplyEffects(skill Skill, effector, effected any)
}

// QuestItems is the item data as quests see it. A QuestWorld may implement
// it.
type QuestItems interface {
	// ItemTemplate returns an item template (Mobius
	// ItemData.getTemplate(id)), false for an unknown ID.
	ItemTemplate(id int) (ItemTemplate, bool)
}

// ItemTemplate is an item template as items.get returns it.
type ItemTemplate struct {
	ID        int
	Name      string
	Stackable bool
}

// TutorialOp selects the packet of QuestActions.Tutorial.
type TutorialOp int

// The tutorial operations.
const (
	TutorialQuestionMark TutorialOp = iota
	TutorialVoice
	TutorialShowHTML
	TutorialCloseHTML
	TutorialClientEvent
)

// Location is a point and a heading.
type Location struct {
	X, Y, Z, Heading int
}

// Located is a PlayerRef or NpcRef with a position: player:x(),
// npc:heading(), util.in_range and the radius checks need it.
type Located interface {
	Location() Location
}

// PlayerDetails is the part of a PlayerRef behind the quest-era Player
// methods (sex, base class, mage, newbie, dead, categories).
type PlayerDetails interface {
	// Sex is 0 male, 1 female.
	Sex() int
	// BaseClassID is the base class (Mobius getBaseClass).
	BaseClassID() int
	// IsMageClass is Mobius isMageClass.
	IsMageClass() bool
	// IsNewbie is Mobius isNewbie.
	IsNewbie() bool
	// IsDead reports whether the player is dead.
	IsDead() bool
	// InCategory is Mobius isInCategory with a Category value.
	InCategory(category int) bool
}

// PlayerClasses is the part of a PlayerRef behind the class change methods
// of the village masters and the class quests.
type PlayerClasses interface {
	// SetClassID is Mobius setPlayerClass: the class change with its
	// effect, message and skills.
	SetClassID(id int)
	// SetBaseClassID is Mobius setBaseClass.
	SetBaseClassID(id int)
	// BroadcastUserInfo is Mobius broadcastUserInfo.
	BroadcastUserInfo()
}

// PlayerVars is Mobius PlayerVariables: string values per character,
// saved with it.
type PlayerVars interface {
	Var(name string) (string, bool)
	SetVar(name, value string)
}

// NpcDetails is the part of an NpcRef behind npc:is_dead() and the script
// value methods.
type NpcDetails interface {
	IsDead() bool
	// ScriptValue and SetScriptValue are Mobius get/setScriptValue.
	ScriptValue() int
	SetScriptValue(v int)
	// IsType reports whether the NPC is of a Mobius NPC class such as
	// "Monster" or "VillageMasterPriest" (Mobius instanceof).
	IsType(class string) bool
}

// NpcStatus is the part of an NpcRef behind npc:level(), npc:hp() and
// npc:max_hp().
type NpcStatus interface {
	// Level is the template level (Mobius getLevel).
	Level() int
	// CurrentHP and MaxHP are Mobius getCurrentHp and getMaxHp.
	CurrentHP() float64
	MaxHP() float64
}

// PlayerCombat is the part of a PlayerRef behind player:active_weapon()
// and player:target().
type PlayerCombat interface {
	// ActiveWeaponID is the item ID of the weapon in the right hand, 0 for
	// none (Mobius getActiveWeaponItem().getId(), without the fists).
	ActiveWeaponID() int
	// TargetObject is the player's target: a PlayerRef, an NpcRef or nil
	// (Mobius getTarget).
	TargetObject() any
}

// PlayerStatus is the part of a PlayerRef behind player:pk_kills(),
// set_pk_kills(), update_user_info() and active_quests().
type PlayerStatus interface {
	// PkKills and SetPkKills are Mobius getPkKills and setPkKills.
	PkKills() int
	SetPkKills(n int)
	// UpdateUserInfo is Mobius updateUserInfo (UserInfo to the player).
	UpdateUserInfo()
	// ActiveQuests returns the IDs of the started quests with an ID from 1
	// to 19999 (Mobius getAllActiveQuests).
	ActiveQuests() []int
}

// PlayerNoble is the part of a PlayerRef behind player:is_noble() and
// player:set_noble().
type PlayerNoble interface {
	IsNoble() bool
	// SetNoble is Mobius setNoble: the flag and the noble skills.
	SetNoble(noble bool)
}

// PlayerSubclass is the part of a PlayerRef behind
// player:is_sub_class_active(), player:class_index() and
// player:total_sub_classes() (M11e). A ref without it is on its base class
// with no subclass.
type PlayerSubclass interface {
	// IsSubClassActive is Mobius isSubClassActive: the class index is not 0.
	IsSubClassActive() bool
	// ClassIndex is Mobius getClassIndex: 0 for the base class, 1-3.
	ClassIndex() int
	// TotalSubClasses is Mobius getTotalSubClasses.
	TotalSubClasses() int
}

// Item is an item as on_item_talk sees it.
type Item struct {
	ObjectID int32
	ID       int
	Count    int64
}

// Zone is a zone as the zone handlers see it.
type Zone struct {
	ID   int
	Type string
}

// Skill is a skill as the attack and skill handlers see it.
type Skill struct {
	ID, Level int
}

// Rates are the quest rates the script side applies (Mobius RatesConfig).
type Rates struct {
	// QuestItemDropAmount multiplies the amounts of giveItemRandomly
	// (QuestItemDropAmountMultiplier). Zero means 1.
	QuestItemDropAmount float64
}

func (r Rates) withDefaults() Rates {
	if r.QuestItemDropAmount <= 0 {
		r.QuestItemDropAmount = 1
	}
	return r
}

// Constants are constant tables generated from Mobius, made global tables
// for scripts. Reading a name a table does not have is an error, so a typo
// fails where it is.
type Constants struct {
	// Tables maps a global name (e.g. "SystemMessageId", "Category") to its
	// constants (name → number, string or boolean). A table here replaces a
	// built-in one of the same name (Config is merged instead, so the game
	// can set single values).
	Tables map[string]map[string]any
	// Classes fill the ClassId table and its helpers.
	Classes []ClassInfo
}

// ClassInfo is one player class (Mobius PlayerClass).
type ClassInfo struct {
	ID     int
	Name   string // Mobius constant name, e.g. "FIGHTER"
	Level  int    // 0 starting class, 1 first class change, ...
	Race   int
	Parent int // -1 for a starting class
	Mage   bool
}
