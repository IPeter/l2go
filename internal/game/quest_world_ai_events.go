package game

// The NPC events the AI scripts hear of from the movement code (Plan 13,
// A7): on_move_finished when an NPC arrives where it was sent (Mobius
// CreatureAI.onEvtArrived's OnNpcMoveFinished) and on_teleport when it was
// teleported (Npc.onSpawn's OnNpcTeleport). Mobius notifies both
// asynchronously, so they run after the current loop command, in order,
// and only for templates a script listens for (hasListener). The other AI
// events are raised by the AI (ai_think.go: on_npc_hate, on_faction_call;
// ai_see.go: on_creature_see), the day/night change (daynight.go) and the
// walkers.

// npcMoveFinished raises on_move_finished for n.
func (s *Server) npcMoveFinished(n *Npc) { s.postNpcEvent(n, "move_finished") }

// npcTeleported raises on_teleport for n.
func (s *Server) npcTeleported(n *Npc) { s.postNpcEvent(n, "teleport") }

// postNpcEvent posts an NPC event of the scripts listening for n's template.
func (s *Server) postNpcEvent(n *Npc, event string) {
	if s.scripts == nil || !s.scripts.Listens(event, n.Template.ID) {
		return
	}
	ref := n.ScriptRef()
	s.loop.Post(func() {
		if s.scripts == nil || !n.InWorld() {
			return
		}
		ctx := s.background()
		switch event {
		case "move_finished":
			_ = s.scripts.MoveFinished(ctx, ref) // failures are logged
		case "teleport":
			_ = s.scripts.Teleported(ctx, ref)
		}
	})
}
