package game

import (
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The chat feature: Say2 and the channels of Mobius's chat handlers
// (dist/game/data/scripts/handlers/chat/channels). Its state is in
// movementFeature.
func init() { registerFeature("chat", setupChat) }

const (
	// chatMaxWithItems is the length limit of a message that links an item
	// (Mobius Say2).
	chatMaxWithItems = 500
	// chatItemMark starts and ends an item link in a message (char 8).
	chatItemMark = '\b'
	// chatTradeLevel is the level trade chat needs (Mobius ChatTrade).
	chatTradeLevel = 20
	// chatGlobalInterval is, in game ticks, how often a player below GM
	// level may speak in a global channel (Mobius
	// FloodProtectorGlobalChatInterval, 500 ms).
	chatGlobalInterval = 5
	// chatSysMsgNoPetition is "You are currently not in a petition chat."
	// (Mobius YOU_ARE_CURRENTLY_NOT_IN_A_PETITION_CHAT).
	chatSysMsgNoPetition serverpackets.SystemMessageID = 745
)

// setupChat registers Say2.
func setupChat(s *Server) error { return s.Register(clientpackets.OpSay2, s.handleSay2) }

// handleSay2 handles Say2 (0x38), a chat message (Mobius Say2):
//   - An unknown chat type or an empty text, which no client sends, gets
//     ActionFailed, and the player is saved and disconnected with
//     LeaveWorld.
//   - A text longer than Settings.Chat.MaxLength (500 with an item link)
//     from a player below GM level is refused with Mobius's message.
//   - A message linking an item the player does not have is dropped.
//   - The Lua events.on_chat handlers may drop the message or change its
//     text; a longer text is cut to the limit.
//
// Then the channel's handler sends it as CreatureSay.
func (s *Server) handleSay2(p *Player, body []byte) {
	req, err := clientpackets.ParseSay2(body)
	if err != nil {
		s.log.Debug("game: bad Say2", append(p.logArgs(), "err", err)...)
		return
	}
	typ := serverpackets.ChatType(req.Type)
	switch {
	case !typ.Valid():
		s.chatKick(p, "invalid chat type", "type", req.Type, "text", req.Text)
		return
	case req.Text == "":
		s.chatKick(p, "empty chat text")
		return
	}
	limit := s.settings.Chat.MaxLength
	text := req.Text
	if !p.IsGM() && chatLength(text) > chatLimit(text, limit) {
		p.SendSystemMessage(serverpackets.SysMsgChatTooLong)
		return
	}
	if typ == serverpackets.ChatPetitionPlayer && p.IsGM() {
		typ = serverpackets.ChatPetitionGM
	}
	if strings.ContainsRune(text, chatItemMark) && !chatItemsOwned(p, text) {
		s.log.Info("game: chat links an item the player does not have, dropped", append(p.logArgs(), "text", text)...)
		return
	}
	if eng := s.scripts; eng != nil {
		res := eng.Invoke(s.background(), scripting.ChatMessage{Player: p, Type: int(typ), Text: text, Target: req.Target}.Event())
		if res.Dropped || res.Text == "" {
			return
		}
		text = res.Text
		if !p.IsGM() {
			text = chatCut(text, chatLimit(text, limit))
		}
	}
	s.log.Debug("chat", append(p.logArgs(), "type", int32(typ), "target", req.Target, "text", text)...)

	switch typ {
	case serverpackets.ChatGeneral:
		s.chatGeneral(p, text)
	case serverpackets.ChatShout:
		s.chatChannel(p, typ, text, s.settings.Chat.Shout)
	case serverpackets.ChatTrade:
		if p.Level() < chatTradeLevel {
			p.SendMessage("Players can use Trade chat after Lv. 20.")
			return
		}
		s.chatChannel(p, typ, text, s.settings.Chat.Trade)
	case serverpackets.ChatWhisper:
		s.chatWhisper(p, req.Target, text)
	case serverpackets.ChatHeroVoice:
		s.chatHeroVoice(p, text)
	case serverpackets.ChatParty:
		p.SendMessage("You are not in a party.")
	case serverpackets.ChatClan:
		p.SendMessage("You are not in a clan.")
	case serverpackets.ChatAlliance:
		p.SendMessage("You are not in an alliance.")
	case serverpackets.ChatPetitionPlayer, serverpackets.ChatPetitionGM:
		p.SendSystemMessage(chatSysMsgNoPetition)
	default:
		// Party rooms, command channels and sieges do not exist yet, and
		// Mobius has no handler for the other types: nothing is sent.
	}
}

// chatKick disconnects a player whose Say2 no client sends (Mobius Say2):
// ActionFailed, then the player is saved and removed from the world and
// the connection closes with LeaveWorld.
func (s *Server) chatKick(p *Player, why string, args ...any) {
	s.log.Warn("game: "+why+", disconnecting the player", append(p.logArgs(), args...)...)
	p.SendActionFailed()
	sess := p.session
	s.leave(p, true, nil)
	if sess != nil {
		sess.CloseAfter(leaveWorldPacket, 0)
	}
}

// chatLimit is the length limit of text for a player below GM level
// (Mobius Say2): limit characters, or 500 with an item link.
func chatLimit(text string, limit int) int {
	if strings.ContainsRune(text, chatItemMark) {
		return chatMaxWithItems
	}
	return limit
}

// chatLength is the length of s as Java counts it, in UTF-16 units.
func chatLength(s string) int {
	n := 0
	for _, r := range s {
		n += max(utf16.RuneLen(r), 1)
	}
	return n
}

// chatCut cuts s to at most limit UTF-16 units, keeping whole characters.
func chatCut(s string, limit int) string {
	n := 0
	for i, r := range s {
		if n += max(utf16.RuneLen(r), 1); n > limit {
			return s[:i]
		}
	}
	return s
}

// chatItemsOwned checks the item links of a message (Mobius
// parseAndPublishItem): each runs from a char 8 to the next and names with
// "ID=" the object ID of an item the player has. A broken link or another
// player's item drops the message. (Mobius also publishes the items so
// that others can look at them; that comes with item links.)
func chatItemsOwned(p *Player, text string) bool {
	rest := text
	for {
		start := strings.IndexRune(rest, chatItemMark)
		if start < 0 {
			return true
		}
		rest = rest[start:]
		at := strings.Index(rest, "ID=")
		if at < 0 {
			return false
		}
		rest = rest[at+3:]
		n := 0
		for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
			n++
		}
		id, err := strconv.ParseInt(rest[:n], 10, 32)
		if err != nil || !chatHasItem(p, int32(id)) {
			return false
		}
		end := strings.IndexRune(rest[n:], chatItemMark)
		if end < 0 {
			return false // no end mark
		}
		rest = rest[n+end+1:]
	}
}

