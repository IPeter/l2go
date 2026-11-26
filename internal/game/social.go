package game

import (
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// socialKickDelay is how long after an illegal action Mobius punishes the
// player (PunishmentManager.handleIllegalPlayerAction schedules it 5
// seconds later).
const socialKickDelay = 5 * time.Second

// socialActionUses are the social actions of the action window's
// RequestActionUse: action ID to SocialAction ID (Mobius RequestActionUse
// cases 12-66).
var socialActionUses = map[int32]int32{
	12: 2, 13: 3, 14: 4, 24: 6, 25: 5, 26: 7, 29: 8, 30: 9, 31: 10, 33: 11, 34: 12, 35: 13, 62: 14, 66: 15,
}

// setupSocial registers RequestSocialAction and the social actions of
// RequestActionUse.
func (s *Server) setupSocial() error {
	errs := []error{s.Register(clientpackets.OpRequestSocialAction, s.handleSocialAction)}
	for actionID, social := range socialActionUses {
		errs = append(errs, s.RegisterActionUse(actionID, func(p *Player, _ clientpackets.RequestActionUse) {
			s.actionUseSocial(p, social)
		}))
	}
	return errors.Join(errs...)
}

// actionUseSocial is a social action of RequestActionUse (Mobius
// tryBroadcastSocial after RequestActionUse's dead check): a dead player
// gets ActionFailed, a fishing one "You cannot do that while fishing."
// (1471); an idle player and the players around see SocialAction.
func (s *Server) actionUseSocial(p *Player, social int32) {
	if p.IsDead() {
		p.SendActionFailed()
		return
	}
	if s.fishingRefusesSocial(p) { // fishing.go
		return
	}
	if !s.socialIdle(p) {
		return
	}
	s.world.Broadcast(p, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{ObjectID: p.ObjectID(), ActionID: social}), true)
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
	if s.fishingRefusesSocial(p) { // fishing.go
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
