package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// InfoProvider is a world object that describes itself to a player that
// starts to see it (Mobius WorldObject.sendInfo): its info packets (CharInfo
// and RelationChanged for a player, NpcInfo for an NPC, ...) and, while it
// moves, a MoveToLocation so that the viewer sees it walk. SendDelta asks
// every object that appears to a player; an object that is not an
// InfoProvider stays unseen. Loop-only.
type InfoProvider interface {
	InfoFor(viewer *Player) [][]byte
}

// UpdateFlusher is a world object other than a player that uses the
// coalesced updates (world.MarkUpdate): at the end of the tick FlushUpdate
// gets the merged update bits, for example to send its NpcInfo again.
// Loop-only.
type UpdateFlusher interface {
	FlushUpdate(u world.Update)
}

// Packets that never change, encoded once.
var (
	actionFailedPacket   = serverpackets.NewActionFailedPacket()
	leaveWorldPacket     = serverpackets.NewLeaveWorldPacket()
	emptySkillCoolTime   = serverpackets.NewSkillCoolTimePacket(nil)
	emptySkillList       = serverpackets.NewSkillListPacket(nil)
	emptyAbnormalStatus  = serverpackets.NewAbnormalStatusUpdatePacket(nil)
	emptyHennaInfo       = serverpackets.NewHennaInfoPacket(serverpackets.HennaInfo{})
	emptyFriendList      = serverpackets.NewFriendListPacket(nil)
	manorListPacket      = serverpackets.NewExSendManorListPacket(serverpackets.InterludeCastles())
	welcomeMessagePacket = serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: serverpackets.SysMsgWelcome})
	// enterMacroList is the empty macro list EnterWorld sends: Mobius starts
	// the revision at 1 and adds 1 before sending.
	enterMacroList = serverpackets.NewSendMacroListPacket(serverpackets.SendMacroList{Revision: 2})
	// anotherLoginPacket tells a client that the account logged in elsewhere
	// (Mobius doKickPlayer).
	anotherLoginPacket = serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: sysMsgAnotherLogin})
	// serverClosePacket is Mobius ServerClose: just the opcode.
	serverClosePacket = []byte{opServerClose}
)

const (
	// sysMsgAnotherLogin is "Another person has logged in with the same
	// account." (Mobius ANOTHER_PERSON_HAS_LOGGED_IN_WITH_THE_SAME_ACCOUNT).
	sysMsgAnotherLogin serverpackets.SystemMessageID = 421
	// opServerClose is the opcode of ServerClose, which has no fields.
	opServerClose = 0x26
)

// AddObject puts o into the world and sends the visibility packets (Mobius
// spawnMe). Loop-only.
func (s *Server) AddObject(o world.Object) error {
	d, err := s.world.Add(o)
	if err != nil {
		return err
	}
	s.SendDelta(d)
	return nil
}

// RemoveObject takes o out of the world and sends DeleteObject to everyone
// who saw it (Mobius decayMe). Loop-only.
func (s *Server) RemoveObject(o world.Object) {
	s.SendDelta(s.world.Remove(o))
}

// MoveObject sets o's position (Mobius setXYZ) and sends the visibility
// packets of a region change. For a player it then runs the OnPlayerMoved
// hooks. The movement code calls it for every position change. An object not
// in the world just gets the position. Loop-only.
func (s *Server) MoveObject(o world.Object, pos world.Position) error {
	from := o.Position()
	d, err := s.world.Move(o, pos)
	if err != nil {
		return err
	}
	s.SendDelta(d)
	if p, ok := o.(*Player); ok && p.InWorld() {
		s.runMoved(p, from, pos)
	}
	return nil
}

// SendDelta turns a visibility change into packets, Disappear first: a
// DeleteObject for every pair that lost sight of each other, then the info
// of every object a player starts to see (InfoProvider). The OnForget hooks
// learn about every pair that lost sight, players or not. Loop-only.
func (s *Server) SendDelta(d world.Delta) {
	if d.Empty() {
		return
	}
	for v, o := range d.Disappear() {
		world.Send(v, serverpackets.NewDeleteObjectPacket(int32(o.ID())))
		forgetRelation(v, o)
	}
	if len(s.hooks.forget) > 0 {
		for _, o := range d.Left {
			s.runForget(d.Subject, o)
			s.runForget(o, d.Subject)
		}
	}
	for v, o := range d.Appear() {
		vp, ok := v.(*Player)
		if !ok {
			continue
		}
		if ip, ok := o.(InfoProvider); ok {
			for _, body := range ip.InfoFor(vp) {
				vp.Send(body)
			}
		}
		if n, ok := o.(*Npc); ok {
			s.npcSeen(n) // ai.go: a monster a player starts to see wakes up
		}
	}
	s.wakeSeenNpcs()
}

// forgetRelation drops what a viewer was told about a player's relation, so
// it is sent again when they meet again (Mobius PlayerAI forgets it).
func forgetRelation(v world.Viewer, o world.Object) {
	vp, ok1 := v.(*Player)
	op, ok2 := o.(*Player)
	if !ok1 || !ok2 {
		return
	}
	delete(op.relations, vp.ID())
	delete(vp.relations, op.ID())
}

