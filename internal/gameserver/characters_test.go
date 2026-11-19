package gameserver

import (
	"strings"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

func TestCheckName(t *testing.T) {
	const (
		ok       = serverpackets.CharCreateFailReason(-1)
		tooLong  = serverpackets.CharCreateFailNameTooLong
		badChars = serverpackets.CharCreateFailIncorrectName
	)
	tests := map[string]serverpackets.CharCreateFailReason{
		"":                            tooLong,
		"A":                           ok,
		"7":                           ok,
		"Frostwind":                   ok,
		"Player2026":                  ok,
		strings.Repeat("a", 16):       ok,
		strings.Repeat("a", 17):       tooLong,
		strings.Repeat("é", 16):       ok, // 16 UTF-16 units, 32 UTF-8 bytes
		"Hérőke":                      ok, // Java isLetterOrDigit accepts any letter
		"名前":                          ok,
		"Bad Name":                    badChars,
		"Bad_Name":                    badChars,
		"Bad-Name":                    badChars,
		"Name!":                       badChars,
		"Smile😀":                      badChars, // outside the BMP: surrogates are not letters
		"𝐀bc":                         badChars, // a letter, but outside the BMP
		strings.Repeat("😀", 9):        tooLong,  // 18 UTF-16 units; length is checked first
		"Tab\tName":                   badChars,
		"�":                           badChars, // what an invalid surrogate decodes to
		strings.Repeat("a", 15) + "!": badChars,
	}
	for name, want := range tests {
		reason, valid := checkName(name)
		switch {
		case want == ok && !valid:
			t.Errorf("checkName(%q) refused with %#x, want accepted", name, reason)
		case want != ok && valid:
			t.Errorf("checkName(%q) accepted, want %#x", name, want)
		case want != ok && reason != want:
			t.Errorf("checkName(%q) = %#x, want %#x", name, reason, want)
		}
	}
}

func TestCheckAppearance(t *testing.T) {
	valid := clientpackets.CharacterCreate{Sex: 0, HairStyle: 4, HairColor: 3, Face: 2}
	tests := []struct {
		name   string
		change func(*clientpackets.CharacterCreate)
		ok     bool
	}{
		{"highest male values", func(*clientpackets.CharacterCreate) {}, true},
		{"lowest values", func(c *clientpackets.CharacterCreate) { c.HairStyle, c.HairColor, c.Face = 0, 0, 0 }, true},
		{"male hair style 5", func(c *clientpackets.CharacterCreate) { c.HairStyle = 5 }, false},
		{"female hair style 6", func(c *clientpackets.CharacterCreate) { c.Sex, c.HairStyle = 1, 6 }, true},
		{"female hair style 7", func(c *clientpackets.CharacterCreate) { c.Sex, c.HairStyle = 1, 7 }, false},
		{"sex 2 is female", func(c *clientpackets.CharacterCreate) { c.Sex, c.HairStyle = 2, 6 }, true},
		{"negative hair style", func(c *clientpackets.CharacterCreate) { c.HairStyle = -1 }, false},
		{"face 3", func(c *clientpackets.CharacterCreate) { c.Face = 3 }, false},
		{"negative face", func(c *clientpackets.CharacterCreate) { c.Face = -1 }, false},
		{"hair color 4", func(c *clientpackets.CharacterCreate) { c.HairColor = 4 }, false},
		{"negative hair color", func(c *clientpackets.CharacterCreate) { c.HairColor = -1 }, false},
		// Mobius casts to byte, so 256 would pass there as 0; L2Go refuses it.
		{"face 256", func(c *clientpackets.CharacterCreate) { c.Face = 256 }, false},
	}
	for _, tt := range tests {
		req := valid
		tt.change(&req)
		problem := checkAppearance(req)
		if (problem == "") != tt.ok {
			t.Errorf("%s: checkAppearance = %q, want ok=%v", tt.name, problem, tt.ok)
		}
	}
}

func TestKeyedMutex(t *testing.T) {
	var k keyedMutex
	counts := map[string]int{}
	var countsMu sync.Mutex // guards the map itself; the keyed lock guards each value
	var wg sync.WaitGroup
	for i := range 200 {
		key := []string{"a", "b", "c"}[i%3]
		wg.Go(func() {
			unlock := k.lock(key)
			defer unlock()
			countsMu.Lock()
			v := counts[key]
			countsMu.Unlock()
			v++
			countsMu.Lock()
			counts[key] = v
			countsMu.Unlock()
		})
	}
	wg.Wait()
	if counts["a"] != 67 || counts["b"] != 67 || counts["c"] != 66 {
		t.Errorf("counts = %v, want a=67 b=67 c=66 (lost updates)", counts)
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if len(k.locks) != 0 {
		t.Errorf("%d locks left after use, want 0", len(k.locks))
	}
}
