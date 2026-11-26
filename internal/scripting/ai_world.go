package scripting

import "time"

// The world as the AI scripts see it (Plan 13, Phase 10.6): NPC control,
// world scans, zones, doors, game time, grand boss state and cinematics.
// Each part is an optional interface the QuestWorld (or, for data that
// belongs to one NPC, the NpcRef) may implement. Without it the Lua calls
// that need it raise "this server cannot do that yet", except where a
// method says what it does instead. Everything runs on the world loop,
// while a handler runs.

// NpcControl is the NPC control the AI scripts use beyond NpcActions: the
// core AI, movement modes, invulnerability, intentions and the hate list
// (Mobius Npc, Attackable, Creature and their AI). A target is a PlayerRef
// or an NpcRef.
type NpcControl interface {
	// NpcSetWalking is Mobius setWalking; NpcIsRunning is isRunning.
	NpcSetWalking(npc NpcRef)
	NpcIsRunning(npc NpcRef) bool
	// NpcDisableCoreAI is Mobius disableCoreAI(value): the NPC's own AI
	// stops thinking; scripts move it.
	NpcDisableCoreAI(npc NpcRef, disabled bool)
	NpcIsCoreAIDisabled(npc NpcRef) bool
	// NpcSetImmobilized is Mobius setImmobilized.
	NpcSetImmobilized(npc NpcRef, immobilized bool)
	// NpcSetInvul is Mobius setInvul; NpcIsInvul isInvul.
	NpcSetInvul(npc NpcRef, invul bool)
	NpcIsInvul(npc NpcRef) bool
	// NpcSetMortal is Mobius setMortal: an immortal NPC keeps 1 HP.
	NpcSetMortal(npc NpcRef, mortal bool)
	NpcIsMortal(npc NpcRef) bool
	// NpcSetRandomWalking is Mobius setRandomWalking.
	NpcSetRandomWalking(npc NpcRef, enabled bool)
	// NpcReturnHome is Mobius Attackable.returnHome: back to the spawn
	// point, hate cleared.
	NpcReturnHome(npc NpcRef)
	// NpcIntention returns the AI intention by its Mobius name: "IDLE",
	// "ACTIVE", "REST", "ATTACK", "CAST", "MOVE_TO", "FOLLOW", "PICK_UP" or
	// "INTERACT".
	NpcIntention(npc NpcRef) string
	// NpcSetIntention is Mobius getAI().setIntention(intention, arg): arg
	// is nil, a target (PlayerRef or NpcRef) or a *Location.
	NpcSetIntention(npc NpcRef, intention string, arg any)
	// NpcFollow is getAI().startFollow(target); NpcStopFollow stopFollow.
	NpcFollow(npc NpcRef, target any)
	NpcStopFollow(npc NpcRef)
	// NpcStopMoving is Mobius getAI().clientStopMoving(null).
	NpcStopMoving(npc NpcRef)
	// NpcIsMoving is Mobius isMoving; NpcIsInCombat isInCombat;
	// NpcIsCastingNow isCastingNow.
	NpcIsMoving(npc NpcRef) bool
	NpcIsInCombat(npc NpcRef) bool
	NpcIsCastingNow(npc NpcRef) bool
	// NpcHateList returns the NPC's aggro list, most hated first (Mobius
	// getAggroList). Empty for NPCs that cannot fight.
	NpcHateList(npc NpcRef) []HateEntry
	// NpcMostHated is Mobius getMostHated, nil for none.
	NpcMostHated(npc NpcRef) any
	// NpcClearAggroList is Mobius clearAggroList; NpcStopHating is
	// stopHating(target); NpcReduceHate is reduceHate(target, amount).
	NpcClearAggroList(npc NpcRef)
	NpcStopHating(npc NpcRef, target any)
	NpcReduceHate(npc NpcRef, target any, amount int64)
	// NpcSetHeading turns the NPC (Mobius setHeading plus the broadcast).
	NpcSetHeading(npc NpcRef, heading int)
}

