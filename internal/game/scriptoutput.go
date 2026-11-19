package game

import (
	"sync/atomic"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// ScriptOutput carries what scripts do to the world (scripting.Output). The
// scripting engine needs it when it is made, before the server exists, so
// it is made first and connected later:
//
//	out := game.NewScriptOutput()
//	engine, err := scripting.New(ctx, scripting.Options{..., Output: out})
//	srv, err := game.New(game.Options{..., Scripts: engine, ScriptOutput: out})
//
// Until New connects it, what scripts do is dropped. The engine calls it on
// the loop, while a handler runs.
type ScriptOutput struct {
	srv atomic.Pointer[Server]
}

var _ scripting.Output = (*ScriptOutput)(nil)

// NewScriptOutput returns an output that New connects to its server.
func NewScriptOutput() *ScriptOutput { return &ScriptOutput{} }

func (o *ScriptOutput) bind(s *Server) { o.srv.Store(s) }

// npcDisplayID is implemented by NPCs whose client ID differs from their
// template ID (Mobius NpcTemplate.getDisplayId).
type npcDisplayID interface {
	DisplayID() int
}

// NpcSay makes npc say text to the players that see it, in NPC chat (Mobius
// Npc.broadcastSay with NPC_GENERAL). An NPC that is not in the world says
// nothing.
func (o *ScriptOutput) NpcSay(npc scripting.NpcRef, text string) {
	s := o.srv.Load()
	if s == nil || npc == nil {
		return
	}
	obj, ok := s.world.Find(world.ObjectID(npc.ObjectID()))
	if !ok {
		return
	}
	id := npc.ID()
	if d, ok := npc.(npcDisplayID); ok {
		id = d.DisplayID()
	}
	s.world.Broadcast(obj, serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
		ObjectID: npc.ObjectID(),
		Type:     serverpackets.ChatGeneral,
		NpcID:    int32(id), //nolint:gosec // NPC IDs fit in int32
		Text:     text,
	}), false)
}
