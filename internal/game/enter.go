package game

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// enterDelay is how long after EnterWorld Mobius sends the skill list and
// the buff icons (Player.sendSkillList and EffectList.updateEffectIcons
// schedule them 300 ms later).
const enterDelay = 300 * time.Millisecond

// SelectCharacter loads the character in the given slot of the session's
// account and sends CharSelected (Mobius CharacterSelect). Another player of
// the same account is kicked out of the game first (decision E7), and the
// character is loaded only once its pending saves are written, so the new
// session sees where the old one left it. The loading happens on the
// calling goroutine; the loop only registers the player (Service).
func (s *Server) SelectCharacter(ctx context.Context, sess Session, slot int) error {
	account := sess.Account()
	unlock := s.selecting.lock(account)
	defer unlock()

	objectID, err := s.slotObjectID(ctx, sess, slot)
	if err != nil {
		return err
	}

	var already bool
	if err := s.loop.Call(ctx, func() {
		if s.bySession[sess] != nil {
			already = true
			return
		}
		if old := s.byAccount[account]; old != nil {
			s.kick(old)
		}
	}); err != nil {
		return fmt.Errorf("game: select a character: %w", err)
	}
	if already {
		return ErrAlreadySelected
	}
	if err := s.saver.wait(ctx, objectID); err != nil {
		return fmt.Errorf("game: wait for the last save of character %d: %w", objectID, err)
	}

	c, err := s.chars.Get(ctx, objectID)
	if err != nil {
		return fmt.Errorf("game: load character %d: %w", objectID, err)
	}
	if c.AccessLevel < storage.AccessLevelPlayer {
		sess.CloseAfter(serverClosePacket, 0)
		return fmt.Errorf("%w: %s (%d)", ErrCharacterBanned, c.Name, c.ObjectID)
	}
	s.migrateCharacter(&c) // subclass.go: the class-index layout (decision X3)
	p, err := s.newPlayer(sess, c)
	if err != nil {
		return err
	}
	s.loadRelations(ctx, p)          // friends.go: the friend and block lists
	summons := s.loadSummons(ctx, p) // summon_store.go: the summons to restore

	if err := s.loop.Call(ctx, func() {
		if s.bySession[sess] != nil {
			already = true
			return
		}
		if other := s.byAccount[account]; other != nil {
			// Only another session racing past the account lock gets here.
			s.kick(other)
		}
		s.bySession[sess] = p
		s.byAccount[account] = p
		sess.SetState(StateEntering)
		s.loadWornItems(p) // item_skill.go: the worn items' skills come with the inventory
		s.loadHennas(p)    // henna.go: the dyes' stats before the HP
		s.loadSubclass(p)  // subclass.go: the active class's dyes, the noble skills
		s.loadHero(p)      // hero.go: a claimed hero's flag and skills
		s.loadSkills(p)    // skill_persist.go: skills before the HP, as Mobius loads them
		s.loadItemState(p) // item_skill.go: shortcuts, item reuse, grade penalty
		s.restoreClan(p)   // clan.go: the clan, privileges and pledge class
		s.summonsSelected(p, summons)
		p.Send(s.charSelected(p, sess))
	}); err != nil {
		return fmt.Errorf("game: select a character: %w", err)
	}
	if already {
		return ErrAlreadySelected
	}
	s.log.Info("character selected", p.logArgs()...)
	return nil
}

// slotObjectID returns the object ID of the character in slot: from the
// session's last character list when it keeps one (CharacterSlots),
// otherwise from the account's characters in list order.
func (s *Server) slotObjectID(ctx context.Context, sess Session, slot int) (int32, error) {
	if cs, ok := sess.(CharacterSlots); ok {
		if slot < 0 || slot > math.MaxInt32 {
			return 0, fmt.Errorf("%w: slot %d", ErrNoCharacter, slot)
		}
		id, ok := cs.SlotObjectID(int32(slot))
		if !ok {
			return 0, fmt.Errorf("%w: slot %d", ErrNoCharacter, slot)
		}
		return id, nil
	}
	chars, err := s.chars.ListByAccount(ctx, sess.Account())
	if err != nil {
		return 0, fmt.Errorf("game: list the characters of %s: %w", sess.Account(), err)
	}
	if slot < 0 || slot >= len(chars) {
		return 0, fmt.Errorf("%w: slot %d of %d", ErrNoCharacter, slot, len(chars))
	}
	return chars[slot].ObjectID, nil
}

