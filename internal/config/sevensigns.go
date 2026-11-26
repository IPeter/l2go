package config

import "fmt"

// SevenSigns configures the Seven Signs, the Festival of Darkness and the
// Dimensional Rift (Plan 14, M11l): Mobius Feature.ini's Seven Signs
// section with its festival timings, FestivalOfDarknessWin, and General.ini's
// Dimension Rift section (Rift). Times in milliseconds are Mobius's
// milliseconds. AltSevenSignsLazyUpdate is not here: the game saves at every
// change (decision X4). AltRequireClanCastle is not either: Mobius never
// reads it.
type SevenSigns struct {
	Comment string `json:"_comment,omitempty"`
	// OpenCatacombs and OpenNecropolis let anyone through the dungeon
	// gatekeepers whatever the seals and the period (AltOpenCatacumbs,
	// AltOpenNecropolis: false).
	OpenCatacombs  bool `json:"openCatacombs"`
	OpenNecropolis bool `json:"openNecropolis"`
	// CastleForDawn makes characters after the second class change without a
	// castle pay to join Dawn (AltCastleForDawn: true); CastleForDusk keeps
	// castle owners out of Dusk (AltCastleForDusk: true).
	CastleForDawn bool `json:"castleForDawn"`
	CastleForDusk bool `json:"castleForDusk"`
	// FestivalMinPlayers is the party size a festival needs
	// (AltFestivalMinPlayer: 5).
	FestivalMinPlayers int `json:"festivalMinPlayers"`
	// MaxPlayerContrib caps a player's seal stone contribution score
	// (AltMaxPlayerContrib: 1000000).
	MaxPlayerContrib int `json:"maxPlayerContrib"`
	// The festival's timings in milliseconds: the first cycle's start after
	// the competition begins (AltFestivalManagerStart: 120000), the
	// festival's length (AltFestivalLength: 1080000), the cycle's length
	// (AltFestivalCycleLength: 2280000), and, from the festival's start, the
	// first spawn (AltFestivalFirstSpawn: 120000), the first swarm
	// (AltFestivalFirstSwarm: 300000), the second spawn
	// (AltFestivalSecondSpawn: 540000), the second swarm
	// (AltFestivalSecondSwarm: 720000) and the chests
	// (AltFestivalChestSpawn: 900000).
	FestivalManagerStart int `json:"festivalManagerStart"`
	FestivalLength       int `json:"festivalLength"`
	FestivalCycleLength  int `json:"festivalCycleLength"`
	FestivalFirstSpawn   int `json:"festivalFirstSpawn"`
	FestivalFirstSwarm   int `json:"festivalFirstSwarm"`
	FestivalSecondSpawn  int `json:"festivalSecondSpawn"`
	FestivalSecondSwarm  int `json:"festivalSecondSwarm"`
	FestivalChestSpawn   int `json:"festivalChestSpawn"`
	// The castle gates' P.Def and M.Def multipliers while Dawn or Dusk owns
	// the Seal of Strife (AltDawnGatesPdefMult 1.1, AltDuskGatesPdefMult 0.8,
	// AltDawnGatesMdefMult 1.1, AltDuskGatesMdefMult 0.8).
	DawnGatesPDefMult float64 `json:"dawnGatesPDefMult"`
	DuskGatesPDefMult float64 `json:"duskGatesPDefMult"`
	DawnGatesMDefMult float64 `json:"dawnGatesMDefMult"`
	DuskGatesMDefMult float64 `json:"duskGatesMDefMult"`
	// StrictSevenSigns lets only the seal owners use the Merchant and the
	// Blacksmith of Mammon (StrictSevenSigns: false).
	StrictSevenSigns bool `json:"strictSevenSigns"`
	// The chamberlain's Lord of the Manor's Certificates of Approval:
	// how many each castle sells per competition
	// (SevenSignsDawnTicketQuantity: 300), the price of one
	// (SevenSignsDawnTicketPrice: 1000), how many come in a bundle
	// (SevenSignsDawnTicketBundle: 10) and the item
	// (SevenSignsManorsAgreementId: 6388).
	DawnTicketQuantity int `json:"dawnTicketQuantity"`
	DawnTicketPrice    int `json:"dawnTicketPrice"`
	DawnTicketBundle   int `json:"dawnTicketBundle"`
	ManorsAgreementID  int `json:"manorsAgreementId"`
	// JoinDawnFee is the adena fee to join Dawn without the certificate
	// (SevenSignsJoinDawnFee: 50000).
	JoinDawnFee int `json:"joinDawnFee"`
	// FestivalOfDarknessWin is the clan reputation each clan member of a
	// winning festival party earns for the clan (Feature.ini
	// FestivalOfDarknessWin: 200).
	FestivalOfDarknessWin int `json:"festivalOfDarknessWin"`
	// Rift configures the Dimensional Rift.
	Rift Rift `json:"rift"`
}

