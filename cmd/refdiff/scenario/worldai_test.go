package scenario

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/testclient"
)

// TestWorldScenarioSpots checks the spots of the Milestone 10 scenarios
// against the Mobius data: next to the door and the town map (within the
// 150 units of a click), within the 1,000 units of boarding from the ferry's
// dock, next to the Cruma Tower spawn.
func TestWorldScenarioSpots(t *testing.T) {
	spots := []struct {
		name  string
		spot  testclient.Position
		at    testclient.Position
		reach float64
	}{
		{"Gludin clan hall door", spotGludinDoor, testclient.Position{X: -84495, Y: 155208, Z: -3150}, 150},
		{"Talking Island town map", spotTownMap, testclient.Position{X: -82873, Y: 244808, Z: -3717}, 150},
		{"Talking Island ferry dock", spotPier, testclient.Position{X: -96622, Y: 261660, Z: -3610}, 1000},
		{"Cruma Tower Porta", spotCruma, testclient.Position{X: 12530, Y: 117248, Z: -12084}, 150},
	}
	for _, sp := range spots {
		if d := sp.spot.DistanceTo(sp.at); d > sp.reach {
			t.Errorf("the spot of the %s is %.0f units from it, want at most %.0f", sp.name, d, sp.reach)
		}
	}
	// The raid spot is far from the spawner's Kutus (-54416, 146480): out of
	// sight of his group.
	if d := spotRaid.DistanceTo(testclient.Position{X: -54416, Y: 146480, Z: -2887}); d < 20000 {
		t.Errorf("the raid spot is %.0f units from the spawner's Kutus", d)
	}
}

func TestWorldScenariosAreGM(t *testing.T) {
	for _, sc := range worldScenarios {
		if !sc.GM {
			t.Errorf("%s is not marked GM", sc.Name)
		}
		got, ok := Find(sc.Name)
		if !ok || got.Name != sc.Name {
			t.Errorf("Find(%s) = %+v, %v", sc.Name, got, ok)
		}
	}
	// The ferry's own ignores do not leak into the shared NPC noise.
	if slices.Contains(npcNoise, "CreatureSay") {
		t.Errorf("npcNoise = %q", npcNoise)
	}
	// cruma-heights compares the heights.
	sc, _ := Find("cruma-heights")
	if slices.Contains(sc.Ignore, "NpcInfo.Z") || slices.Contains(sc.Ignore, "*.Z") {
		t.Errorf("cruma-heights ignores the heights: %q", sc.Ignore)
	}
}
