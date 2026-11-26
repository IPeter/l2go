package game

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// Crests (Plan 12 Phase 7 item 5, decision T3): the crest registry (Mobius
// CrestTable), the clan crest and the large clan crest (RequestSetPledgeCrest,
// RequestExSetPledgeCrestLarge, Clan.changeClanCrest and changeLargeCrest)
// and the crest requests (RequestPledgeCrest, RequestExPledgeCrestLarge).
//
// The alliance crest packets (0x87, 0x88) belong to the alliances; they use
// the helpers here: newCrest makes a crest, crestData reads one, dropCrest
// deletes one and applyAllyCrest is Mobius Clan.changeAllyCrest.
//
// Every upload gets a new crest ID (clients cache crests by ID) and the old
// crest is deleted. The ID comes from the crest store off the loop, so an
// upload finishes a moment later on the loop.

func init() { registerFeature("clan crests", setupCrests) }

const (
	// crestLoadTimeout bounds loading the crests at startup.
	crestLoadTimeout = time.Minute
	// crestIDTimeout bounds reserving a crest ID.
	crestIDTimeout = 10 * time.Second
	// crestMinClanLevel is the clan level a clan crest needs.
	crestMinClanLevel = 3
)

// System messages of the crests (Mobius SystemMessageId).
const (
	crestMsgImageSize       serverpackets.SystemMessageID = 209  // "The size of the image file is inappropriate. Please adjust to 16*12."
	crestMsgNeedsClanLevel3 serverpackets.SystemMessageID = 272  // "A clan crest can only be registered when the clan's skill level is 3 or above."
	crestMsgDissolving      serverpackets.SystemMessageID = 552  // "During the grace period for dissolving a clan, ..."
	crestMsgLargeRegistered serverpackets.SystemMessageID = 1663 // "The clan crest was successfully registered. Remember, ..."
	crestMsgDeleted         serverpackets.SystemMessageID = 1861 // "The clan's crest has been deleted."
)

// The plain-text messages of the crest uploads (Mobius sendMessage).
const (
	crestMsgRegisteredText = "The crest was successfully registered."
	crestMsgBadSizeText    = "The size of the uploaded crest or insignia does not meet the standard requirements."
)

// crestFeature is the crest registry on the Server (Server.crests).
// Loop-only, except store, which New sets, and pending.
type crestFeature struct {
	// store hands out crest IDs and loads the crests; nil keeps them in
	// memory with IDs counted from lastID.
	store  storage.CrestStore
	crests map[int32]storage.Crest
	lastID int32 // the highest crest ID known, for a nil store
	// pending counts the crest ID reservations in flight; each posts its
	// result to the loop before it is done. Tests wait on it.
	pending sync.WaitGroup
}

// setupCrests loads the crests (Mobius CrestTable.load: crests no clan uses
// are deleted, crest IDs of clans whose crest is gone are cleared) and
// registers the crest packets. It runs after the clan registry is loaded.
func setupCrests(s *Server) error {
	f := &s.crests
	f.crests = make(map[int32]storage.Crest)
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), crestLoadTimeout)
		defer cancel()
		if err := s.LoadCrests(ctx); err != nil {
			return err
		}
	}
	return errors.Join(
		s.Register(clientpackets.OpRequestPledgeCrest, s.handleRequestPledgeCrest),
		s.Register(clientpackets.OpRequestSetPledgeCrest, s.handleSetPledgeCrest),
		s.RegisterEx(clientpackets.ExOpRequestExPledgeCrestLarge, s.handleRequestPledgeCrestLarge),
		s.RegisterEx(clientpackets.ExOpRequestExSetPledgeCrestLarge, s.handleSetPledgeCrestLarge),
	)
}