// charSelected builds CharSelected with the game time (Mobius CharSelected).
func (s *Server) charSelected(p *Player, sess Session) []byte {
	v := p.stats.View()
	pos := p.Position()
	return serverpackets.NewCharSelectedPacket(serverpackets.CharSelected{
		Name:      p.char.Name,
		ObjectID:  p.ObjectID(),
		Title:     p.Title(),
		SessionID: sess.PlayKey1(),
		ClanID:    p.char.ClanID,
		Female:    p.Female(),
		Race:      p.char.Race,
		ClassID:   p.char.ClassID,
		X:         pos.X, Y: pos.Y, Z: pos.Z,
		CurHP:    p.CurHP,
		CurMP:    p.CurMP,
		SP:       p.char.SP,
		Exp:      p.char.Exp,
		Level:    int32(p.Level()), //nolint:gosec // levels are small
		Karma:    p.char.Karma,
		PKKills:  p.char.PKKills,
		INT:      v.INT,
		STR:      v.STR,
		CON:      v.CON,
		MEN:      v.MEN,
		DEX:      v.DEX,
		WIT:      v.WIT,
		GameTime: s.world.GameTime().Minutes(),
	})
}

// EnterWorld puts the selected character into the world on the loop and
// waits for it (Service). Without a selected character the connection is
// closed with LeaveWorld, as Mobius does.
func (s *Server) EnterWorld(sess Session) {
	s.runFor(sess, func() {
		p := s.playerOf(sess)
		switch {
		case p == nil:
			s.log.Warn("game: EnterWorld without a selected character", "remote", sess.RemoteAddr())
			sess.CloseAfter(leaveWorldPacket, 0)
		case p.state != stateSelected:
			s.log.Debug("game: EnterWorld repeated, ignored", p.logArgs()...)
		default:
			s.enterWorld(p)
		}
	})
}

// enterWorld runs Mobius EnterWorld.runImpl for what exists so far. The
// packets, in Mobius order:
//
//  1. UserInfo
//  2. SendMacroList (empty, revision 2)
//  3. ItemList (window closed)
//  4. ShortcutInit
//  5. HennaInfo
//  6. QuestList (empty)
//  7. the spawn: CharInfo and RelationChanged both ways with the players
//     around, and the info of every other visible object
//  8. EtcStatusUpdate
//  9. ExStorageMaxCount
//  10. FriendList, and "$s1 (Friend) has logged in." to the friends online
//  11. SystemMessage "Welcome to the World of Lineage II."
//  12. SkillCoolTime (empty)
//  13. UserInfo again, and CharInfo to the players around
//     (broadcastUserInfo)
//  14. ValidateLocation
//  15. ActionFailed
//
// and 300 ms later AbnormalStatusUpdate and SkillList (both empty). The
// OnPlayerEnter hooks run between 11 and 12, where Mobius calls
// onPlayerEnter. A clan member gets the clan packets after the first
// UserInfo (clanEnterWorld) and the join penalty notice after SkillCoolTime;
// Seven Signs, announcement and GM startup packets are left out (decision
// E1).
func (s *Server) enterWorld(p *Player) {
	sess := p.session
	sess.SetState(StateInGame)
	p.state = stateInGame

	p.sendUserInfo()
	s.clanEnterWorld(p) // clan.go: PledgeSkillList, the members' notices, the clan window
	p.Send(enterMacroList)
	p.Send(serverpackets.NewItemListPacket(serverpackets.ItemList{Items: p.inv.ItemInfos()}))
	p.Send(shortcutInitPacket(p))
	p.Send(hennaInfoPacket(p))
	s.questsEnterWorld(p) // quest.go: the quests load, the tutorial, QuestList

	if err := s.world.AddPlayer(p); err != nil {
		// Mobius disconnects both players of a duplicate object ID.
		s.log.Error("game: the character is in the game twice", append(p.logArgs(), "err", err)...)
		s.leave(p, false, nil)
		sess.CloseAfter(serverClosePacket, 0)
		return
	}
	s.byName[storage.NameKey(p.char.Name)] = p
	_ = p.SetPosition(spawnPosition(p.Position())) // not in the world yet
	if err := s.AddObject(p); err != nil {
		s.log.Error("game: cannot put the character into the world", append(p.logArgs(), "pos", p.Position(), "err", err)...)
		s.leave(p, false, nil)
		sess.CloseAfter(serverClosePacket, 0)
		return
	}
	// Mobius onSpawn looks at the zones around the new position.
	s.runMoved(p, p.Position(), p.Position())
	if p.state != stateInGame {
		return // a hook made the player leave
	}
	// Mobius updateEffectIcons sends the buff icons 300 ms after the spawn.
	p.Effects().UpdateIcons(false)

	p.Send(etcStatusPacket(p))
	p.Send(s.storageMaxCount(p))
	s.sendFriendsOnEnter(p) // friends.go: FriendList, the friends' log-in notices
	p.Send(welcomeMessagePacket)

	s.runEnter(p)
	if p.state != stateInGame {
		return // a hook made the player leave
	}

	// Mobius sendSkillList sends the list 300 ms later.
	p.SendSkillList()
	p.Send(s.SkillCoolTimePacket(p))
	s.clanEnterWorldLate(p) // clan_members.go: the join penalty notice

	p.BroadcastUserInfo()
	s.world.FlushUpdates()
	p.Send(validateLocation(p))
	p.SendActionFailed()

	if iv := s.settings.AutosaveInterval; iv > 0 {
		p.autosave = s.loop.Every(iv, func() {
			if p.state == stateInGame {
				s.Save(p, nil)
			}
		})
	}
	s.log.Info("player entered the world", append(p.logArgs(), "pos", p.Position(), "online", s.world.PlayerCount())...)
}

