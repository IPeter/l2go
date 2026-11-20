package game

import "time"

func init() { registerFeature("spawnprotect", setupSpawnProtection) }

// spawnProtectionEnded is the message Mobius Player.onActionRequest sends
// when an action ends the spawn protection outside a peace zone.
const spawnProtectionEnded = "You are no longer protected from aggressive monsters."

// setupSpawnProtection protects a player that enters the world for
// Settings.SpawnProtection (Mobius EnterWorld: setSpawnProtection(true)).
func setupSpawnProtection(s *Server) error {
	if s.settings.SpawnProtection <= 0 {
		return nil
	}
	s.OnPlayerEnter(func(p *Player) { p.SpawnProtectedUntil = s.Now().Add(s.settings.SpawnProtection) })
	return nil
}

// SpawnProtected reports whether p is still safe from aggressive monsters
// (Mobius isSpawnProtected). Loop-only.
func (s *Server) SpawnProtected(p *Player) bool {
	return !p.SpawnProtectedUntil.IsZero() && s.Now().Before(p.SpawnProtectedUntil)
}

// onActionRequest is Mobius Player.onActionRequest, which the packets of a
// deliberate action run (Action, AttackRequest, MoveToLocation; Mobius also
// UseItem and RequestMagicSkillUse): the action ends the spawn protection,
// with a message outside a peace zone. Loop-only.
func (s *Server) onActionRequest(p *Player) {
	if !s.SpawnProtected(p) {
		return
	}
	p.SpawnProtectedUntil = time.Time{}
	if !p.Zones.Inside(ZonePeace) {
		p.SendMessage(spawnProtectionEnded)
	}
}
