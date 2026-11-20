package game

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The friend and block lists (Plan 12 Phase 4, decision T4): Mobius
// BlockList, the friend list of Player and the packets RequestFriendInvite,
// RequestAnswerFriendInvite, RequestFriendList, RequestFriendDel,
// RequestBlock and RequestSendFriendMsg. The lists are loaded with the
// character (SelectCharacter) and written through the saver as relation
// changes (SaveJoint), so they also reach offline characters.
func init() { registerFeature("friends", setupFriends) }

// System messages of the friend and block lists (Mobius SystemMessageId).
const (
	sysMsgAddedToFriends       serverpackets.SystemMessageID = 132  // $s1 has been added to your friends list.
	sysMsgBusyTryLater         serverpackets.SystemMessageID = 153  // $s1 is busy. Please try again later.
	sysMsgCannotAddYourself    serverpackets.SystemMessageID = 165  // You cannot add yourself to your own friend list.
	sysMsgRequestedFriends     serverpackets.SystemMessageID = 168  // $s1 has requested to become friends.
	sysMsgFriendNotFound       serverpackets.SystemMessageID = 170  // The user who requested to become friends is not found in the game.
	sysMsgNotOnFriendList      serverpackets.SystemMessageID = 171  // $s1 is not on your friend list.
	sysMsgRefusalMode          serverpackets.SystemMessageID = 177  // Message refusal mode.
	sysMsgAcceptanceMode       serverpackets.SystemMessageID = 178  // Message acceptance mode.
	sysMsgJoinedAsFriend       serverpackets.SystemMessageID = 479  // $s1 has joined as a friend.
	sysMsgDeletedFromFriends   serverpackets.SystemMessageID = 481  // $s1 has been deleted from your friends list.
	sysMsgAlreadyFriend        serverpackets.SystemMessageID = 484  // This player is already registered in your friends list.
	sysMsgFriendsListHeader    serverpackets.SystemMessageID = 487  // ======<Friends List>======
	sysMsgFriendOnline         serverpackets.SystemMessageID = 488  // $s1 (Currently: Online)
	sysMsgFriendOffline        serverpackets.SystemMessageID = 489  // $s1 (Currently: Offline)
	sysMsgListFooter           serverpackets.SystemMessageID = 490  // ========================
	sysMsgFriendInviteSuccess  serverpackets.SystemMessageID = 525  // You have succeeded in inviting a friend to your friends list.
	sysMsgFriendInviteFailed   serverpackets.SystemMessageID = 526  // You have failed to add a friend to your friends list.
	sysMsgIgnoreListHeader     serverpackets.SystemMessageID = 613  // ======<Ignore List>======
	sysMsgIgnoreRegisterFailed serverpackets.SystemMessageID = 615  // You have failed to register the user to your Ignore List.
	sysMsgAddedToIgnore        serverpackets.SystemMessageID = 617  // $s1 has been added to your Ignore List.
	sysMsgRemovedFromIgnore    serverpackets.SystemMessageID = 618  // $s1 has been removed from your Ignore List.
	sysMsgCannotBlockGM        serverpackets.SystemMessageID = 827  // You may not impose a block on a GM.
	sysMsgYouHaveBlocked       serverpackets.SystemMessageID = 2057 // You have blocked $s1.
)

// SysMsgPlacedOnIgnoreList is "$s1 has placed you on his/her Ignore List."
// Party invites and trade requests to a player who blocks the sender get it
// too.
const SysMsgPlacedOnIgnoreList serverpackets.SystemMessageID = 619

// friendMessageMax is the longest friend message in UTF-16 units (Mobius
// RequestSendFriendMsg).
const friendMessageMax = 300

// relationLoadTimeout bounds the lookup of a character that is not in the
// game, by name, for the block list.
const relationLoadTimeout = 10 * time.Second

// friendsFeature is the server state of the friend lists.
type friendsFeature struct {
	// store reads the friend and block lists at character selection; nil
	// starts every list empty (Options.Relations).
	store storage.RelationStore
}

// playerSocial are a player's friend and block lists and its message
// refusal mode (Mobius _friendList, BlockList and _messageRefusal).
type playerSocial struct {
	friends   []socialEntry // in stored order, then in the order added
	blocks    []socialEntry
	refuseAll bool // message refusal: blocks everyone; not stored, as in Mobius
}