// HateEntry is one row of an NPC's aggro list (Mobius AggroInfo).
type HateEntry struct {
	// Target is the hated PlayerRef or NpcRef.
	Target any
	Hate   int64
	Damage int64
}

// NpcMinions are leaders and minions (Mobius MinionList, Monster).
type NpcMinions interface {
	// NpcLeader returns the NPC's leader, nil for none (Mobius getLeader).
	NpcLeader(npc NpcRef) NpcRef
	// NpcMinionsOf returns the leader's living minions (Mobius
	// getMinionList().getSpawnedMinions()).
	NpcMinionsOf(npc NpcRef) []NpcRef
	// NpcAddMinion spawns a minion of the template for the leader (Mobius
	// addMinion(master, minionId)) and returns it, or nil.
	NpcAddMinion(leader NpcRef, npcID int) NpcRef
}

// NpcSpawnInfo is the part of an NpcRef behind its spawn: the spawn point
// and the spawn's AI data (Mobius getSpawn().getLocation(), getAIValue,
// hasAIValue).
type NpcSpawnInfo interface {
	// SpawnLocation is the spawn point, false for an NPC without a spawn.
	SpawnLocation() (Location, bool)
	// AIValue is a value of the spawn's AI data (Mobius
	// NpcPersonalAIData), false when it has none of that name.
	AIValue(name string) (int, bool)
}

// NpcTemplateParams is the part of an NpcRef behind its template
// parameters (Mobius getParameters(), StatSet values as strings).
type NpcTemplateParams interface {
	TemplateParam(name string) (string, bool)
}

// ChestActions are Mobius Chest's methods (treasure chests and mimics).
type ChestActions interface {
	ChestIsBox(npc NpcRef) bool
	ChestIsInteracted(npc NpcRef) bool
	ChestSetInteracted(npc NpcRef)
	// ChestTreasureBomb is Mobius Chest.treasureBomb: the box explodes.
	ChestTreasureBomb(npc NpcRef)
}

// SkillTargets is the skill data's target type (Mobius
// skill.getTargetType().name(), e.g. "ONE", "UNLOCKABLE").
type SkillTargets interface {
	SkillTargetType(skill Skill) string
}

// PlayerHP is the part of a PlayerRef behind player:hp() and
// player:max_hp() (Mobius getCurrentHp, getMaxHp). NpcStatus has the same
// methods for NPCs.
type PlayerHP interface {
	CurrentHP() float64
	MaxHPValue() float64 // not MaxHP: the game's Player.MaxHP returns int32
}

// PlayerSummon is the part of a PlayerRef behind player:summon(): the
// summon or pet, a PlayerRef or NpcRef, or nil (Mobius getSummon).
type PlayerSummon interface {
	SummonObject() any
}

// WorldScan finds objects around a point or by template. A nil result is
// an empty one.
type WorldScan interface {
	// NpcsAround returns the NPCs within radius of obj, obj itself
	// excluded (Mobius World.getVisibleObjectsInRange(obj, Npc.class,
	// radius)). radius 0 is the visible range.
	NpcsAround(obj any, radius int) []NpcRef
	// PlayersAround is the same for players.
	PlayersAround(obj any, radius int) []PlayerRef
	// NpcsByID returns the NPCs of the template in the world (Mobius
	// SpawnTable/World scans; restart-safe scripts find their boss here).
	NpcsByID(npcID int) []NpcRef
}

// WorldGeo is the geodata as scripts see it.
type WorldGeo interface {
	// ValidLocation is Mobius GeoEngine.getValidLocation: the last point of
	// the straight walk from from to to. Without WorldGeo the destination is
	// returned, as Mobius does without geodata.
	ValidLocation(from, to Location) Location
}

