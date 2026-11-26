package storage

import "context"

// Manor state (Plan 14 M11k, decision X4): one document per castle in the
// manor collection with the seeds the castle sells and the crops it buys in
// the running and the next period (Mobius castle_manor_production and
// castle_manor_procure), and the manor's mode with its next change (which
// Mobius does not store; it takes the mode from the clock at start). Both
// are written with a Batch (BatchWriter), in the transaction of the
// castles' treasuries and the clan warehouse items a change moves. Times are
// Unix milliseconds, 0 for none.
type ManorCastle struct {
	CastleID int32

	Production, ProductionNext []SeedProduction
	Procure, ProcureNext       []CropProcure
}

// SeedProduction is a castle_manor_production row: a seed the castle sells.
type SeedProduction struct {
	SeedID      int32
	Amount      int64
	Price       int64
	StartAmount int64
}

// CropProcure is a castle_manor_procure row: a crop the castle buys, paid
// with reward type RewardType (1 or 2).
type CropProcure struct {
	CropID      int32
	Amount      int64
	Price       int64
	StartAmount int64
	RewardType  int32
}

// ManorState is the manor's mode (Mobius ManorMode: 0 disabled, 1
// modifiable, 2 maintenance, 3 approved) and when it changes next.
type ManorState struct {
	Mode       int32
	NextChange int64
}

// ManorStore reads the manor. Both stores implement it on their character
// store, next to BatchWriter, which writes it (Batch.ManorCastles and
// Batch.ManorState).
type ManorStore interface {
	// LoadManor returns the stored state (ok false when none is stored) and
	// the castles' lists sorted by castle ID, nil for none.
	LoadManor(ctx context.Context) (state ManorState, ok bool, castles []ManorCastle, err error)
}