// LoadCrests replaces the crest registry with the crests of the crest store
// and cleans up as Mobius CrestTable.load does. New calls it after the
// clans are loaded; call it before Run. It does nothing without a crest
// store.
func (s *Server) LoadCrests(ctx context.Context) error {
	f := &s.crests
	if f.store == nil {
		return nil
	}
	list, err := f.store.LoadCrests(ctx)
	if err != nil {
		return fmt.Errorf("game: load the crests: %w", err)
	}
	used := make(map[int32]bool)
	for _, c := range s.clan.clans {
		for _, id := range []int32{c.CrestID, c.CrestLargeID, c.AllyCrestID} {
			if id != 0 {
				used[id] = true
			}
		}
	}
	clear(f.crests)
	var b storage.Batch
	for _, cr := range list {
		f.lastID = max(f.lastID, cr.ID)
		if !used[cr.ID] {
			b.DeletedCrests = append(b.DeletedCrests, cr.ID)
			continue
		}
		f.crests[cr.ID] = cr
	}
	for _, c := range sortedClans(s.clan.clans) {
		changed := false
		if c.CrestID != 0 && !f.has(c.CrestID) {
			s.log.Info("game: removing a missing clan crest", "clan", c.Name, "crestId", c.CrestID)
			c.CrestID, changed = 0, true
		}
		if c.CrestLargeID != 0 && !f.has(c.CrestLargeID) {
			s.log.Info("game: removing a missing large clan crest", "clan", c.Name, "crestLargeId", c.CrestLargeID)
			c.CrestLargeID, changed = 0, true
		}
		if c.AllyCrestID != 0 && !f.has(c.AllyCrestID) {
			s.log.Info("game: removing a missing alliance crest", "clan", c.Name, "allyCrestId", c.AllyCrestID)
			c.AllyCrestID, changed = 0, true
		}
		if changed {
			b.Clans = append(b.Clans, c.document())
		}
	}
	if !b.Empty() {
		s.SaveJoint(nil, b)
	}
	s.log.Info("crests loaded", "crests", len(f.crests), "deleted", len(b.DeletedCrests))
	return nil
}

func (f *crestFeature) has(id int32) bool {
	_, ok := f.crests[id]
	return ok
}

// crestData returns the image of the crest id, nil when there is none
// (Mobius CrestTable.getCrest(id).getData()). Loop-only.
func (s *Server) crestData(id int32) []byte {
	if cr, ok := s.crests.crests[id]; ok {
		return cr.Data
	}
	return nil
}

// newCrest makes a crest of typ (storage.CrestPledge, CrestPledgeLarge or
// CrestAlly) from data with a new crest ID (Mobius CrestTable.createCrest).
// The ID is reserved off the loop; then runs on the loop with it and a
// batch that stores the crest, which is in the registry already. then
// changes the crest's owner, deletes the crest it replaces (dropCrest) and
// writes the batch, with the owner's document, with SaveJoint. When no ID
// can be had, the failure is logged and then does not run. Loop-only.
func (s *Server) newCrest(typ int32, data []byte, then func(id int32, b *storage.Batch)) {
	f := &s.crests
	data = bytes.Clone(data)
	apply := func(id int32) {
		cr := storage.Crest{ID: id, Type: typ, Data: data}
		f.crests[id] = cr
		f.lastID = max(f.lastID, id)
		then(id, &storage.Batch{Crests: []storage.Crest{cr}})
	}
	if f.store == nil {
		apply(f.lastID + 1)
		return
	}
	f.pending.Add(1)
	go func() {
		defer f.pending.Done()
		ctx, cancel := context.WithTimeout(s.background(), crestIDTimeout)
		id, err := f.store.NextCrestID(ctx)
		cancel()
		if err != nil {
			s.log.Warn("game: cannot reserve a crest ID", "err", err)
			return
		}
		s.loop.Post(func() { apply(id) })
	}()
}

