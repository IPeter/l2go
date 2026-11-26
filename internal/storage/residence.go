package storage

import "context"

// Castle state (Plan 14 M11i, decision X4): one document per castle in the
// castles collection, with what Mobius keeps in the castle row and its side
// tables castle_functions, castle_doorupgrade, castle_trapupgrade,
// siege_clans and the hired guards of castle_siege_guards. The owner is the
// clan whose Clan.CastleID names the castle (Mobius clan_data.hasCastle).
// Castles are written with a Batch (BatchWriter), in the same transaction
// as the clans a transition changes (a new owner, reputation). Times are
// Unix milliseconds, 0 for none.
type Castle struct {
	ID             int32
	TaxPercent     int32
	Treasury       int64
	SiegeDate      int64
	RegTimeOver    bool
	RegTimeEnd     int64
	ShowNpcCrest   bool
	TicketBuyCount int32

	Functions    []CastleFunction
	DoorUpgrades []CastleDoorUpgrade
	TrapUpgrades []CastleTrapUpgrade
	// SiegeClans are the clans registered for the next siege.
	SiegeClans []SiegeClan
	// HiredGuards are the mercenaries placed with tickets.
	HiredGuards []HiredGuard
}

// CastleFunction is a castle_functions row.
type CastleFunction struct {
	Type    int32
	Level   int32
	Lease   int32
	Rate    int64
	EndTime int64
}

// CastleDoorUpgrade is a castle_doorupgrade row: a door's HP ratio.
type CastleDoorUpgrade struct {
	DoorID int32
	Ratio  int32
}

// CastleTrapUpgrade is a castle_trapupgrade row: a flame tower's level.
type CastleTrapUpgrade struct {
	TowerIndex int32
	Level      int32
}

// SiegeClan is a siege_clans row: the clan and its side (0 defender, 1
// attacker, 2 defender waiting for approval).
type SiegeClan struct {
	ClanID int32
	Type   int32
}

// HiredGuard is a hired castle_siege_guards row: a mercenary a ticket placed.
type HiredGuard struct {
	NpcID      int32
	X, Y, Z    int32
	Heading    int32
	TicketItem int32 // the ticket's item ID (not in Mobius, which finds it again)
}

// CastleStore reads the castles. Both stores implement it on their
// character store, next to BatchWriter, which writes them (Batch.Castles).
type CastleStore interface {
	// LoadCastles returns the stored castles sorted by ID, nil for none.
	LoadCastles(ctx context.Context) ([]Castle, error)
}
