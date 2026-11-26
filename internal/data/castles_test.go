package data

import (
	"testing"
	"time"
)

// TestLoadCastles checks the generated castle data against the Mobius rows
// it comes from.
func TestLoadCastles(t *testing.T) {
	c, err := LoadCastles("../../data")
	if err != nil {
		t.Fatalf("LoadCastles: %v", err)
	}
	if len(c.Castles) != 9 || len(c.Guards) != 3689 || len(c.Tickets) != 9*52 {
		t.Fatalf("%d castles, %d guards, %d tickets; want 9, 3689, 468", len(c.Castles), len(c.Guards), len(c.Tickets))
	}
	gludio, ok := c.Castle(1)
	if !ok || gludio.Name != "Gludio" || gludio.SiegeWeekday() != time.Sunday || gludio.SiegeHour != 16 ||
		len(gludio.ControlTowers) != 3 || len(gludio.FlameTowers) != 2 || gludio.MaxMercenaries != 100 {
		t.Errorf("Gludio = %+v", gludio)
	}
	// GludioFlameTower1=-18154,107591,-2560,13004,70017,70019
	if ft := gludio.FlameTowers[0]; ft.X != -18154 || ft.NpcID != 13004 || len(ft.Zones) != 2 || ft.Zones[1] != 70019 {
		t.Errorf("Gludio's first flame tower = %+v", ft)
	}
	if dion, _ := c.Castle(2); dion.SiegeHour != 20 || dion.MaxMercenaries != 150 {
		t.Errorf("Dion = %+v", dion)
	}
	// (1,1,35085,-18460,113107,-2759,15608,10800,0)
	if g := c.GuardsOf(1)[0]; g.ID != 1 || g.NpcID != 35085 || g.X != -18460 || g.Heading != 15608 || g.RespawnDelay != 10800 {
		t.Errorf("Gludio's first guard = %+v", g)
	}
	if tk, ok := c.Ticket(3960); !ok || tk.CastleID != 1 || tk.NpcID != 35010 || tk.MaxPerType != 10 {
		t.Errorf("ticket 3960 = %+v", tk)
	}
	if tk, ok := c.Ticket(7972); !ok || tk.CastleID != 9 || tk.NpcID != 35061 || tk.Index != 51 {
		t.Errorf("ticket 7972 = %+v", tk)
	}
}
