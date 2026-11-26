package config

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

// Events configures the seasonal events (Plan 14, M11n): Mobius's
// LongTimeEvent scripts Christmas, L2Day and HeavyMedal, whose dates come
// from their config.xml (data/events/longtimeevents.json). Every one of
// those dates is in the past, so nothing runs unless Dates moves an event.
type Events struct {
	Comment string `json:"_comment,omitempty"`
	// ItemMaxLevelDifference: a player more than this many levels above a
	// monster gets no event drops from it (Mobius Rates.ini
	// EventItemMaxLevelDifference).
	ItemMaxLevelDifference int `json:"eventItemMaxLevelDifference"`
	// Dates replace the dates of an event, by script name ("L2Day").
	Dates map[string]EventDates `json:"dates,omitempty"`
}

// EventDates are an event's period and drop period in Mobius's config.xml
// form: "dd MM yyyy-dd MM yyyy", or "dd MM-dd MM" for every year (in the
// server's time zone, from the start of each day).
type EventDates struct {
	// Active is the event period.
	Active string `json:"active"`
	// DropPeriod is when monsters drop the event items; empty means the
	// whole event period. It must lie within Active.
	DropPeriod string `json:"dropPeriod,omitempty"`
}

// DefaultEvents returns the Mobius defaults: no date changes.
func DefaultEvents() Events {
	return Events{ItemMaxLevelDifference: 9}
}

// eventPeriod is the syntax of an event period; the dates themselves are
// checked when the game reads them.
var eventPeriod = regexp.MustCompile(`^(\d\d \d\d \d{4}-\d\d \d\d \d{4}|\d\d \d\d-\d\d \d\d)$`)

// validateEvents checks the events section.
func (g GameServer) validateEvents() []error {
	var errs []error
	e := g.Events
	if e.ItemMaxLevelDifference < 0 {
		errs = append(errs, fmt.Errorf("events.eventItemMaxLevelDifference: %d is negative", e.ItemMaxLevelDifference))
	}
	for _, name := range slices.Sorted(maps.Keys(e.Dates)) {
		d := e.Dates[name]
		if !eventPeriod.MatchString(d.Active) {
			errs = append(errs, fmt.Errorf("events.dates.%s.active: %q is not dd MM yyyy-dd MM yyyy or dd MM-dd MM", name, d.Active))
		}
		if d.DropPeriod != "" && !eventPeriod.MatchString(d.DropPeriod) {
			errs = append(errs, fmt.Errorf("events.dates.%s.dropPeriod: %q is not dd MM yyyy-dd MM yyyy or dd MM-dd MM", name, d.DropPeriod))
		}
	}
	return errs
}
