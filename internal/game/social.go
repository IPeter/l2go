package game

import (
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// socialKickDelay is how long after an illegal action Mobius punishes the
// player (PunishmentManager.handleIllegalPlayerAction schedules it 5
// seconds later).
const socialKickDelay = 5 * time.Second

// setupSocial registers RequestSocialAction.
func (s *Server) setupSocial() error {
	return s.Register(clientpackets.OpRequestSocialAction, s.handleSocialAction)
}

// handleSocialAction handles RequestSocialAction (0x1B), a greeting, bow or
// dance from the Actions window (Mobius RequestSocialAction): the player
// and the players around see it as SocialAction, when the player is idle.
// The actions players may not ask for (outside 2-13) are illegal actions.
func (s *Server) handleSocialAction(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSocialAction(body)
	if err != nil {
		s.log.Debug("game: bad RequestSocialAction", append(p.logArgs(), "err", err)...)
		return
	}
	if !req.Allowed() {
		s.socialPunishIllegal(p, "requested an internal social action", "actionId", req.ActionID)
		return
	}
	if !s.socialIdle(p) {
		return
	}
	s.world.Broadcast(p, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
		ObjectID: p.ObjectID(),
		ActionID: req.ActionID,
	}), true)
}

// socialIdle reports whether p may play a social action: Mobius wants no
// private store and no pending request, a living player whose skills are
// not disabled, and an idle AI. Here that is a player in the world that
// neither teleports, sits, walks nor walks up to an NPC.
func (s *Server) socialIdle(p *Player) bool {
	return p.InWorld() && !p.Teleporting() && !p.Sitting && p.Move == nil && s.npc.approaches[p] == nil
}

// socialPunishIllegal is Mobius PunishmentManager.handleIllegalPlayerAction
// with the default punishment, KICK: the player is told at once, the log
// gets an audit line, and 5 seconds later a player who is not a GM is
// saved and disconnected with LeaveWorld.
func (s *Server) socialPunishIllegal(p *Player, what string, args ...any) {
	s.log.Warn("game: illegal action: "+what, append(p.logArgs(), args...)...)
	p.SendMessage("You will be kicked for illegal action, GM informed.")
	p.after(socialKickDelay, func() {
		if p.IsGM() {
			return
		}
		if sess := p.Session(); sess != nil {
			s.Logout(sess)
		}
	})
}