// ZoneControl is the zones as boss scripts use them (Mobius ZoneType and
// BossZone).
type ZoneControl interface {
	// ZonePlayers returns the players inside the zone.
	ZonePlayers(zoneID int) []PlayerRef
	// ZoneIsInside reports whether obj, a PlayerRef, an NpcRef or a
	// Location, is inside the zone.
	ZoneIsInside(zoneID int, obj any) bool
	// ZoneMovePlayers is BossZone.movePlayersTo(loc).
	ZoneMovePlayers(zoneID int, loc Location)
	// ZoneOustAll is BossZone.oustAllPlayers.
	ZoneOustAll(zoneID int)
	// ZoneAllowEntry is BossZone.allowPlayerEntry(player, duration).
	ZoneAllowEntry(zoneID int, p PlayerRef, d time.Duration)
	// ZoneRemovePlayer is BossZone.removePlayer(player).
	ZoneRemovePlayer(zoneID int, p PlayerRef)
}

// DoorControl is the doors (Mobius DoorData, Door).
type DoorControl interface {
	// OpenDoor and CloseDoor are openMe and closeMe; false for an unknown
	// door.
	OpenDoor(id int) bool
	CloseDoor(id int) bool
	// DoorIsOpen reports the door's state; ok is false for an unknown door.
	DoorIsOpen(id int) (open, ok bool)
}

// GameClock is the game time (Mobius GameTimeTaskManager).
type GameClock interface {
	// GameTicks is getGameTicks; GameTime is getGameTime, the minutes since
	// game midnight (0 to 1439).
	GameTicks() int64
	GameTime() int
	// IsNight is isNight (game hour < 6).
	IsNight() bool
}

// Grand boss statuses as Mobius scripts number them; scripts may use
// others.
const (
	BossAlive    = 0
	BossDead     = 1
	BossFighting = 2
	BossWaiting  = 3
)

// BossInfo is a grand boss's saved StatSet (Mobius grandboss_data).
type BossInfo struct {
	// RespawnTime is the respawn time in Unix milliseconds, 0 for none.
	RespawnTime int64
	X, Y, Z     int
	Heading     int
	HP, MP      float64
}

// GrandBossStore is Mobius GrandBossManager: the saved status and info of
// each grand boss and the live boss.
type GrandBossStore interface {
	// BossStatus is getStatus; false for an unknown boss.
	BossStatus(id int) (int, bool)
	// SetBossStatus is setStatus: saved at once.
	SetBossStatus(id, status int)
	// BossInfo is getStatSet; false for an unknown boss.
	BossInfo(id int) (BossInfo, bool)
	// SetBossInfo is setStatSet.
	SetBossInfo(id int, info BossInfo)
	// LiveBoss is getBoss: the boss in the world, or nil.
	LiveBoss(id int) NpcRef
	// AddBoss is addBoss: npc is the live boss of its template.
	AddBoss(npc NpcRef)
}

// Audience is who sees a cinematic: one player, the players around an
// object, or the players inside a zone. Exactly one is set.
type Audience struct {
	Player PlayerRef
	Around any // a PlayerRef or NpcRef
	Zone   int
}

// Camera is a SpecialCamera, in Mobius's argument order.
type Camera struct {
	Force, Angle1, Angle2, Time, Range, Duration int
	RelYaw, RelPitch, Wide, RelAngle, Unk        int
}

// Cinematics are the packets boss scripts show (SpecialCamera,
// Earthquake, PlaySound, ExShowScreenMessage).
type Cinematics interface {
	// SpecialCamera points the audience's camera at target.
	SpecialCamera(to Audience, target any, cam Camera)
	// Earthquake shakes the screens around loc.
	Earthquake(to Audience, loc Location, intensity, duration int)
	// PlaySoundFrom plays a sound; source (PlayerRef or NpcRef, may be nil)
	// is where it comes from (PlaySound type 1).
	PlaySoundFrom(to Audience, sound string, source any)
	// ScreenMessage is ExShowScreenMessage(text, position, ms).
	ScreenMessage(to Audience, text string, position, ms int)
}
