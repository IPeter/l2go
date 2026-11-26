package storage

import "context"

// Boss state (Plan 13, decision A8): what Mobius keeps in raidboss_spawnlist,
// grandboss_data, grandboss_list and character_raid_points. The spawn points
// and delays stay in the data files, but for a raid boss a GM moved; the
// stores keep only what changes. Times are Unix milliseconds, 0 for none.

// RaidBossState is a raid boss's stored state (the respawn_time, currentHP
// and currentMP columns of Mobius raidboss_spawnlist, and its loc_x, loc_y,
// loc_z and heading once a GM moved the boss).
type RaidBossState struct {
	BossID int32
	// RespawnTime is when a dead boss comes back, 0 while it lives.
	RespawnTime int64
	CurHP       float64
	CurMP       float64
	// Moved is set once a GM moved the boss's spawn (//recall_npc, which
	// rewrites Mobius's raidboss_spawnlist row); X, Y, Z and Heading are
	// then its spawn point instead of the data's.
	Moved            bool
	X, Y, Z, Heading int32
}

// GrandBossState is a grand boss's stored state (a Mobius grandboss_data
// row). Status means what the boss's script says.
type GrandBossState struct {
	BossID           int32
	X, Y, Z, Heading int32
	RespawnTime      int64
	CurHP, CurMP     float64
	Status           int32
}

// BossZonePlayer is a player allowed into a boss zone (a Mobius
// grandboss_list row): the zone lets the player back in after a restart.
type BossZonePlayer struct {
	Zone   int32
	Player int32 // the character's object ID
}

// RaidPoints are the points a character won on one raid boss (a Mobius
// character_raid_points row).
type RaidPoints struct {
	CharID int32
	BossID int32
	Points int32
}

// BossStore keeps the boss state. The game writes it in the background,
// never from the world loop. Lists come back sorted by their keys (boss ID;
// zone and player; character and boss), nil for none.
type BossStore interface {
	// LoadRaidBosses returns the stored raid bosses.
	LoadRaidBosses(ctx context.Context) ([]RaidBossState, error)
	// SaveRaidBosses stores each raid boss, replacing its stored state.
	SaveRaidBosses(ctx context.Context, bosses []RaidBossState) error

	// SeedGrandBosses stores the grand bosses that are not stored yet and
	// leaves the others as they are (Mobius's INSERT IGNORE seed).
	SeedGrandBosses(ctx context.Context, seeds []GrandBossState) error
	// LoadGrandBosses returns the stored grand bosses.
	LoadGrandBosses(ctx context.Context) ([]GrandBossState, error)
	// SaveGrandBossStatus changes the status of a stored grand boss only
	// (Mobius UPDATE_GRAND_BOSS_DATA2); a boss that is not stored is left
	// out.
	SaveGrandBossStatus(ctx context.Context, bossID, status int32) error
	// SaveGrandBosses stores each grand boss whole, replacing its stored
	// state (Mobius UPDATE_GRAND_BOSS_DATA).
	SaveGrandBosses(ctx context.Context, bosses []GrandBossState) error

	// LoadBossZonePlayers returns the players allowed into the boss zones.
	LoadBossZonePlayers(ctx context.Context) ([]BossZonePlayer, error)
	// SaveBossZonePlayers replaces every stored row with rows (Mobius
	// storeMe: DELETE FROM grandboss_list, then the inserts).
	SaveBossZonePlayers(ctx context.Context, rows []BossZonePlayer) error

	// LoadRaidPoints returns every character's raid points.
	LoadRaidPoints(ctx context.Context) ([]RaidPoints, error)
	// SaveRaidPoints stores each row, replacing the stored points of the
	// same character and boss (Mobius REPLACE INTO).
	SaveRaidPoints(ctx context.Context, rows []RaidPoints) error
	// ClearRaidPoints deletes every character's raid points (the weekly
	// reset).
	ClearRaidPoints(ctx context.Context) error
}
