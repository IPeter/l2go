// Package henna holds the rules of the dyes (hennas) that symbol makers draw
// on players (Plan 14, M11c): how many slots a class level gives, the stat
// bonus of the worn dyes with Mobius's cap, and the checks of drawing and
// removing a dye (Mobius Player.getHennaEmptySlots, addHenna, removeHenna,
// recalcHennaStats and the RequestHennaEquip and RequestHennaRemove
// packets). It knows no packets, players or storage: internal/game adapts it.
package henna

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// MaxSlots is the number of dye slots (Mobius Player._henna).
const MaxSlots = 3

// StatCap is the most a base stat can gain from the worn dyes. It caps the
// positive side only: the minuses add up without a limit (Mobius
// recalcHennaStats, a quirk kept for parity).
const StatCap = 5

// SymbolMakers are the NPCs that draw and remove dyes (Mobius
// ai/others/SymbolMaker NPCS).
var SymbolMakers = []int{
	31046, // Marsden
	31047, // Kell
	31048, // McDermott
	31049, // Pepper
	31050, // Thora
	31051, // Keach
	31052, // Heid
	31053, // Kidder
	31264, // Olsun
	31308, // Achim
	31953, // Rankar
}

// IsSymbolMaker reports whether the NPC template is a symbol maker.
func IsSymbolMaker(npcID int) bool { return slices.Contains(SymbolMakers, npcID) }

// Slots returns how many dye slots a class of the level has: none for the
// base classes (level 0), 2 after the first class change, 3 after the
// second and the third (Mobius getHennaEmptySlots).
func Slots(classLevel int) int {
	switch {
	case classLevel == 1:
		return 2
	case classLevel > 1:
		return MaxSlots
	}
	return 0
}

// Worn are the dyes in slots 1-3 (index 0-2), nil for an empty slot.
type Worn [MaxSlots]*data.Henna

// Restore returns the dyes of stored dye IDs by slot, as Mobius restoreHenna
// does: 0 and dye IDs the table does not know leave the slot empty.
func Restore(table *data.Hennas, ids [MaxSlots]int32) Worn {
	var w Worn
	for i, id := range ids {
		if id != 0 {
			w[i] = table.Henna(int(id))
		}
	}
	return w
}

// IDs returns the dye IDs by slot, 0 for an empty slot: what is stored.
func (w Worn) IDs() [MaxSlots]int32 {
	var ids [MaxSlots]int32
	for i, h := range w {
		if h != nil {
			ids[i] = int32(h.DyeID) //nolint:gosec // dye IDs are small
		}
	}
	return ids
}

// List returns the worn dyes in slot order, without the empty slots (Mobius
// HennaInfo and HennaRemoveList skip them).
func (w Worn) List() []*data.Henna {
	var out []*data.Henna
	for _, h := range w {
		if h != nil {
			out = append(out, h)
		}
	}
	return out
}

// Count returns how many dyes are worn.
func (w Worn) Count() int { return len(w.List()) }

// EmptySlots returns the slots left for a class of the level: its slots
// minus the worn dyes, never below 0 (Mobius getHennaEmptySlots).
func (w Worn) EmptySlots(classLevel int) int {
	return max(Slots(classLevel)-w.Count(), 0)
}

// Bonus returns the base stat bonus of the worn dyes (Mobius
// recalcHennaStats, entering the stats through FuncHenna). Slot by slot, a
// dye's plus is cut so that the sum stays at most StatCap; minuses are never
// cut.
func (w Worn) Bonus() stats.BaseStats {
	var b stats.BaseStats
	add := func(sum *int, v int) {
		if *sum+v > StatCap {
			*sum += StatCap - *sum
		} else {
			*sum += v
		}
	}
	for _, h := range w {
		if h == nil {
			continue
		}
		// Mobius's order; each stat is on its own.
		add(&b.INT, h.INT)
		add(&b.STR, h.STR)
		add(&b.MEN, h.MEN)
		add(&b.CON, h.CON)
		add(&b.WIT, h.WIT)
		add(&b.DEX, h.DEX)
	}
	return b
}

