package skills

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/IPeter/l2go/internal/stats"
)

// Config holds the Mobius settings of skills and effects. DefaultConfig
// returns the Mobius defaults (Player.ini and General.ini).
type Config struct {
	// MaxBuffAmount and MaxDanceAmount are the buff and dance limits
	// (MaxBuffAmount 20, MaxDanceAmount 12). Divine Inspiration (skill
	// 1405) adds its level to the buff limit.
	MaxBuffAmount, MaxDanceAmount int
	// EffectTickRatio is the length of an effect tick in milliseconds
	// (EffectTickRatio, 666): an effect with ticks="3" acts every 1998 ms,
	// and its amount is scaled by 3 × 0.666.
	EffectTickRatio int
	// MagicFailures lets magic damage fail with half or no damage
	// (MagicFailures, true).
	MagicFailures bool
	// MinAbnormalStateSuccessRate and MaxAbnormalStateSuccessRate bound the
	// landing chance of effects of skills without minChance and maxChance
	// (0 and 25).
	MinAbnormalStateSuccessRate, MaxAbnormalStateSuccessRate int
	// ShowEffectMessagesOnLogin sends "The effects of S1 flow through you."
	// for the effects restored at login (ShowEffectMessagesOnLogin, false).
	ShowEffectMessagesOnLogin bool
	// DebuffDurationUsesResists shortens debuffs by the target's
	// resistances (General.ini DebuffDurationUsesResists, false).
	DebuffDurationUsesResists bool
	// MagicSuccessBySkillMagicLevel compares the target's level with the
	// skill's magic level instead of the caster's level in the magic
	// success roll (CalculateMagicSuccessBySkillMagicLevel, true).
	MagicSuccessBySkillMagicLevel bool
	// DanceConsumeAdditionalMP makes each active dance raise the MP cost of
	// the next one by half its base cost (DanceConsumeAdditionalMP, true).
	DanceConsumeAdditionalMP bool
	// RandomizePhysicalSkillDamage and RandomizeMagicalSkillDamage multiply
	// skill damage by the weapon's random damage factor
	// (RandomizePhysicalSkillDamage and RandomizeMagicalSkillDamage, true).
	// Physical skills draw it a second time unconditionally, as Mobius does.
	RandomizePhysicalSkillDamage, RandomizeMagicalSkillDamage bool
	// ShieldBlocks is AltShieldBlocks (false): a blocked hit loses the
	// shield defence from its damage instead of adding it to P.Def.
	ShieldBlocks bool
	// PerfectShieldBlockRate is AltPerfectShieldBlockRate (10).
	PerfectShieldBlockRate int
	// CancelCastByHit and CancelBowByHit are AltGameCancelByHit ("cast"):
	// which actions a hit can break.
	CancelCastByHit, CancelBowByHit bool
	// GMSkillRestriction makes GMs meet skill conditions like everyone else
	// (General.ini GMSkillRestriction, true).
	GMSkillRestriction bool
	// PartyRange is the reach of party skills that name no affect range
	// (AltPartyRange, 1500).
	PartyRange int
	// StoreDances and StoreToggles keep dances and toggles among the
	// effects a player saves (AltStoreDances true, AltStoreToggles false).
	StoreDances, StoreToggles bool
}

// DefaultConfig returns the Mobius defaults.
func DefaultConfig() Config {
	return Config{
		MaxBuffAmount:                 20,
		MaxDanceAmount:                12,
		EffectTickRatio:               666,
		MagicFailures:                 true,
		MinAbnormalStateSuccessRate:   0,
		MaxAbnormalStateSuccessRate:   25,
		ShowEffectMessagesOnLogin:     false,
		DebuffDurationUsesResists:     false,
		MagicSuccessBySkillMagicLevel: true,
		DanceConsumeAdditionalMP:      true,
		RandomizePhysicalSkillDamage:  true,
		RandomizeMagicalSkillDamage:   true,
		ShieldBlocks:                  false,
		PerfectShieldBlockRate:        10,
		CancelCastByHit:               true,
		CancelBowByHit:                false,
		GMSkillRestriction:            true,
		PartyRange:                    1500,
		StoreDances:                   true,
		StoreToggles:                  false,
	}
}

// ErrInvalid is wrapped by the errors this package returns for bad input.
var ErrInvalid = errors.New("invalid skill input")

// Validate checks that the numbers make sense.
func (c *Config) Validate() error {
	switch {
	case c.MaxBuffAmount < 0 || c.MaxDanceAmount < 0:
		return fmt.Errorf("skills: %w: buff limits %d and %d must not be negative", ErrInvalid, c.MaxBuffAmount, c.MaxDanceAmount)
	case c.EffectTickRatio <= 0:
		return fmt.Errorf("skills: %w: effect tick ratio %d must be positive", ErrInvalid, c.EffectTickRatio)
	case c.MinAbnormalStateSuccessRate > c.MaxAbnormalStateSuccessRate:
		return fmt.Errorf("skills: %w: abnormal state success rate %d-%d", ErrInvalid, c.MinAbnormalStateSuccessRate, c.MaxAbnormalStateSuccessRate)
	case c.PerfectShieldBlockRate < 0 || c.PerfectShieldBlockRate > 100:
		return fmt.Errorf("skills: %w: perfect shield block rate %d, want 0 to 100", ErrInvalid, c.PerfectShieldBlockRate)
	case c.PartyRange < 0:
		return fmt.Errorf("skills: %w: party range %d must not be negative", ErrInvalid, c.PartyRange)
	}
	return nil
}

