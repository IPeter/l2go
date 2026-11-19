package game

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// vitalsStartingItems returns the starting items of a class as the
// character screen stores them: the worn ones in their paperdoll slots.
func vitalsStartingItems(e *testEnv, classID int) []storage.Item {
	start := e.data.InitialItems(classID)
	items := make([]storage.Item, len(start))
	for i, it := range start {
		items[i] = storage.Item{ObjectID: int32(1000 + i), ItemID: int32(it.ItemID), Count: it.Count, Slot: storage.PaperdollNone}
		if it.Equipped {
			items[i].Slot = e.slotOf(it.ItemID)
		}
	}
	return items
}

// A new character starts as Mobius CharacterCreate makes it: HP and MP full
// with the CON and MEN bonuses, CP 0. Mobius fills them before the starting
// items are worn and saves the maxima with them: a Human Mystic's starting
// equipment adds MP, so it starts with 59 of 102.
func TestNewCharacterVitals(t *testing.T) {
	e := newEnv(t, nil)
	for _, tc := range []struct {
		name    string
		classID int
		items   []storage.Item
		want    CharacterVitals
	}{
		{"Human Fighter, 80 × 1.58 HP and 30 × 1.28 MP", classHumanFighter, nil,
			CharacterVitals{CurHP: 126, MaxHP: 126, CurMP: 38, MaxMP: 38, MaxCP: 50}},
		{"Human Fighter with its starting items", classHumanFighter, vitalsStartingItems(e, classHumanFighter),
			CharacterVitals{CurHP: 126, MaxHP: 126, CurMP: 38, MaxMP: 38, MaxCP: 50}},
		{"Human Mystic with its starting items", classHumanMystic, vitalsStartingItems(e, classHumanMystic),
			CharacterVitals{CurHP: 98, MaxHP: 98, CurMP: 59, MaxMP: 102, MaxCP: 49}},
	} {
		got, err := e.srv.NewCharacterVitals(tc.classID, 1, tc.items)
		if err != nil || got != tc.want {
			t.Errorf("%s: %+v, %v; want %+v", tc.name, got, err, tc.want)
		}
	}
	if _, err := e.srv.NewCharacterVitals(9999, 1, nil); err == nil {
		t.Error("NewCharacterVitals of an unknown class succeeded")
	}
}