// InfoFor returns what a player that starts to see p gets (Mobius
// Player.sendInfo): CharInfo, RelationChanged when the relation is new or
// changed, and MoveToLocation while p walks (InfoProvider).
func (p *Player) InfoFor(viewer *Player) [][]byte {
	out := [][]byte{p.charInfoFor(viewer)}
	if rel, ok := p.relationUpdate(viewer); ok {
		out = append(out, rel)
	}
	if m := p.Move; m != nil {
		out = append(out, moveToLocationPacket(p, m.Destination))
	}
	return out
}

// moveToLocationPacket is the MoveToLocation of o walking to dest.
func moveToLocationPacket(o world.Object, dest world.Position) []byte {
	pos := o.Position()
	return serverpackets.NewMoveToLocationPacket(serverpackets.MoveToLocation{
		ObjectID: int32(o.ID()),
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		DestX: dest.X, DestY: dest.Y, DestZ: dest.Z,
	})
}

// relationUpdate returns RelationChanged for viewer when the relation it
// knows is missing or out of date, and remembers the new one (Mobius
// RelationCache).
func (p *Player) relationUpdate(viewer *Player) ([]byte, bool) {
	r := relation{bits: p.relationTo(viewer), autoAttackable: p.autoAttackableBy(viewer)}
	if old, ok := p.relations[viewer.ID()]; ok && old == r {
		return nil, false
	}
	p.relations[viewer.ID()] = r
	return serverpackets.NewRelationChangedPacket(serverpackets.RelationChanged{
		ObjectID:       p.ObjectID(),
		Relation:       r.bits,
		AutoAttackable: r.autoAttackable,
		Karma:          p.char.Karma,
		PvPFlag:        p.PvPFlag,
	}), true
}

// flushUpdate sends the coalesced refreshes of one object at the end of the
// tick (world.SetUpdateFlush).
func (s *Server) flushUpdate(o world.Object, u world.Update) {
	switch o := o.(type) {
	case *Player:
		s.flushPlayer(o, u)
	case UpdateFlusher:
		o.FlushUpdate(u)
	}
}

// flushPlayer sends a player's merged refreshes: UserInfo to itself, then
// CharInfo to the players around it (Mobius broadcastUserInfo).
func (s *Server) flushPlayer(p *Player, u world.Update) {
	if p.state != stateInGame {
		return
	}
	if u&world.UpdateUserInfo != 0 {
		p.sendUserInfo()
	}
	if u&world.UpdateCharInfo != 0 && p.InWorld() {
		s.broadcastCharInfo(p)
	}
}

// broadcastCharInfo sends p's CharInfo to every player that sees it, and
// RelationChanged where the relation changed (Mobius broadcastCharInfo). The
// usual CharInfo is encoded once; a GM who sees an invisible p gets its own.
func (s *Server) broadcastCharInfo(p *Player) {
	var shared []byte
	for v := range s.world.VisiblePlayers(p) {
		vp, ok := v.(*Player)
		if !ok || !s.world.CanSee(v, p) {
			continue
		}
		if p.Invisible() && vp.IsGM() {
			vp.Send(p.charInfoFor(vp))
		} else {
			if shared == nil {
				shared = p.charInfoFor(vp)
			}
			vp.Send(shared)
		}
		if rel, ok := p.relationUpdate(vp); ok {
			vp.Send(rel)
		}
	}
}

// sendUserInfo sends the player its UserInfo now and remembers the numbers
// it showed (RefreshStats compares with them).
func (p *Player) sendUserInfo() { p.Send(p.userInfo()) }

