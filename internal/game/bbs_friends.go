package game

import (
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/bbs"
	"github.com/IPeter/l2go/internal/storage"
)

// The friends board (Mobius FriendsBoard): the friend and block lists, with
// a selection to delete from or (for friends) to write to.

// friendsBoard handles "_friendlist", "_friendblocklist", "_friend;..." and
// "_block;..." (Mobius FriendsBoard.onCommand).
func (s *Server) friendsBoard(p *Player, command string) {
	switch {
	case command == "_friendlist" || command == "_friendlist_0_":
		s.rememberBoardPage(p, "Friends List", command)
		s.showFriendsList(p, false)
	case command == "_friendblocklist":
		s.rememberBoardPage(p, "Ignore list", command)
		s.showBlockList(p, false)
	case strings.HasPrefix(command, "_friend"):
		action, id, ok := listAction(command)
		if ok {
			s.friendAction(p, action, id)
		}
	case strings.HasPrefix(command, "_block"):
		action, id, ok := listAction(command)
		if ok {
			s.blockAction(p, action, id)
		}
	}
}

// listAction splits "_friend;<action>[;<id>]": the ID is 0 without one; ok
// is false without an action or with an ID that is no number (Mobius fails
// on both).
func listAction(command string) (action string, id int32, ok bool) {
	tokens := bbs.Tokens(command, ';')
	if len(tokens) < 2 {
		return "", 0, false
	}
	if len(tokens) > 2 && (tokens[1] == "select" || tokens[1] == "deselect") {
		if id, ok = parseJavaInt(tokens[2]); !ok {
			return "", 0, false
		}
	}
	return tokens[1], id, true
}

// selectID adds id to a selection once (Mobius selectFriend, selectBlock).
func selectID(sel []int32, id int32) []int32 {
	if slices.Contains(sel, id) {
		return sel
	}
	return append(sel, id)
}

// friendAction runs an action of the friend list.
func (s *Server) friendAction(p *Player, action string, id int32) {
	st := s.boardState(p)
	switch action {
	case "select":
		st.selFriends = selectID(st.selFriends, id)
		s.showFriendsList(p, false)
	case "deselect":
		st.selFriends = slices.DeleteFunc(st.selFriends, func(v int32) bool { return v == id })
		s.showFriendsList(p, false)
	case "delall":
		s.deleteFriends(p, p.FriendIDs(), false)
		st.selFriends = nil
		s.showFriendsList(p, false)
		p.SendMessage("You have cleared your friend list.")
		p.Send(s.friendListPacket(p))
	case "delconfirm":
		s.showFriendsList(p, true)
	case "del":
		s.deleteFriends(p, st.selFriends, true)
		st.selFriends = nil
		s.showFriendsList(p, false)
		p.Send(s.friendListPacket(p))
	case "mail":
		var names []string
		for _, id := range st.selFriends {
			if i := indexEntry(p.social.friends, id); i >= 0 {
				names = append(names, p.social.friends[i].name)
			}
		}
		if len(st.selFriends) == 0 {
			p.SendMessage("Please select character.")
			return
		}
		if html, ok := s.boardFile(p, "friends/friend-mail.html"); ok {
			s.sendBoard(p, strings.ReplaceAll(html, "%list%", strings.Join(names, ";")))
		}
	}
}

// deleteFriends removes friends of p on both sides, in one write; the
// friends online get their FriendList again. With announce, p is told each
// name (Mobius "del"; "delall" says one line instead). Mobius's "Delete
// All" names columns its table does not have, so it only clears the lists
// until the next login, and neither of its deletes updates the friend's own
// list; here both sides lose the friendship, as with RequestFriendDel.
func (s *Server) deleteFriends(p *Player, ids []int32, announce bool) {
	var removed []storage.Relation
	for _, id := range slices.Clone(ids) {
		i := indexEntry(p.social.friends, id)
		if i < 0 {
			continue
		}
		name := p.social.friends[i].name
		removed = append(removed, storage.FriendRelations(p.ObjectID(), id)...)
		p.social.friends = slices.Delete(p.social.friends, i, i+1)
		if friend, ok := s.onlinePlayer(id); ok {
			if j := indexEntry(friend.social.friends, p.ObjectID()); j >= 0 {
				friend.social.friends = slices.Delete(friend.social.friends, j, j+1)
			}
			friend.Send(s.friendListPacket(friend))
		}
		if announce {
			p.SendSystemMessage(sysMsgDeletedFromFriends, textParam(name))
		}
	}
	if len(removed) > 0 {
		s.saveRelations(nil, removed)
	}
}

// blockAction runs an action of the block list.
func (s *Server) blockAction(p *Player, action string, id int32) {
	st := s.boardState(p)
	switch action {
	case "select":
		st.selBlocks = selectID(st.selBlocks, id)
		s.showBlockList(p, false)
	case "deselect":
		st.selBlocks = slices.DeleteFunc(st.selBlocks, func(v int32) bool { return v == id })
		s.showBlockList(p, false)
	case "delall":
		for _, e := range slices.Clone(p.social.blocks) {
			s.blockTarget(p, false, e.id, e.name, storage.AccessLevelPlayer)
		}
		st.selBlocks = nil
		s.showBlockList(p, false)
	case "delconfirm":
		s.showBlockList(p, true)
	case "del":
		for _, id := range st.selBlocks {
			name := ""
			if i := indexEntry(p.social.blocks, id); i >= 0 {
				name = p.social.blocks[i].name
			}
			// Mobius BlockList.removeFromBlockList: "That is the incorrect
			// target." for one not on the list.
			s.blockTarget(p, false, id, name, storage.AccessLevelPlayer)
		}
		st.selBlocks = nil
		s.showBlockList(p, false)
	}
}

// listEntries are a friend or block list with the online states.
func (s *Server) listEntries(list []socialEntry) []bbs.ListEntry {
	out := make([]bbs.ListEntry, len(list))
	for i, e := range list {
		_, online := s.onlinePlayer(e.id)
		out[i] = bbs.ListEntry{ID: e.id, Name: e.name, Online: online}
	}
	return out
}

// showFriendsList shows the friend list page; with confirm it asks whether
// to delete every friend.
func (s *Server) showFriendsList(p *Player, confirm bool) {
	html, ok := s.boardFile(p, "friends/friends_list.html")
	if !ok {
		return
	}
	entries, sel := s.listEntries(p.social.friends), s.boardState(p).selFriends
	html = strings.ReplaceAll(html, "%friendslist%", bbs.ListRows("_friend", entries, sel, false))
	html = strings.ReplaceAll(html, "%selectedFriendsList%", bbs.ListRows("_friend", entries, sel, true))
	box := ""
	if confirm {
		box = bbs.FriendDeleteAllBox
	}
	s.sendBoard(p, strings.ReplaceAll(html, "%deleteMSG%", box))
}

// showBlockList shows the block list page; with confirm it asks whether to
// unblock everyone.
func (s *Server) showBlockList(p *Player, confirm bool) {
	html, ok := s.boardFile(p, "friends/friends_block_list.html")
	if !ok {
		return
	}
	entries, sel := s.listEntries(p.social.blocks), s.boardState(p).selBlocks
	html = strings.ReplaceAll(html, "%blocklist%", bbs.ListRows("_block", entries, sel, false))
	html = strings.ReplaceAll(html, "%selectedBlocksList%", bbs.ListRows("_block", entries, sel, true))
	box := ""
	if confirm {
		box = bbs.BlockDeleteAllBox
	}
	s.sendBoard(p, strings.ReplaceAll(html, "%deleteMSG%", box))
}