// Rift configures the Dimensional Rift (Mobius General.ini, Dimension
// Rift).
type Rift struct {
	Comment string `json:"_comment,omitempty"`
	// MinPartySize is the party size the rift needs, 2-9 (RiftMinPartySize:
	// 2).
	MinPartySize int `json:"minPartySize"`
	// MaxJumps is how many times a party changes rooms before it goes back
	// (MaxRiftJumps: 4).
	MaxJumps int `json:"maxJumps"`
	// SpawnDelay is how long after entering a room its monsters come, in
	// milliseconds (RiftSpawnDelay: 10000).
	SpawnDelay int `json:"spawnDelay"`
	// AutoJumpsDelayMin and AutoJumpsDelayMax bound the time between
	// automatic jumps, in seconds (AutoJumpsDelayMin: 480,
	// AutoJumpsDelayMax: 600).
	AutoJumpsDelayMin int `json:"autoJumpsDelayMin"`
	AutoJumpsDelayMax int `json:"autoJumpsDelayMax"`
	// BossRoomTimeMultiply stretches the time in the boss room
	// (BossRoomTimeMultiply: 1.5).
	BossRoomTimeMultiply float64 `json:"bossRoomTimeMultiply"`
	// The dimension fragments each member pays to enter each rift type
	// (RecruitCost 18, SoldierCost 21, OfficerCost 24, CaptainCost 27,
	// CommanderCost 30, HeroCost 33).
	RecruitCost   int `json:"recruitCost"`
	SoldierCost   int `json:"soldierCost"`
	OfficerCost   int `json:"officerCost"`
	CaptainCost   int `json:"captainCost"`
	CommanderCost int `json:"commanderCost"`
	HeroCost      int `json:"heroCost"`
}

// DefaultSevenSigns returns the Mobius values.
func DefaultSevenSigns() SevenSigns {
	return SevenSigns{
		CastleForDawn:         true,
		CastleForDusk:         true,
		FestivalMinPlayers:    5,
		MaxPlayerContrib:      1000000,
		FestivalManagerStart:  120000,
		FestivalLength:        1080000,
		FestivalCycleLength:   2280000,
		FestivalFirstSpawn:    120000,
		FestivalFirstSwarm:    300000,
		FestivalSecondSpawn:   540000,
		FestivalSecondSwarm:   720000,
		FestivalChestSpawn:    900000,
		DawnGatesPDefMult:     1.1,
		DuskGatesPDefMult:     0.8,
		DawnGatesMDefMult:     1.1,
		DuskGatesMDefMult:     0.8,
		DawnTicketQuantity:    300,
		DawnTicketPrice:       1000,
		DawnTicketBundle:      10,
		ManorsAgreementID:     6388,
		JoinDawnFee:           50000,
		FestivalOfDarknessWin: 200,
		Rift:                  DefaultRift(),
	}
}