// TimerID identifies a timer of a Scheduler. Zero is never a valid ID.
type TimerID uint64

// Scheduler is the world loop's clock and timer heap as skills use them
// (Plan 10 K7). Functions run on the loop, like everything else here.
type Scheduler interface {
	// After runs fn once d has passed.
	After(d time.Duration, fn func()) TimerID
	// Every runs fn every d, the first time once d has passed (a fixed-rate
	// schedule).
	Every(d time.Duration, fn func()) TimerID
	// Cancel stops a timer; it reports whether the timer was pending.
	Cancel(id TimerID) bool
	// GameTicks returns the game ticks (100 ms) of the loop clock, Mobius
	// GameTimeTaskManager.getGameTicks. Only differences are used.
	GameTicks() int64
}

// Env is what the skills of one server share: the random source, the
// timers, the settings and the world queries. Build it once and pass the
// same pointer everywhere.
type Env struct {
	// Rand is the random source, with Mobius Rnd semantics.
	Rand stats.Rand
	// Sched runs the buff clocks, effect ticks and icon updates.
	Sched Scheduler
	// World answers the target and party queries; it may be nil in tests
	// that use none.
	World World
	// Config holds the settings.
	Config Config
	// Log receives the warnings Mobius logs (a passive with an abnormal
	// type, ...); nil means slog.Default().
	Log *slog.Logger
}

// NewEnv checks cfg and returns an Env.
func NewEnv(rnd stats.Rand, sched Scheduler, world World, cfg Config, log *slog.Logger) (*Env, error) {
	if rnd == nil || sched == nil {
		return nil, fmt.Errorf("skills: %w: an Env needs a random source and a scheduler", ErrInvalid)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Env{Rand: rnd, Sched: sched, World: world, Config: cfg, Log: log}, nil
}

func (e *Env) log() *slog.Logger {
	if e.Log != nil {
		return e.Log
	}
	return slog.Default()
}

// tickPeriod is the period of an effect with the given ticks.
func (e *Env) tickPeriod(ticks int) time.Duration {
	return time.Duration(ticks*e.Config.EffectTickRatio) * time.Millisecond
}

// World answers what the target types and a few conditions ask about the
// world around a creature. internal/game implements it over its regions,
// parties and geodata.
type World interface {
	// VisibleInRange returns the creatures visible to c within rng (3D
	// distance from c, collision ignored), c excluded, in the order the
	// world iterates them (Mobius World.getVisibleObjectsInRange).
	VisibleInRange(c Creature, rng int) []Creature
	// Visible returns every creature visible to c, c excluded (Mobius
	// World.forEachVisibleObject).
	Visible(c Creature) []Creature
	// CanSee reports line of sight (GeoEngine.canSeeTarget).
	CanSee(from, to Creature) bool
	// Party returns the members of c's party, leader first, or nil when c
	// is in none. A summon answers with its owner's party.
	Party(c Creature) []Creature
	// SameParty reports whether a and b are in the same party (Mobius
	// compares the leaders). Summons count as their owners.
	SameParty(a, b Creature) bool
	// Summon returns the summon or pet of a player, nil without one.
	Summon(owner Creature) Creature
	// OldCorpse reports whether a dead monster is too old to be consumed
	// by caster's corpse skills (Attackable.isOldCorpse with
	// CorpseConsumeSkillAllowedTimeBeforeDecay); it tells caster why.
	OldCorpse(corpse, caster Creature) bool
	// IsNight reports game-time night.
	IsNight() bool
}

// ZoneKind is a kind of zone the skills ask about (Mobius ZoneId).
type ZoneKind uint8

// The zone kinds.
const (
	ZonePvP ZoneKind = iota
	ZonePeace
	ZoneSiege
)

// Location is a position in the world.
type Location struct {
	X, Y, Z int
}

// ShotType is a kind of shot a weapon can be charged with (Mobius
// ShotType).
type ShotType uint8

// The shot types.
const (
	Soulshot ShotType = iota
	Spiritshot
	BlessedSpiritshot
	FishSoulshot
)

// TeleportWhere is where an escape skill sends its target (Mobius
// TeleportWhereType).
type TeleportWhere uint8

// The escape destinations.
const (
	TeleportCastle TeleportWhere = iota
	TeleportClanHall
	TeleportSiegeFlag
	TeleportTown
)

var teleportWhereNames = [...]string{"CASTLE", "CLANHALL", "SIEGEFLAG", "TOWN"}

// String returns the Mobius name of w.
func (w TeleportWhere) String() string {
	if int(w) < len(teleportWhereNames) {
		return teleportWhereNames[w]
	}
	return fmt.Sprintf("TeleportWhere(%d)", uint8(w))
}

// ParseTeleportWhere returns the destination with the Mobius name.
func ParseTeleportWhere(name string) (TeleportWhere, bool) {
	for i, n := range teleportWhereNames {
		if n == name {
			return TeleportWhere(i), true
		}
	}
	return 0, false
}