// chatHasItem reports whether p has the item with the given object ID.
func chatHasItem(p *Player, objectID int32) bool {
	return p.inv.ByObjectID(objectID) != nil
}

// chatSay is the CreatureSay of a message p sends in a channel, shown with
// name.
func chatSay(p *Player, typ serverpackets.ChatType, name, text string) []byte {
	return serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{
		ObjectID:   p.ObjectID(),
		Type:       typ,
		SenderName: name,
		Text:       text,
	})
}

// chatGeneral sends general chat (Mobius ChatGeneral) to the players around
// p within Settings.Chat.GeneralRange (3D), then back to p.
func (s *Server) chatGeneral(p *Player, text string) {
	say := chatSay(p, serverpackets.ChatGeneral, p.Name(), text)
	if p.InWorld() {
		r := int32(min(s.settings.Chat.GeneralRange, 1<<30)) //nolint:gosec // clamped
		for v := range s.world.VisiblePlayersInRange(p, r) {
			world.Send(v, say)
		}
	}
	p.Send(say)
}

// chatChannel sends shout or trade chat the way mode says (Mobius ChatShout
// and ChatTrade with GlobalChat and TradeChat): to the players in p's map
// region and instance, to every player (with the flood protector for
// players below GM level), only from GMs, or to nobody.
func (s *Server) chatChannel(p *Player, typ serverpackets.ChatType, text string, mode config.ChatMode) {
	switch mode {
	case config.ChatModeOff:
	case config.ChatModeGM:
		if p.IsGM() {
			s.chatToRegion(p, typ, text)
		}
	case config.ChatModeGlobal:
		if !s.chatFloodOK(p) {
			if typ == serverpackets.ChatTrade {
				p.SendMessage("Do not spam trade channel.")
			} else {
				p.SendMessage("Do not spam shout channel.")
			}
			return
		}
		say := chatSay(p, typ, p.Name(), text)
		for v := range s.Players() {
			v.Send(say)
		}
	default: // config.ChatModeOn, Mobius's default
		s.chatToRegion(p, typ, text)
	}
}

// chatToRegion sends a message to the players in p's map region and
// instance, p included (Mobius GlobalChat = ON).
func (s *Server) chatToRegion(p *Player, typ serverpackets.ChatType, text string) {
	region := s.chatRegionOf(p.Position())
	say := chatSay(p, typ, p.Name(), text)
	for v := range s.Players() {
		if v.InstanceID() == p.InstanceID() && s.chatRegionOf(v.Position()) == region {
			v.Send(say)
		}
	}
}

// chatRegionOf returns the map region of pos for shout and trade chat.
func (s *Server) chatRegionOf(pos world.Position) int {
	if f := s.movement.chatRegion; f != nil {
		return f(pos)
	}
	return s.chatMapRegion(pos)
}

// chatMapRegion returns the map region of pos (Mobius getMapRegionLocId):
// 0 outside every region, and without world data, where every player is
// in region 0.
func (s *Server) chatMapRegion(pos world.Position) int {
	if s.worldData == nil {
		return 0
	}
	return s.worldData.MapRegionLocID(int(pos.X), int(pos.Y))
}

// chatFloodOK is Mobius's flood protector of the global channels: a player
// below GM level may speak once per 500 ms.
func (s *Server) chatFloodOK(p *Player) bool {
	if p.IsGM() {
		return true
	}
	m := s.motionOf(p)
	tick := gameTick(s.Now())
	if tick < m.nextGlobalChat {
		return false
	}
	m.nextGlobalChat = tick + chatGlobalInterval
	return true
}

// chatWhisper sends a whisper to the player called target, in any case
// (Mobius ChatWhisper): the target gets it with p's name, and p gets it back
// with "->" and the target's name. When no such player is in the game, p
// gets "That player is not online."
func (s *Server) chatWhisper(p *Player, target, text string) {
	to, ok := s.PlayerByName(target)
	if !ok {
		p.SendSystemMessage(serverpackets.SysMsgPlayerNotOnline)
		return
	}
	to.Send(chatSay(p, serverpackets.ChatWhisper, p.Name(), text))
	p.Send(chatSay(p, serverpackets.ChatWhisper, "->"+to.Name(), text))
}

// chatHeroVoice sends hero voice to every player (Mobius ChatHeroVoice).
// Only heroes and GMs may use it; there are no heroes yet, and the flood
// protector lets GMs through.
func (s *Server) chatHeroVoice(p *Player, text string) {
	if !p.IsGM() {
		p.SendMessage("Only Heroes can enter the Hero channel.")
		return
	}
	say := chatSay(p, serverpackets.ChatHeroVoice, p.Name(), text)
	for v := range s.Players() {
		v.Send(say)
	}
}