// socialEntry is a character on a list, with its name for the packets and
// messages (Mobius CharInfoTable.getNameById).
type socialEntry struct {
	id   int32
	name string
}

func setupFriends(s *Server) error {
	for _, r := range []struct {
		op byte
		h  Handler
	}{
		{clientpackets.OpRequestFriendInvite, s.handleFriendInvite},
		{clientpackets.OpRequestAnswerFriendInvite, s.handleAnswerFriendInvite},
		{clientpackets.OpRequestFriendList, s.handleFriendList},
		{clientpackets.OpRequestFriendDel, s.handleFriendDel},
		{clientpackets.OpRequestBlock, s.handleBlock},
		{clientpackets.OpRequestSendFriendMsg, s.handleFriendMessage},
	} {
		if err := s.Register(r.op, r.h); err != nil {
			return err
		}
	}
	// Mobius onPlayerEnter (Player.java 10722) and deleteMe (11697): the
	// friends online see the player come and go.
	s.OnPlayerEnter(func(p *Player) { s.notifyFriends(p, true) })
	s.OnPlayerLeave(func(p *Player) { s.notifyFriends(p, false) })
	return nil
}

// IsFriend reports whether id is on p's friend list. Loop-only.
func (p *Player) IsFriend(id int32) bool { return indexEntry(p.social.friends, id) >= 0 }

// HasBlocked reports whether id is on p's block list (Mobius
// BlockList.isInBlockList). Loop-only.
func (p *Player) HasBlocked(id int32) bool { return indexEntry(p.social.blocks, id) >= 0 }

// MessageRefusal reports whether p refuses all messages (Mobius
// getMessageRefusal), set with the block list's "refuse all". Loop-only.
func (p *Player) MessageRefusal() bool { return p.social.refuseAll }

// Blocks reports whether p refuses whispers, chat and requests from other
// (Mobius BlockList.isBlocked(p, other)): p is in message refusal mode or
// has other on its block list. Whispers, party and clan chat, party
// invites, trade requests and friend invites check it. Loop-only.
func (p *Player) Blocks(other *Player) bool { return p.BlocksID(other.ObjectID()) }

// BlocksID is Blocks by object ID. Loop-only.
func (p *Player) BlocksID(id int32) bool { return p.social.refuseAll || p.HasBlocked(id) }

// FriendIDs returns the object IDs on p's friend list, in list order.
// Loop-only.
func (p *Player) FriendIDs() []int32 {
	ids := make([]int32, len(p.social.friends))
	for i, e := range p.social.friends {
		ids[i] = e.id
	}
	return ids
}

func indexEntry(list []socialEntry, id int32) int {
	return slices.IndexFunc(list, func(e socialEntry) bool { return e.id == id })
}

// indexName finds a name on a list, ignoring case.
func indexName(list []socialEntry, name string) int {
	key := storage.NameKey(name)
	return slices.IndexFunc(list, func(e socialEntry) bool { return storage.NameKey(e.name) == key })
}

// loadRelations reads p's friend and block lists (Mobius
// restoreFriendList and BlockList.loadList) with the names of the
// characters on them. It runs off the loop, before p is registered. A
// failed read leaves the lists empty, as in Mobius; characters that no
// longer exist are left out.
func (s *Server) loadRelations(ctx context.Context, p *Player) {
	store := s.friends.store
	if store == nil {
		return
	}
	rels, err := store.Relations(ctx, p.ObjectID())
	if err != nil {
		s.log.Warn("game: cannot load the friend and block lists", append(p.logArgs(), "err", err)...)
		return
	}
	for _, r := range rels {
		if r.Target == p.ObjectID() {
			continue
		}
		c, err := s.chars.Get(ctx, r.Target)
		if err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				s.log.Warn("game: cannot load a character of the friend or block list", append(p.logArgs(), "target", r.Target, "err", err)...)
			}
			continue
		}
		e := socialEntry{id: r.Target, name: c.Name}
		switch r.Type {
		case storage.RelationFriend:
			p.social.friends = append(p.social.friends, e)
		case storage.RelationBlock:
			p.social.blocks = append(p.social.blocks, e)
		}
	}
}

// onlinePlayer returns the player in the game with the object ID.
func (s *Server) onlinePlayer(id int32) (*Player, bool) {
	q, ok := s.Player(world.ObjectID(id))
	if !ok || q.state != stateInGame {
		return nil, false
	}
	return q, true
}

