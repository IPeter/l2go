package scripting_test

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The room spawns of scripts/lib/four_sepulchers.lua are Mobius's
// FourSepulchers.xml, row for row. The test needs the Mobius checkout next
// to the repository (or L2J_MOBIUS_INTERLUDE) and is skipped without it.
func TestFourSepulchersSpawnsMatchMobius(t *testing.T) {
	root := os.Getenv("L2J_MOBIUS_INTERLUDE")
	if root == "" {
		root = filepath.Join("..", "..", "..", "..", "L2J_Mobius", "L2J_Mobius_CT_0_Interlude")
	}
	raw, err := os.ReadFile(filepath.Join(root, "dist", "game", "data", "scripts", "ai", "areas",
		"ImperialTomb", "FourSepulchers", "FourSepulchers.xml"))
	if err != nil {
		t.Skipf("no Mobius FourSepulchers.xml: %v", err)
	}
	var doc struct {
		Spawns []struct {
			Sepulcher int `xml:"sepulcherId,attr"`
			Wave      int `xml:"wave,attr"`
			Npc       int `xml:"npcId,attr"`
			X         int `xml:"x,attr"`
			Y         int `xml:"y,attr"`
			Z         int `xml:"z,attr"`
			Heading   int `xml:"heading,attr"`
		} `xml:"spawn"`
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, s := range doc.Spawns {
		want = append(want, fmt.Sprintf("{ %d, %d, %d, %d, %d, %d, %d },", s.Sepulcher, s.Wave, s.Npc, s.X, s.Y, s.Z, s.Heading))
	}
	lua, err := os.ReadFile(filepath.Join(scriptsDir, "lib", "four_sepulchers.lua"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for line := range strings.Lines(string(lua)) {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "{ ") {
			got = append(got, line)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%d rows, Mobius has %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %s, Mobius %s", i+1, got[i], want[i])
		}
	}
}
