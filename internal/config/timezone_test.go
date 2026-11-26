package config

import (
	"strings"
	"testing"
	"time"
)

// world.timeZone: empty is the local zone, an IANA name its zone, and an
// unknown name a validation error.
func TestWorldTimeZone(t *testing.T) {
	if tz := Default().GameServers[0].World.TimeZone; tz != "" {
		t.Errorf("default timeZone = %q, want empty", tz)
	}
	loc, err := DefaultWorld().Location()
	if err != nil || loc != time.Local {
		t.Errorf("empty timeZone: %v, %v; want time.Local", loc, err)
	}
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "world": {"timeZone": "UTC"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	w := cfg.GameServers[0].World
	if w.TimeZone != "UTC" || w.TickInterval != DefaultTickInterval {
		t.Errorf("world = %+v, want timeZone UTC and the other defaults", w)
	}
	if loc, err := w.Location(); err != nil || loc.String() != "UTC" {
		t.Errorf("UTC: %v, %v", loc, err)
	}
	cfg = Default()
	cfg.GameServers[0].World.TimeZone = "Mars/Olympus_Mons"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].world.timeZone") {
		t.Errorf("Validate = %v, want the timeZone error with its path", err)
	}
	if _, err := cfg.GameServers[0].World.Location(); err == nil {
		t.Error("Location of an unknown zone did not fail")
	}
}
