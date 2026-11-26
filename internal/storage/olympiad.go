package storage

import "context"

// The Olympiad and the heroes (Plan 14, M11h, decision X4): what Mobius
// keeps in olympiad_data, olympiad_nobles, olympiad_nobles_eom and heroes
// (one state, saved whole at every change), and in olympiad_fights and
// heroes_diary (logs that only grow).

// Olympiad is the stored state of the Olympiad.
type Olympiad struct {
	// Cycle and Period are Mobius current_cycle and period (0 competition,
	// 1 validation); 0 cycles means none is stored.
	Cycle, Period int32
	// OlympiadEnd, ValidationEnd and NextWeeklyChange are Unix
	// milliseconds, 0 for none.
	OlympiadEnd, ValidationEnd, NextWeeklyChange int64
	// Nobles are the nobles of the running period, NoblesEOM those of the
	// last period's end.
	Nobles, NoblesEOM []OlympiadNoble
	// Heroes are every hero ever (Mobius heroes).
	Heroes []Hero
	// Unclaimed are the noblesse gate passes nobles earned and did not
	// settle yet (Mobius keeps them as the character variable
	// UNCLAIMED_OLYMPIAD_POINTS).
	Unclaimed []UnclaimedPasses
}

// OlympiadNoble is a row of olympiad_nobles.
type OlympiadNoble struct {
	CharID  int32
	ClassID int32
	Name    string
	// Points and the matches done, won, lost and drawn.
	Points, Done, Won, Lost, Drawn int32
}

// Hero is a row of heroes.
type Hero struct {
	CharID, ClassID int32
	Name            string
	Count           int32
	Played, Claimed bool
	Message         string
}

// UnclaimedPasses are a noble's passes to settle.
type UnclaimedPasses struct {
	CharID int32
	Passes int64
}

// OlympiadFight is a row of olympiad_fights: a validated match.
type OlympiadFight struct {
	CharOneID, CharTwoID       int32
	CharOneName, CharTwoName   string
	CharOneClass, CharTwoClass int32
	// Winner is 1 or 2 for the winning side, 0 for a tie.
	Winner int32
	// Start is when the fight started (Unix milliseconds), Time how long
	// it lasted (milliseconds).
	Start, Time int64
	Classed     bool
}

// HeroDiaryEntry is a row of heroes_diary.
type HeroDiaryEntry struct {
	CharID int32
	// Time is Unix milliseconds.
	Time int64
	// Action is 1 a raid boss killed (Param the NPC ID), 2 the hero status
	// gained, 3 a castle taken (Param the castle ID).
	Action, Param int32
}

// OlympiadStore keeps the Olympiad. The game writes it in the background,
// never from the world loop.
type OlympiadStore interface {
	// LoadOlympiad returns the stored state, the zero value when none is
	// stored.
	LoadOlympiad(ctx context.Context) (Olympiad, error)
	// SaveOlympiad replaces the stored state, in one write.
	SaveOlympiad(ctx context.Context, o Olympiad) error
	// AddOlympiadFight stores a fight.
	AddOlympiadFight(ctx context.Context, f OlympiadFight) error
	// OlympiadFights returns a character's fights (on either side) that
	// started before before (Unix milliseconds), oldest first.
	OlympiadFights(ctx context.Context, charID int32, before int64) ([]OlympiadFight, error)
	// AddHeroDiary stores a diary entry.
	AddHeroDiary(ctx context.Context, e HeroDiaryEntry) error
	// HeroDiary returns a character's diary, oldest first.
	HeroDiary(ctx context.Context, charID int32) ([]HeroDiaryEntry, error)
}
