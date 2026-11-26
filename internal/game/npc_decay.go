package game

import "time"

// Corpse decay (Mobius DecayTaskManager): a scan every second takes out the
// corpses whose time is up. A corpse stays the template's corpseTime, or
// the configured DefaultCorpseTime, so with the 1 s scan it goes 7 to 8 s
// after a death at the 7 s default. Spoiled corpses stay longer once spoil
// exists (M7).

// decayScanPeriod is the period of the decay scan.
const decayScanPeriod = time.Second

// corpseTime is how long the corpse of n stays (Mobius
// NpcTemplate.getCorpseTime).
func (s *Server) corpseTime(n *Npc) time.Duration {
	if secs := n.Template.CorpseTime; secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return time.Duration(s.settings.NPCs.DefaultCorpseTime)
}

// startDecay schedules the decay of a dead NPC, replacing a scheduled one
// (Mobius DecayTaskManager.add).
func (s *Server) startDecay(n *Npc) {
	f := &s.kill
	if _, ok := f.decay[n]; !ok {
		f.decayOrder = append(f.decayOrder, n)
	}
	f.decay[n] = s.Now().Add(s.corpseTime(n) + s.manorCorpseExtension(n)) // manor_sow.go: a sown corpse
}

// CancelNpcDecay stops the decay of n's corpse, for a revive (Mobius
// DecayTaskManager.cancel). Loop-only.
func (s *Server) CancelNpcDecay(n *Npc) { delete(s.kill.decay, n) }

// EndNpcDecay makes n's corpse decay now (Mobius Npc.endDecayTask): it
// leaves the world and its spawn schedules the respawn. Nothing happens to
// a living NPC or one out of the world. Loop-only.
func (s *Server) EndNpcDecay(n *Npc) {
	s.CancelNpcDecay(n)
	if n.Dead && n.InWorld() {
		s.DespawnNpc(n)
	}
}

// NpcDecaysAt returns when n's corpse decays, and false when no decay is
// scheduled. Loop-only.
func (s *Server) NpcDecaysAt(n *Npc) (time.Time, bool) {
	t, ok := s.kill.decay[n]
	return t, ok
}

// decayScan is the decay scan (Mobius DecayTaskManager.run): corpses past
// their time decay (Npc.onDecay through DespawnNpc); corpses revived or
// taken out of the world meanwhile leave the scan.
func (s *Server) decayScan() {
	f := &s.kill
	if len(f.decayOrder) == 0 {
		return
	}
	now := s.Now()
	var due []*Npc
	kept := f.decayOrder[:0]
	for _, n := range f.decayOrder {
		at, ok := f.decay[n]
		switch {
		case !ok:
		case !n.Dead || !n.InWorld():
			delete(f.decay, n)
		case now.After(at):
			delete(f.decay, n)
			due = append(due, n)
		default:
			kept = append(kept, n)
		}
	}
	clear(f.decayOrder[len(kept):])
	f.decayOrder = kept
	for _, n := range due {
		s.DespawnNpc(n)
	}
}
