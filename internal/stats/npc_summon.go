package stats

// SetTemplate replaces the NPC template the stat set computes from, keeping
// its modifiers and motion: a pet's base stats and level change with its
// level (Mobius PetStat reads PetLevelData). nil is ignored.
func (n *Npc) SetTemplate(t *NpcTemplate) {
	if t != nil && t != n.tmpl {
		n.tmpl = t
		n.changed()
	}
}
