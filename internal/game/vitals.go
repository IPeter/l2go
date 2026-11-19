package game

import (
	"fmt"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// CharacterVitals are the HP, MP and CP of a character: the current values
// and their maximum.
type CharacterVitals struct {
	CurHP, MaxHP float64
	CurMP, MaxMP float64
	CurCP, MaxCP float64
}

// NewCharacterVitals returns the HP, MP and CP a new character of class
// classID at level starts with, as Mobius CharacterCreate sets them: HP and
// MP full and CP 0, from the class's stats with the CON and MEN bonuses (a
// Human Fighter has 126 HP and 38 MP, not the template's 80 and 30). Mobius
// fills HP and MP before the starting items are worn, and saves the maxima
// with them worn: items are the new character's items, the worn ones with
// their paperdoll slot, so a Human Mystic starts with 59 of 102 MP. Safe
// on any goroutine.
func (s *Server) NewCharacterVitals(classID, level int, items []storage.Item) (CharacterVitals, error) {
	tmpl := s.classes[classID]
	if tmpl == nil {
		return CharacterVitals{}, fmt.Errorf("game: class %d has no stat template", classID)
	}
	st, err := stats.NewPlayer(s.rules, tmpl, level)
	if err != nil {
		return CharacterVitals{}, fmt.Errorf("game: vitals of class %d: %w", classID, err)
	}
	v := CharacterVitals{CurHP: float64(st.MaxHP()), CurMP: float64(st.MaxMP())}
	for _, it := range items {
		t, ok := s.data.ItemTemplate(int(it.ItemID))
		if !it.Slot.Equipped() || !ok {
			continue
		}
		// An item without usable stats changes nothing, as when it is worn.
		_ = st.Equip(stats.Slot(it.Slot), stats.ItemOwner(it.ObjectID), statsItem(t, int(it.Enchant))) //nolint:gosec // Equipped: a slot from 0 up
	}
	v.MaxHP, v.MaxMP, v.MaxCP = float64(st.MaxHP()), float64(st.MaxMP()), float64(st.MaxCP())
	return v, nil
}

// vitalsStatusUpdate is the StatusUpdate a player gets about its own HP, MP
// and CP (Mobius Player.broadcastStatusUpdate): the maximum and the current
// value, cut to an int, of each.
func vitalsStatusUpdate(p *Player) []byte {
	v := p.stats.View()
	return serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
		ObjectID: p.ObjectID(),
		Values: []serverpackets.StatusValue{
			{Attr: serverpackets.StatusMaxHP, Value: v.MaxHP},
			{Attr: serverpackets.StatusCurHP, Value: int32(p.CurHP)},
			{Attr: serverpackets.StatusMaxMP, Value: v.MaxMP},
			{Attr: serverpackets.StatusCurMP, Value: int32(p.CurMP)},
			{Attr: serverpackets.StatusMaxCP, Value: v.MaxCP},
			{Attr: serverpackets.StatusCurCP, Value: int32(p.CurCP)},
		},
	})
}
