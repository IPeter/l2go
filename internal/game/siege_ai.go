package game

// The siege guards' AI (Mobius SiegeGuardAI) runs on the attackable AI
// (ai_think.go) with its own aggression: a Defender attacks the players it
// is auto-attackable by (everyone but the defenders of its siege, while the
// siege runs) that it sees within its aggro range (SiegeGuardAI
// autoAttackCondition). It never walks around and, being spawned by its
// siege only, never outlives it. Deviation: Mobius's SiegeGuardAI also
// re-targets the most hated defender-attacker each think and calls nearby
// defenders against a player attacking another guard (thinkAttack's
// faction call); the attackable AI's clan help does the latter for guards
// that share a clan, and its hate list does the former.

// defenderAggressiveTowards is SiegeGuardAI.autoAttackCondition for a living
// player within the aggro range: the guard is auto-attackable by it and
// sees it.
func (s *Server) defenderAggressiveTowards(n *Npc, p *Player) bool {
	a, ok := s.siegeNpcAutoAttackable(n, p)
	return ok && a && s.canSee(n, p)
}
