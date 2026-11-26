package storage

// HennaSlots are the dyes of one class: the dye ID (Mobius symbol_id) in
// slots 1-3 (index 0-2), 0 for an empty slot (a row of Mobius
// character_hennas per worn dye).
type HennaSlots [3]int32

// Empty reports whether no slot holds a dye.
func (h HennaSlots) Empty() bool { return h == HennaSlots{} }

// CloneHennas returns a copy of hennas without the classes that wear no
// dye, or nil if none is left: Character.Hennas as the stores return it.
// The game replaces the map on every change instead of changing it, so a
// snapshot handed to the saver can keep sharing it.
func CloneHennas(hennas map[int32]HennaSlots) map[int32]HennaSlots {
	var out map[int32]HennaSlots
	for k, v := range hennas {
		if v.Empty() {
			continue
		}
		if out == nil {
			out = make(map[int32]HennaSlots, len(hennas))
		}
		out[k] = v
	}
	return out
}