// friendListPacket builds FriendList (Mobius FriendList): p's friends with
// their online state.
func (s *Server) friendListPacket(p *Player) []byte {
	list := make([]serverpackets.Friend, len(p.social.friends))
	for i, e := range p.social.friends {
		_, online := s.onlinePlayer(e.id)
		list[i] = serverpackets.Friend{ObjectID: e.id, Name: e.name, Online: online}
	}
	return serverpackets.NewFriendListPacket(list)
}

// sendFriendsOnEnter sends p its FriendList and tells its friends online
// that p logged in (Mobius EnterWorld 412-422).
func (s *Server) sendFriendsOnEnter(p *Player) {
	p.Send(s.friendListPacket(p))
	if len(p.social.friends) == 0 {
		return
	}
	msg := serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{
		ID:     serverpackets.SysMsgFriendLoggedIn,
		Params: []serverpackets.SystemMessageParam{serverpackets.TextParam(p.Name())},
	})
	for _, e := range p.social.friends {
		if q, ok := s.onlinePlayer(e.id); ok {
			q.Send(msg)
		}
	}
}

// notifyFriends sends FriendStatusPacket about p to its friends online
// (Mobius notifyFriends): online when it enters, offline when it leaves.
func (s *Server) notifyFriends(p *Player, online bool) {
	if len(p.social.friends) == 0 {
		return
	}
	pkt := serverpackets.NewFriendStatusPacket(online, p.Name(), p.ObjectID())
	for _, e := range p.social.friends {
		if q, ok := s.onlinePlayer(e.id); ok && q != p {
			q.Send(pkt)
		}
	}
}

// saveRelations writes relation changes through the saver (decision T4):
// after the pending writes of both characters, in one transaction.
func (s *Server) saveRelations(added, removed []storage.Relation) {
	s.SaveJoint(nil, storage.Batch{Relations: storage.RelationChanges{Added: added, Removed: removed}})
}

func textParam(s string) serverpackets.SystemMessageParam { return serverpackets.TextParam(s) }

// handleFriendInvite handles RequestFriendInvite (0x5E) with Mobius's
// checks in order (RequestFriendInvite.java 49-114).
func (s *Server) handleFriendInvite(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestFriendInvite(body)
	if err != nil {
		s.log.Debug("game: bad RequestFriendInvite", append(p.logArgs(), "err", err)...)
		return
	}
	friend, ok := s.PlayerByName(req.Name)
	switch {
	case !ok || friend.state != stateInGame || friend.session == nil || friend.Invisible():
		p.SendSystemMessage(sysMsgFriendNotFound)
	case friend == p:
		p.SendSystemMessage(sysMsgCannotAddYourself)
	case friend.Blocks(p):
		p.SendMessage("You are in " + req.Name + "'s block list.")
	case p.Blocks(friend):
		p.SendSystemMessage(sysMsgYouHaveBlocked, textParam(friend.Name()))
	case p.IsFriend(friend.ObjectID()):
		p.SendSystemMessage(sysMsgAlreadyFriend)
	case s.processingRequest(friend): // the "active requester" slot (decision T7)
		p.SendSystemMessage(sysMsgBusyTryLater, textParam(req.Name))
	default:
		s.onTransactionRequest(p, friend)
		friend.Send(serverpackets.NewFriendAddRequestPacket(p.Name()))
		p.SendMessage("You've requested " + req.Name + " to be on your Friends List.")
		friend.SendSystemMessage(sysMsgRequestedFriends, textParam(p.Name()))
	}
}