// userInfo builds UserInfo (Mobius UserInfo).
func (p *Player) userInfo() []byte {
	v := p.stats.View()
	p.lastView = v
	pos := p.Position()
	recLeft, recHave := p.recommendations()
	title := p.Title()
	abnormal := abnormalMask(p.skill)
	if p.Invisible() {
		abnormal |= serverpackets.AbnormalStealth
		if p.IsGM() {
			title = "[Invisible]"
		}
	}
	return serverpackets.NewUserInfoPacket(serverpackets.UserInfo{
		X: pos.X, Y: pos.Y, Z: pos.Z,
		ObjectID:              p.ObjectID(),
		Name:                  p.char.Name,
		Race:                  p.char.Race,
		Female:                p.Female(),
		BaseClassID:           int32(p.BaseClassID()), //nolint:gosec // class IDs are small
		Level:                 int32(p.Level()),       //nolint:gosec // levels are small
		Exp:                   p.char.Exp,
		STR:                   v.STR,
		DEX:                   v.DEX,
		CON:                   v.CON,
		INT:                   v.INT,
		WIT:                   v.WIT,
		MEN:                   v.MEN,
		MaxHP:                 v.MaxHP,
		CurHP:                 p.CurHP,
		MaxMP:                 v.MaxMP,
		CurMP:                 p.CurMP,
		SP:                    p.char.SP,
		CurLoad:               p.inv.Load(),
		MaxLoad:               v.WeightLimit,
		WeaponEquipped:        p.inv.Paperdoll(storage.PaperdollRHand) != nil,
		Paperdoll:             p.inv.PacketPaperdoll(),
		PAtk:                  v.PAtk,
		PAtkSpd:               v.PAtkSpd,
		PDef:                  v.PDef,
		Evasion:               v.Evasion,
		Accuracy:              v.Accuracy,
		Critical:              v.Crit,
		MAtk:                  v.MAtk,
		MAtkSpd:               v.MAtkSpd,
		MDef:                  v.MDef,
		PvPFlag:               p.PvPFlag,
		Karma:                 p.char.Karma,
		RunSpd:                v.RunSpd,
		WalkSpd:               v.WalkSpd,
		SwimRunSpd:            v.SwimRunSpd,
		SwimWalkSpd:           v.SwimWalkSpd,
		FlyRunSpd:             v.FlyRunSpd,
		FlyWalkSpd:            v.FlyWalkSpd,
		MoveMultiplier:        v.MoveMultiplier,
		AttackSpeedMultiplier: v.AttackSpeedMultiplier,
		CollisionRadius:       v.CollisionRadius,
		CollisionHeight:       v.CollisionHeight,
		HairStyle:             p.char.HairStyle,
		HairColor:             p.char.HairColor,
		Face:                  p.char.Face,
		GM:                    p.IsGM(),
		Title:                 title,
		PKKills:               p.char.PKKills,
		PvPKills:              p.char.PvPKills,
		AbnormalEffects:       abnormal,
		InWater:               p.Zones.Inside(ZoneWater),
		RecomLeft:             recLeft,
		RecomHave:             recHave,
		InventoryLimit:        int16(min(v.InventoryLimit, 32767)), //nolint:gosec // clamped
		ClassID:               p.char.ClassID,
		MaxCP:                 v.MaxCP,
		CurCP:                 p.CurCP,
		EnchantEffect:         p.enchantEffect(),
		Noble:                 p.nobleFlag(),
		HeroAura:              p.IsHero(),
		NameColor:             p.nameColor(),
		Running:               p.Running(),
		TitleColor:            p.titleColor(),
	})
}

// charInfoFor builds the CharInfo viewer gets about p (Mobius CharInfo): a GM
// who sees an invisible p gets the title "Invisible" and the stealth effect
// instead of the invisible flag.
func (p *Player) charInfoFor(viewer *Player) []byte {
	v := p.stats.View()
	pos := p.Position()
	gmSeesInvisible := p.Invisible() && viewer.IsGM()
	title := p.Title()
	abnormal := abnormalMask(p.skill)
	if gmSeesInvisible {
		title = "Invisible"
		abnormal |= serverpackets.AbnormalStealth
	}
	recLeft, recHave := p.recommendations()
	return serverpackets.NewCharInfoPacket(serverpackets.CharInfo{
		X: pos.X, Y: pos.Y, Z: pos.Z,
		ObjectID:              p.ObjectID(),
		Name:                  p.char.Name,
		Race:                  p.char.Race,
		Female:                p.Female(),
		BaseClassID:           int32(p.BaseClassID()), //nolint:gosec // class IDs are small
		Paperdoll:             p.inv.PacketPaperdoll(),
		PvPFlag:               p.PvPFlag,
		Karma:                 p.char.Karma,
		MAtkSpd:               v.MAtkSpd,
		PAtkSpd:               v.PAtkSpd,
		RunSpd:                v.RunSpd,
		WalkSpd:               v.WalkSpd,
		SwimRunSpd:            v.SwimRunSpd,
		SwimWalkSpd:           v.SwimWalkSpd,
		FlyRunSpd:             v.FlyRunSpd,
		FlyWalkSpd:            v.FlyWalkSpd,
		MoveMultiplier:        v.MoveMultiplier,
		AttackSpeedMultiplier: v.AttackSpeedMultiplier,
		CollisionRadius:       v.CollisionRadius,
		CollisionHeight:       v.CollisionHeight,
		HairStyle:             p.char.HairStyle,
		HairColor:             p.char.HairColor,
		Face:                  p.char.Face,
		Title:                 title,
		Sitting:               p.Sitting,
		Running:               p.Running(),
		InCombat:              p.srv.InCombat(p),
		Dead:                  p.dead,
		Invisible:             !gmSeesInvisible && p.Invisible(),
		AbnormalEffects:       abnormal,
		RecomLeft:             recLeft,
		RecomHave:             recHave,
		ClassID:               p.char.ClassID,
		MaxCP:                 v.MaxCP,
		CurCP:                 p.CurCP,
		EnchantEffect:         p.enchantEffect(),
		Noble:                 p.nobleFlag(),
		HeroAura:              p.IsHero(),
		NameColor:             p.nameColor(),
		Heading:               pos.Heading,
		TitleColor:            p.titleColor(),
	})
}

// validateLocation is the ValidateLocation of o at its position.
func validateLocation(o world.Object) []byte {
	pos := o.Position()
	return serverpackets.NewValidateLocationPacket(serverpackets.ValidateLocation{
		ObjectID: int32(o.ID()),
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		Heading: pos.Heading,
	})
}