// DefaultRift returns the Mobius values.
func DefaultRift() Rift {
	return Rift{
		MinPartySize:         2,
		MaxJumps:             4,
		SpawnDelay:           10000,
		AutoJumpsDelayMin:    480,
		AutoJumpsDelayMax:    600,
		BossRoomTimeMultiply: 1.5,
		RecruitCost:          18,
		SoldierCost:          21,
		OfficerCost:          24,
		CaptainCost:          27,
		CommanderCost:        30,
		HeroCost:             33,
	}
}

// validateSevenSigns checks the sevenSigns section.
func (g GameServer) validateSevenSigns() []error {
	var errs []error
	s := g.SevenSigns
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	for _, f := range []struct {
		name string
		v    int
		min  int
	}{
		{"festivalMinPlayers", s.FestivalMinPlayers, 1}, {"maxPlayerContrib", s.MaxPlayerContrib, 0},
		{"festivalManagerStart", s.FestivalManagerStart, 0}, {"festivalLength", s.FestivalLength, 1},
		{"festivalCycleLength", s.FestivalCycleLength, 1}, {"festivalFirstSpawn", s.FestivalFirstSpawn, 0},
		{"festivalFirstSwarm", s.FestivalFirstSwarm, 0}, {"festivalSecondSpawn", s.FestivalSecondSpawn, 0},
		{"festivalSecondSwarm", s.FestivalSecondSwarm, 0}, {"festivalChestSpawn", s.FestivalChestSpawn, 0},
		{"dawnTicketQuantity", s.DawnTicketQuantity, 0}, {"dawnTicketPrice", s.DawnTicketPrice, 0},
		{"dawnTicketBundle", s.DawnTicketBundle, 1}, {"manorsAgreementId", s.ManorsAgreementID, 1},
		{"joinDawnFee", s.JoinDawnFee, 0}, {"festivalOfDarknessWin", s.FestivalOfDarknessWin, 0},
	} {
		if f.v < f.min {
			add("sevenSigns.%s: %d is below %d", f.name, f.v, f.min)
		}
	}
	if s.FestivalLength >= s.FestivalCycleLength {
		add("sevenSigns.festivalLength: %d is not shorter than festivalCycleLength %d", s.FestivalLength, s.FestivalCycleLength)
	}
	for _, f := range []struct {
		name string
		v    float64
	}{
		{"dawnGatesPDefMult", s.DawnGatesPDefMult}, {"duskGatesPDefMult", s.DuskGatesPDefMult},
		{"dawnGatesMDefMult", s.DawnGatesMDefMult}, {"duskGatesMDefMult", s.DuskGatesMDefMult},
	} {
		if f.v <= 0 {
			add("sevenSigns.%s: %g is not positive", f.name, f.v)
		}
	}
	r := s.Rift
	if r.MinPartySize < 2 || r.MinPartySize > 9 {
		add("sevenSigns.rift.minPartySize: %d is not 2-9", r.MinPartySize)
	}
	for _, f := range []struct {
		name string
		v    int
		min  int
	}{
		{"maxJumps", r.MaxJumps, 1}, {"spawnDelay", r.SpawnDelay, 0}, {"autoJumpsDelayMin", r.AutoJumpsDelayMin, 1},
		{"recruitCost", r.RecruitCost, 0}, {"soldierCost", r.SoldierCost, 0}, {"officerCost", r.OfficerCost, 0},
		{"captainCost", r.CaptainCost, 0}, {"commanderCost", r.CommanderCost, 0}, {"heroCost", r.HeroCost, 0},
	} {
		if f.v < f.min {
			add("sevenSigns.rift.%s: %d is below %d", f.name, f.v, f.min)
		}
	}
	if r.AutoJumpsDelayMax < r.AutoJumpsDelayMin {
		add("sevenSigns.rift.autoJumpsDelayMax: %d is below autoJumpsDelayMin %d", r.AutoJumpsDelayMax, r.AutoJumpsDelayMin)
	}
	if r.BossRoomTimeMultiply <= 0 {
		add("sevenSigns.rift.bossRoomTimeMultiply: %g is not positive", r.BossRoomTimeMultiply)
	}
	return errs
}