// handleAnswerFriendInvite handles RequestAnswerFriendInvite (0x5F)
// (Mobius RequestAnswerFriendInvite): an acceptance writes the friendship
// both ways in one transaction and adds each to the other's window.
func (s *Server) handleAnswerFriendInvite(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAnswerFriendInvite(body)
	if err != nil {
		s.log.Debug("game: bad RequestAnswerFriendInvite", append(p.logArgs(), "err", err)...)
		return
	}
	requestor := s.activeRequester(p)
	if requestor == nil {
		return
	}
	if requestor.state != stateInGame {
		s.setActiveRequester(p, nil) // the requestor left
		return
	}
	if p.IsFriend(requestor.ObjectID()) || requestor.IsFriend(p.ObjectID()) {
		// Mobius leaves the request open here.
		requestor.SendSystemMessage(sysMsgAlreadyFriend)
		return
	}
	if req.Accepted() {
		s.saveRelations(storage.FriendRelations(requestor.ObjectID(), p.ObjectID()), nil)
		requestor.SendSystemMessage(sysMsgFriendInviteSuccess)
		requestor.SendSystemMessage(sysMsgAddedToFriends, textParam(p.Name()))
		requestor.social.friends = append(requestor.social.friends, socialEntry{id: p.ObjectID(), name: p.Name()})
		p.SendSystemMessage(sysMsgJoinedAsFriend, textParam(requestor.Name()))
		p.social.friends = append(p.social.friends, socialEntry{id: requestor.ObjectID(), name: requestor.Name()})
		p.Send(l2FriendOf(requestor, serverpackets.L2FriendAdd))
		requestor.Send(l2FriendOf(p, serverpackets.L2FriendAdd))
	} else {
		requestor.SendSystemMessage(sysMsgFriendInviteFailed)
	}
	s.setActiveRequester(p, nil)
	s.onTransactionResponse(requestor)
}

// l2FriendOf builds L2Friend about a player (Mobius L2Friend(Player, int)).
func l2FriendOf(p *Player, action int32) []byte {
	return serverpackets.NewL2FriendPacket(serverpackets.L2Friend{
		Action: action, Name: p.Name(), Online: p.state == stateInGame, ObjectID: p.ObjectID(),
	})
}

// handleFriendList handles RequestFriendList (0x60), /friendlist: the
// friends with their online state as system messages (Mobius
// RequestFriendList).
func (s *Server) handleFriendList(p *Player, _ []byte) {
	p.SendSystemMessage(sysMsgFriendsListHeader)
	for _, e := range p.social.friends {
		id := sysMsgFriendOffline
		if _, ok := s.onlinePlayer(e.id); ok {
			id = sysMsgFriendOnline
		}
		p.SendSystemMessage(id, textParam(e.name))
	}
	p.SendSystemMessage(sysMsgListFooter)
}

// handleFriendDel handles RequestFriendDel (0x61) (Mobius
// RequestFriendDel): the friendship goes both ways, also when the friend is
// offline. Only a friend online gets L2Friend, as in Mobius.
func (s *Server) handleFriendDel(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestFriendDel(body)
	if err != nil {
		s.log.Debug("game: bad RequestFriendDel", append(p.logArgs(), "err", err)...)
		return
	}
	i := indexName(p.social.friends, req.Name)
	if i < 0 {
		p.SendSystemMessage(sysMsgNotOnFriendList, textParam(req.Name))
		return
	}
	id := p.social.friends[i].id
	s.saveRelations(nil, storage.FriendRelations(p.ObjectID(), id))
	p.SendSystemMessage(sysMsgDeletedFromFriends, textParam(req.Name))
	p.social.friends = slices.Delete(p.social.friends, i, i+1)
	if target, ok := s.PlayerByName(req.Name); ok {
		if j := indexEntry(target.social.friends, p.ObjectID()); j >= 0 {
			target.social.friends = slices.Delete(target.social.friends, j, j+1)
		}
		p.Send(l2FriendOf(target, serverpackets.L2FriendRemove))
		target.Send(l2FriendOf(p, serverpackets.L2FriendRemove))
	}
}

