package game

import "time"

func init() { registerFeature("tickstats", setupTickStats) }

// tickStatsInterval is how often the server logs the tick durations of the
// world loop.
const tickStatsInterval = time.Minute

// setupTickStats logs a summary of the world loop's ticks every
// tickStatsInterval, so a load test can tell the tick percentiles.
func setupTickStats(s *Server) error {
	s.loop.Every(tickStatsInterval, s.logTickStats)
	return nil
}

// logTickStats logs the ticks since the last summary: their number, the
// median, 99th percentile and longest duration, how many took longer than
// the tick interval, and the longest delay of a tick's start. Loop-only.
func (s *Server) logTickStats() {
	st := s.loop.TakeTickStats()
	s.log.Info("game world ticks",
		"ticks", st.Ticks, "p50", st.P50, "p99", st.P99, "max", st.Max,
		"over", st.Over, "maxLate", st.MaxLate, "players", s.PlayerCount())
}