// after runs fn on the loop after d, unless the player has left by then.
func (p *Player) after(d time.Duration, fn func()) {
	var id world.TimerID
	id = p.srv.loop.After(d, func() {
		p.dropTimer(id)
		if p.state == stateInGame {
			fn()
		}
	})
	p.timers = append(p.timers, id)
}

func (p *Player) dropTimer(id world.TimerID) {
	for i, t := range p.timers {
		if t == id {
			p.timers = append(p.timers[:i], p.timers[i+1:]...)
			return
		}
	}
}

// storageMaxCount builds ExStorageMaxCount (Mobius ExStorageMaxCount with
// the Player.ini defaults and the configured warehouse sizes).
func (s *Server) storageMaxCount(p *Player) []byte {
	const (
		privateSellOther = 3 // MaxPvtStoreSellSlotsOther
		privateSellDwarf = 4 // MaxPvtStoreSellSlotsDwarf
		privateBuyOther  = 4 // MaxPvtStoreBuySlotsOther
		privateBuyDwarf  = 5 // MaxPvtStoreBuySlotsDwarf
	)
	var sell, buy int32 = privateSellOther, privateBuyOther
	if p.char.Race == stats.RaceDwarf {
		sell, buy = privateSellDwarf, privateBuyDwarf
	}
	// The limits add the modifiers of their stat (calcStat with base 0);
	// Mobius casts each to int.
	bonus := func(s stats.Stat) int32 { return int32(p.stats.Calc(s, 0)) }
	return serverpackets.NewExStorageMaxCountPacket(serverpackets.ExStorageMaxCount{
		Inventory:      p.stats.View().InventoryLimit,
		Warehouse:      int32(min(s.warehouseLimit(p), math.MaxInt32)),           //nolint:gosec // clamped
		ClanWarehouse:  int32(min(s.gs.Items.WarehouseSlotsClan, math.MaxInt32)), //nolint:gosec // clamped
		PrivateSell:    sell + bonus(stats.PrivateSellLimit),
		PrivateBuy:     buy + bonus(stats.PrivateBuyLimit),
		DwarfRecipe:    int32(s.recipeLimit(p, true)),  //nolint:gosec // small limits (craft.go)
		CommonRecipe:   int32(s.recipeLimit(p, false)), //nolint:gosec // small limits
		InventoryExtra: bonus(stats.InventoryLimit),
	})
}

// spawnPosition keeps a spawn point inside the world the way Mobius spawnMe
// does: 5000 units inside the border it crossed.
func spawnPosition(pos world.Position) world.Position {
	const margin = 5000
	switch {
	case pos.X >= world.WorldXMax:
		pos.X = world.WorldXMax - margin
	case pos.X < world.WorldXMin:
		pos.X = world.WorldXMin + margin
	}
	switch {
	case pos.Y >= world.WorldYMax:
		pos.Y = world.WorldYMax - margin
	case pos.Y < world.WorldYMin:
		pos.Y = world.WorldYMin + margin
	}
	return pos
}
