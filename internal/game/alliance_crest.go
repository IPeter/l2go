package game

import (
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// The alliance crest (Mobius RequestSetAllyCrest, RequestAllyCrest), on the
// crest registry of clan_crest.go.

// handleAllyCrest handles RequestAllyCrest (0x88): the image of an alliance
// crest, empty for an unknown one (Mobius AllyCrest).
func (s *Server) handleAllyCrest(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAllyCrest(body)
	if err != nil {
		s.log.Debug("game: bad RequestAllyCrest", append(p.logArgs(), "err", err)...)
		return
	}
	p.Send(serverpackets.NewAllyCrestPacket(req.CrestID, s.crestData(req.CrestID)))
}

// handleSetAllyCrest handles RequestSetAllyCrest (0x87), Mobius's checks in
// order: the image size (at most 192 bytes), an alliance, its leader clan's
// leader. An empty image deletes the crest; a new one replaces it for every
// clan of the alliance (Clan.changeAllyCrest(id, false)).
func (s *Server) handleSetAllyCrest(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSetAllyCrest(body)
	if err != nil {
		s.log.Debug("game: bad RequestSetAllyCrest", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Length < 0 {
		p.SendMessage("File transfer error.")
		return
	}
	if req.Length > clientpackets.MaxAllyCrestSize {
		p.SendSystemMessage(allyMsgCrestSize)
		return
	}
	allyID := p.AllyID()
	if allyID == 0 {
		p.SendSystemMessage(allyMsgOnlyAllyLeaders)
		return
	}
	leader := s.Clan(allyID)
	if leader == nil || p.clan.clan != leader || !p.IsClanLeader() {
		p.SendSystemMessage(allyMsgOnlyAllyLeaders)
		return
	}
	if req.Length == 0 {
		if leader.AllyCrestID != 0 {
			s.applyAllyCrest(leader, 0, false, storage.Batch{})
			s.log.Info("alliance crest deleted", append(p.logArgs(), "alliance", leader.AllyName)...)
		}
		return
	}
	s.newCrest(storage.CrestAlly, req.Data, func(id int32, b *storage.Batch) {
		if s.clan.clans[leader.ID] != leader || leader.AllyID != allyID {
			s.dropCrest(id, b) // the alliance went meanwhile; the crest was never stored
			return
		}
		s.applyAllyCrest(leader, id, false, *b)
		p.SendMessage(crestMsgRegisteredText)
		s.log.Info("alliance crest registered", append(p.logArgs(), "alliance", leader.AllyName, "crestId", id)...)
	})
}