// dropCrest deletes the crest id from the registry and adds its delete to b
// (Mobius CrestTable.removeCrest). 0 is no crest. Loop-only.
func (s *Server) dropCrest(id int32, b *storage.Batch) {
	if id == 0 {
		return
	}
	delete(s.crests.crests, id)
	b.DeletedCrests = append(b.DeletedCrests, id)
}

// broadcastClanUserInfo sends UserInfo and CharInfo of every member online
// (Mobius: member.broadcastUserInfo() for getOnlineMembers(0)).
func broadcastClanUserInfo(c *Clan) {
	for _, q := range c.OnlineMembers() {
		q.BroadcastUserInfo()
	}
}

// changeClanCrest is Mobius Clan.changeClanCrest: the old crest is deleted,
// c gets crestID (0 for none), the clan is written with b (which may carry
// the new crest) and the members online show it. Loop-only.
func (s *Server) changeClanCrest(c *Clan, crestID int32, b storage.Batch) {
	s.dropCrest(c.CrestID, &b)
	c.CrestID = crestID
	b.Clans = append(b.Clans, c.document())
	s.SaveJoint(nil, b)
	broadcastClanUserInfo(c)
}

// changeLargeClanCrest is Mobius Clan.changeLargeCrest, like
// changeClanCrest. Loop-only.
func (s *Server) changeLargeClanCrest(c *Clan, crestID int32, b storage.Batch) {
	s.dropCrest(c.CrestLargeID, &b)
	c.CrestLargeID = crestID
	b.Clans = append(b.Clans, c.document())
	s.SaveJoint(nil, b)
	broadcastClanUserInfo(c)
}

// applyAllyCrest is Mobius Clan.changeAllyCrest for the alliances. With
// onlyThisClan only c's alliance crest becomes crestID (a clan joining or
// leaving an alliance) and c is written. Otherwise the alliance's old crest
// is deleted and every clan of c's alliance gets crestID (an AllyCrests
// write, Mobius's UPDATE ... WHERE ally_id). The members online of the
// changed clans show it. b may carry the new crest (newCrest); it is written
// jointly. Loop-only.
func (s *Server) applyAllyCrest(c *Clan, crestID int32, onlyThisClan bool, b storage.Batch) {
	if onlyThisClan {
		c.AllyCrestID = crestID
		b.Clans = append(b.Clans, c.document())
		s.SaveJoint(nil, b)
		broadcastClanUserInfo(c)
		return
	}
	s.dropCrest(c.AllyCrestID, &b)
	allyID := c.AllyID
	var changed []*Clan
	if allyID != 0 {
		b.AllyCrests = append(b.AllyCrests, storage.AllyCrest{AllyID: allyID, CrestID: crestID})
		for _, o := range sortedClans(s.clan.clans) {
			if o.AllyID == allyID {
				o.AllyCrestID = crestID
				changed = append(changed, o)
			}
		}
	}
	if !b.Empty() {
		s.SaveJoint(nil, b)
	}
	for _, o := range changed {
		broadcastClanUserInfo(o)
	}
}

// ----- packets -----

// handleRequestPledgeCrest handles RequestPledgeCrest (0x68): the image of
// a crest, empty for an unknown one (Mobius PledgeCrest). The session layer
// lets it through on the character screen too, where, as in Mobius, it is
// answered only once a player is in the game.
func (s *Server) handleRequestPledgeCrest(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPledgeCrest(body)
	if err != nil {
		s.log.Debug("game: bad RequestPledgeCrest", append(p.logArgs(), "err", err)...)
		return
	}
	p.Send(serverpackets.NewPledgeCrestPacket(req.CrestID, s.crestData(req.CrestID)))
}

// handleRequestPledgeCrestLarge handles RequestExPledgeCrestLarge (D0:10):
// the image of a large crest (Mobius ExPledgeEmblem).
func (s *Server) handleRequestPledgeCrestLarge(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestExPledgeCrestLarge(body)
	if err != nil {
		s.log.Debug("game: bad RequestExPledgeCrestLarge", append(p.logArgs(), "err", err)...)
		return
	}
	p.Send(serverpackets.NewExPledgeEmblemPacket(req.CrestID, s.crestData(req.CrestID)))
}