// handleFriendMessage handles RequestSendFriendMsg (0xCC) (Mobius
// RequestSendFriendMsg): 1 to 300 characters, to a player online who has
// the sender on its friend list.
func (s *Server) handleFriendMessage(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSendFriendMsg(body)
	if err != nil {
		s.log.Debug("game: bad RequestSendFriendMsg", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Message == "" || chatLength(req.Message) > friendMessageMax {
		return
	}
	target, ok := s.PlayerByName(req.Receiver)
	if !ok || !target.IsFriend(p.ObjectID()) {
		p.SendSystemMessage(serverpackets.SysMsgPlayerNotOnline)
		return
	}
	s.log.Debug("friend message", append(p.logArgs(), "to", target.Name(), "text", req.Message)...)
	target.Send(serverpackets.NewL2FriendSayPacket(p.Name(), req.Receiver, req.Message))
}

// handleBlock handles RequestBlock (0xA0) (Mobius RequestBlock and
// BlockList): block, unblock, show the list, refuse or accept all.
func (s *Server) handleBlock(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestBlock(body)
	if err != nil {
		s.log.Debug("game: bad RequestBlock", append(p.logArgs(), "err", err)...)
		return
	}
	switch req.Type {
	case clientpackets.BlockAdd, clientpackets.BlockRemove:
		block := req.Type == clientpackets.BlockAdd
		if !block {
			// Unblocking someone on the list needs no lookup.
			if i := indexName(p.social.blocks, req.Name); i >= 0 {
				s.blockTarget(p, false, p.social.blocks[i].id, p.social.blocks[i].name, storage.AccessLevelPlayer)
				return
			}
		}
		s.lookupCharacter(p, req.Name, func(id int32, name string, access storage.AccessLevel, found bool) {
			if !found {
				p.SendSystemMessage(sysMsgIgnoreRegisterFailed)
				return
			}
			s.blockTarget(p, block, id, name, access)
		})
	case clientpackets.BlockShowList:
		p.SendSystemMessage(sysMsgIgnoreListHeader)
		for i, e := range p.social.blocks {
			p.SendMessage(strconv.Itoa(i+1) + ". " + e.name)
		}
		p.SendSystemMessage(sysMsgListFooter)
	case clientpackets.BlockRefuseAll, clientpackets.BlockAcceptAll:
		refuse := req.Type == clientpackets.BlockRefuseAll
		if refuse {
			p.SendSystemMessage(sysMsgRefusalMode)
		} else {
			p.SendSystemMessage(sysMsgAcceptanceMode)
		}
		p.social.refuseAll = refuse
		p.Send(etcStatusPacket(p)) // Mobius setMessageRefusal
	default:
		s.log.Info("game: unknown RequestBlock type", append(p.logArgs(), "type", req.Type)...)
	}
}

// blockTarget blocks or unblocks a character that exists (Mobius RequestBlock
// with BlockList.addToBlockList and removeFromBlockList).
func (s *Server) blockTarget(p *Player, block bool, id int32, name string, access storage.AccessLevel) {
	switch {
	case access > storage.AccessLevelPlayer:
		p.SendSystemMessage(sysMsgCannotBlockGM)
		return
	case id == p.ObjectID():
		return
	}
	rel := storage.Relation{Owner: p.ObjectID(), Target: id, Type: storage.RelationBlock}
	if !block {
		i := indexEntry(p.social.blocks, id)
		if i < 0 {
			p.SendSystemMessage(sysMsgIncorrectTarget)
			return
		}
		p.social.blocks = slices.Delete(p.social.blocks, i, i+1)
		s.saveRelations(nil, []storage.Relation{rel})
		p.SendSystemMessage(sysMsgRemovedFromIgnore, textParam(name))
		return
	}
	switch {
	case p.IsFriend(id):
		p.SendSystemMessage(sysMsgAlreadyFriend)
		return
	case p.HasBlocked(id):
		p.SendMessage("Already in ignore list.")
		return
	}
	p.social.blocks = append(p.social.blocks, socialEntry{id: id, name: name})
	s.saveRelations([]storage.Relation{rel}, nil)
	p.SendSystemMessage(sysMsgAddedToIgnore, textParam(name))
	if target, ok := s.onlinePlayer(id); ok {
		target.SendSystemMessage(SysMsgPlacedOnIgnoreList, textParam(p.Name()))
	}
}

// lookupCharacter finds the character called name, in the game or in the
// store (Mobius CharInfoTable.getIdByName and getAccessLevelById), and runs
// then on the loop with what it found, unless p left meanwhile. A character
// in the game is found at once; otherwise the store is read off the loop.
func (s *Server) lookupCharacter(p *Player, name string, then func(id int32, name string, access storage.AccessLevel, found bool)) {
	if q, ok := s.PlayerByName(name); ok {
		then(q.ObjectID(), q.Name(), q.AccessLevel(), true)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), relationLoadTimeout)
		defer cancel()
		c, err := s.chars.GetByName(ctx, name)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			s.log.Warn("game: cannot look up a character", append(p.logArgs(), "name", name, "err", err)...)
		}
		s.loop.Post(func() {
			if p.state == stateInGame {
				then(c.ObjectID, c.Name, c.AccessLevel, err == nil)
			}
		})
	}()
}
