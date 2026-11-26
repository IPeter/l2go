package game

import (
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// The server's calendar (Plan 14, decision X5): the systems that run on
// wall-clock time (the fishing championship now; sieges, the Olympiad,
// Seven Signs, the manor and events later) compute their next transition
// with it from the loop's clock (Server.Now), save it, and replay a missed
// one when the server starts.

// calendarOf returns the calendar of Options.TimeZone: time.Local when it
// is nil, as Mobius uses the JVM's default zone.
func calendarOf(loc *time.Location) calendar.Calendar {
	if loc == nil {
		loc = time.Local
	}
	return calendar.New(loc)
}

// Calendar returns the server's calendar: its time zone (Options.TimeZone)
// and Java's weeks from Sunday. Use it with Server.Now. Safe on any
// goroutine.
func (s *Server) Calendar() calendar.Calendar { return s.calendar }
