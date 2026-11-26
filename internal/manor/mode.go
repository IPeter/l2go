// Package manor holds the rules of the castle manor (Plan 14 M11k, Mobius
// CastleManorManager and mechanics/siege/manor): the seeds of each castle's
// manor, the daily modes and their transitions, the lists the lords set (the
// seeds a castle sells and the crops it buys, for the running period and the
// next one), the costs, the maintenance transfer and the sowing and
// harvesting odds.
//
// The package is pure: the game (internal/game, manor*.go) holds the state on
// its loop, gives the times (on the server's calendar), the treasuries and the
// random source, and stores what changed.
package manor

import (
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// Mode is the manor's mode (Mobius ManorMode), in Mobius's order.
type Mode int

// The modes. The manor goes APPROVED → MAINTENANCE at the refresh time
// (20:00), MAINTENANCE → MODIFIABLE after the maintenance minutes (6), and
// MODIFIABLE → APPROVED at the approve time (06:00).
const (
	Disabled Mode = iota
	Modifiable
	Maintenance
	Approved
)

// String returns Mobius's enum name.
func (m Mode) String() string {
	switch m {
	case Modifiable:
		return "MODIFIABLE"
	case Maintenance:
		return "MAINTENANCE"
	case Approved:
		return "APPROVED"
	}
	return "DISABLED"
}

// Next returns the mode a change leads to (Mobius changeMode); Disabled
// stays.
func (m Mode) Next() Mode {
	switch m {
	case Approved:
		return Maintenance
	case Maintenance:
		return Modifiable
	case Modifiable:
		return Approved
	}
	return Disabled
}

// Settings are the manor's times and rate (Mobius General.ini Manor and
// Rates.ini RateDropManor).
type Settings struct {
	// RefreshHour and RefreshMin start the maintenance (AltManorRefreshTime,
	// AltManorRefreshMin: 20:00).
	RefreshHour, RefreshMin int
	// ApproveHour and ApproveMin end the modifiable period
	// (AltManorApproveTime, AltManorApproveMin: 06:00).
	ApproveHour, ApproveMin int
	// MaintenanceMin is how long the maintenance lasts (AltManorMaintenanceMin:
	// 6).
	MaintenanceMin int
	// DropRate multiplies the harvests and the lords' limits (RateDropManor:
	// 1).
	DropRate int
}

// DefaultSettings are Mobius's defaults.
func DefaultSettings() Settings {
	return Settings{RefreshHour: 20, ApproveHour: 6, MaintenanceMin: 6, DropRate: 1}
}

// InitialMode is the mode the manor starts in when nothing is stored (the
// CastleManorManager constructor), from the wall-clock hour and minute.
//
// Mobius quirk, kept: the first test needs both the hour past the refresh
// hour and the minute past the maintenance's end, so a start at 21:00-21:05
// (and the same minutes of the later hours) is APPROVED.
func InitialMode(hour, minute int, s Settings) Mode {
	maintenanceMin := s.RefreshMin + s.MaintenanceMin
	switch {
	case (hour >= s.RefreshHour && minute >= maintenanceMin) || hour < s.ApproveHour || (hour == s.ApproveHour && minute <= s.ApproveMin):
		return Modifiable
	case hour == s.RefreshHour && minute >= s.RefreshMin && minute < maintenanceMin:
		return Maintenance
	}
	return Approved
}

// NextChange is scheduleModeChange: when the manor in mode leaves it, seen at
// now. It is the mode's end time today, with now's milliseconds (Java's
// Calendar keeps them), or tomorrow when that is already past, and at least
// one second after now. Disabled never changes: the zero time.
func NextChange(mode Mode, now time.Time, cal calendar.Calendar, s Settings) time.Time {
	var hour, minute int
	switch mode {
	case Modifiable:
		hour, minute = s.ApproveHour, s.ApproveMin
	case Maintenance:
		hour, minute = s.RefreshHour, s.RefreshMin+s.MaintenanceMin
	case Approved:
		hour, minute = s.RefreshHour, s.RefreshMin
	default:
		return time.Time{}
	}
	next := cal.SetTime(now, hour, minute, 0)
	if next.Before(now) {
		next = cal.AddDays(next, 1)
	}
	if next.Sub(now) < time.Second {
		next = now.Add(time.Second)
	}
	return next
}
