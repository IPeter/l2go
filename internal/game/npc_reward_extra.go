package game

// Extra drops: drops that come on top of a monster's own drop list, rolled
// right after it for the player Mobius hands them to (Attackable.
// calculateRewards: EventDropManager.doEventDrop after doItemDrop). The
// seasonal events (long_time_event.go) use it.

// ExtraDropHook rolls extra drops for npc, killed with killer as the one
// Mobius gives them to: the leader of the party that did the most damage,
// else the main damage dealer or the killer. killer may be an NPC or nil.
type ExtraDropHook func(npc *Npc, killer Creature)

// extraDropFeature holds the extra drop hooks, in registration order.
type extraDropFeature struct {
	hooks []ExtraDropHook
}

// OnExtraDrop adds an extra drop hook. Call it while the features are set
// up.
func (s *Server) OnExtraDrop(fn ExtraDropHook) {
	s.extraDrop.hooks = append(s.extraDrop.hooks, fn)
}

// extraDrops runs the extra drop hooks for a kill.
func (s *Server) extraDrops(n *Npc, killer Creature) {
	for _, fn := range s.extraDrop.hooks {
		fn(n, killer)
	}
}