// Add puts h into the first empty slot and returns that slot (1-3), or 0
// when every slot is taken (Mobius addHenna).
func (w *Worn) Add(h *data.Henna) int {
	for i := range w {
		if w[i] == nil {
			w[i] = h
			return i + 1
		}
	}
	return 0
}

// Remove empties a slot (1-3) and returns the dye that was there, nil for
// an empty or invalid slot (Mobius removeHenna).
func (w *Worn) Remove(slot int) *data.Henna {
	if slot < 1 || slot > MaxSlots {
		return nil
	}
	h := w[slot-1]
	w[slot-1] = nil
	return h
}

// Find returns the first slot (1-3) that holds the dye, 0 if none
// (RequestHennaRemove's loop).
func (w Worn) Find(dyeID int) int {
	for i, h := range w {
		if h != nil && h.DyeID == dyeID {
			return i + 1
		}
	}
	return 0
}

// Disallowed returns the slots (1-3) whose dyes the class may not wear: what
// a class change removes (Mobius setPlayerClass).
func (w Worn) Disallowed(classID int) []int {
	var out []int
	for i, h := range w {
		if h != nil && !h.AllowedClass(classID) {
			out = append(out, i+1)
		}
	}
	return out
}

// EquipResult is the outcome of CheckEquip.
type EquipResult int

const (
	// EquipOK: the dye can be drawn.
	EquipOK EquipResult = iota
	// EquipNoSlot: no empty slot ("No slot exists to draw the symbol.").
	EquipNoSlot
	// EquipUnknown: the table has no such dye (logged, ActionFailed).
	EquipUnknown
	// EquipRefused: the class may not wear it, or too few dyes or adena
	// ("The symbol cannot be drawn.").
	EquipRefused
)

// Player is what the checks need of the player drawing or removing a dye.
type Player struct {
	ClassID    int
	ClassLevel int
	// DyeCount is how many of the dye's item the player carries; Adena its
	// adena.
	DyeCount int64
	Adena    int64
}

// CheckEquip checks drawing h on a player wearing w, in the order of Mobius
// RequestHennaEquip: an empty slot first, then a known dye, then the class,
// the dye count and the fee. h may be nil (an unknown dye ID). forbidden is
// set when the class may not wear a known dye: Mobius punishes that request
// as an exploit (players only).
func CheckEquip(w Worn, h *data.Henna, p Player) (res EquipResult, forbidden bool) {
	switch {
	case w.EmptySlots(p.ClassLevel) == 0:
		return EquipNoSlot, false
	case h == nil:
		return EquipUnknown, false
	case !h.AllowedClass(p.ClassID):
		return EquipRefused, true
	case p.DyeCount < int64(h.WearCount) || p.Adena < int64(h.WearFee):
		return EquipRefused, false
	}
	return EquipOK, false
}

// RemoveResult is the outcome of CheckRemove.
type RemoveResult int

const (
	// RemoveOK: the dye can be removed.
	RemoveOK RemoveResult = iota
	// RemoveNotWorn: the player wears no such dye (logged, ActionFailed).
	RemoveNotWorn
	// RemoveNoAdena: too little adena for the fee ("You do not have enough
	// adena.").
	RemoveNoAdena
)

// CheckRemove checks removing the dye with dyeID from w (Mobius
// RequestHennaRemove) and returns the slot it is in.
func CheckRemove(w Worn, dyeID int, adena int64) (slot int, res RemoveResult) {
	slot = w.Find(dyeID)
	switch {
	case slot == 0:
		return 0, RemoveNotWorn
	case adena < int64(w[slot-1].CancelFee):
		return slot, RemoveNoAdena
	}
	return slot, RemoveOK
}
