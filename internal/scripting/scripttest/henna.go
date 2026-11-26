package scripttest

import "github.com/IPeter/l2go/internal/scripting"

// The symbol windows (scripting.PlayerHenna): a fake player records them in
// HennaWindows.

var _ scripting.PlayerHenna = playerRef{}

func (r playerRef) ShowHennaEquipList() { r.p.HennaWindows = append(r.p.HennaWindows, "equip") }

func (r playerRef) ShowHennaRemoveList() { r.p.HennaWindows = append(r.p.HennaWindows, "remove") }