// handleSetPledgeCrest handles RequestSetPledgeCrest (0x53), Mobius's
// checks in order: the image size, a clan, no dissolution, the CHANGE_CREST
// privilege; an empty image deletes the crest, a new one needs clan level 3.
func (s *Server) handleSetPledgeCrest(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSetPledgeCrest(body)
	if err != nil {
		s.log.Debug("game: bad RequestSetPledgeCrest", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Length < 0 {
		p.SendMessage(crestMsgBadSizeText)
		return
	}
	if req.Length > clientpackets.MaxPledgeCrestSize {
		p.SendSystemMessage(crestMsgImageSize)
		return
	}
	c := p.clan.clan
	if c == nil || !s.checkCrestChange(p, c) {
		return
	}
	if req.Length == 0 {
		if c.CrestID != 0 {
			s.changeClanCrest(c, 0, storage.Batch{})
			p.SendSystemMessage(crestMsgDeleted)
			s.log.Info("clan crest deleted", append(p.logArgs(), "clan", c.Name)...)
		}
		return
	}
	if c.Level < crestMinClanLevel {
		p.SendSystemMessage(crestMsgNeedsClanLevel3)
		return
	}
	s.newCrest(storage.CrestPledge, req.Data, func(id int32, b *storage.Batch) {
		if s.clan.clans[c.ID] != c {
			s.dropCrest(id, b) // the clan went meanwhile; the crest was never stored
			return
		}
		s.changeClanCrest(c, id, *b)
		p.SendMessage(crestMsgRegisteredText)
		s.log.Info("clan crest registered", append(p.logArgs(), "clan", c.Name, "crestId", id)...)
	})
}

// handleSetPledgeCrestLarge handles RequestExSetPledgeCrestLarge (D0:11),
// as handleSetPledgeCrest: an image above 2,176 bytes is ignored, as Mobius
// does not read it.
func (s *Server) handleSetPledgeCrestLarge(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestExSetPledgeCrestLarge(body)
	if err != nil {
		s.log.Debug("game: bad RequestExSetPledgeCrestLarge", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Length < 0 || req.Length > clientpackets.MaxPledgeCrestLargeSize {
		return
	}
	c := p.clan.clan
	if c == nil || !s.checkCrestChange(p, c) {
		return
	}
	if req.Length == 0 {
		if c.CrestLargeID != 0 {
			s.changeLargeClanCrest(c, 0, storage.Batch{})
			p.SendSystemMessage(crestMsgDeleted)
			s.log.Info("large clan crest deleted", append(p.logArgs(), "clan", c.Name)...)
		}
		return
	}
	if c.Level < crestMinClanLevel {
		p.SendSystemMessage(crestMsgNeedsClanLevel3)
		return
	}
	s.newCrest(storage.CrestPledgeLarge, req.Data, func(id int32, b *storage.Batch) {
		if s.clan.clans[c.ID] != c {
			s.dropCrest(id, b)
			return
		}
		s.changeLargeClanCrest(c, id, *b)
		p.SendSystemMessage(crestMsgLargeRegistered)
		s.log.Info("large clan crest registered", append(p.logArgs(), "clan", c.Name, "crestId", id)...)
	})
}

// checkCrestChange is the shared part of the crest uploads: no change while
// the clan is dissolving, and only with the CHANGE_CREST privilege.
func (s *Server) checkCrestChange(p *Player, c *Clan) bool {
	switch {
	case c.DissolvingExpiry > s.Now().UnixMilli():
		p.SendSystemMessage(crestMsgDissolving)
	case !p.HasClanAccess(social.AccessChangeCrest):
		p.SendSystemMessage(clanMsgNotAuthorized)
	default:
		return true
	}
	return false
}
