package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The GM commands of death, levels and karma (Mobius AdminKill, AdminRes,
// AdminLevel, AdminExpSp and AdminEditChar): //kill, //res, //set_level,
// //add_level, //add_exp_sp, //remove_exp_sp, //setkarma and
// //set_pvp_flag. The exp/SP panel is in admin_char.go.

// gmSysMsgIncorrectTarget is "That is the incorrect target." (Mobius
// THAT_IS_THE_INCORRECT_TARGET).
const gmSysMsgIncorrectTarget serverpackets.SystemMessageID = 144

// registerDeathGMCommands registers the commands for access level 1 and up.
func (s *Server) registerDeathGMCommands() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("kill", admin, s.gmKill),
		s.RegisterGMCommand("res", admin, s.gmRes),
		s.RegisterGMCommand("set_level", admin, s.gmSetLevel),
		s.RegisterGMCommand("add_level", admin, s.gmAddLevel),
		s.RegisterGMCommand("add_exp_sp", admin, s.gmAddExpSP),
		s.RegisterGMCommand("remove_exp_sp", admin, s.gmRemoveExpSP),
		s.RegisterGMCommand("setkarma", admin, s.gmSetKarma),
		s.RegisterGMCommand("set_pvp_flag", admin, s.gmSetPvPFlag),
	)
}

// creaturesAround returns the creatures within r of o, except o and skip
// (Mobius forEachVisibleObjectInRange), collected before anything changes.
func (s *Server) creaturesAround(o world.Object, r int32, skip world.Object) []Creature {
	var out []Creature
	for x := range s.world.VisibleInRange(o, r) {
		if c, ok := x.(Creature); ok && x != skip {
			out = append(out, c)
		}
	}
	return out
}

// gmKill is //kill (Mobius AdminKill): the target creature dies; //kill
// <name> kills that player, //kill <name> <radius> the creatures around it
// (not the player itself, as in Mobius), //kill <radius> those around the
// GM. "Invalid target." without a creature target.
func (s *Server) gmKill(p *Player, args []string) {
	if len(args) == 0 {
		c, ok := p.Target.(Creature)
		if !ok || c == nil {
			p.SendSystemMessage(gmSysMsgInvalidTarget)
			return
		}
		s.gmKillCreature(p, c)
		return
	}
	if t, ok := s.PlayerByName(args[0]); ok {
		if len(args) < 2 {
			s.gmKillCreature(p, t)
			return
		}
		radius, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			p.SendMessage("Invalid radius.")
			return
		}
		for _, c := range s.creaturesAround(t, int32(radius), p) {
			s.gmKillCreature(p, c)
		}
		p.SendMessage(fmt.Sprintf("Killed all characters within a %d unit radius.", radius))
		return
	}
	radius, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		p.SendMessage("Usage: //kill <player_name | radius>")
		return
	}
	for _, c := range s.creaturesAround(p, int32(radius), p) {
		s.gmKillCreature(p, c)
	}
	p.SendMessage(fmt.Sprintf("Killed all characters within a %d unit radius.", radius))
}

// gmKillCreature is AdminKill.kill: a hit of max HP (+ max CP for a player)
// + 1 from the GM.
func (s *Server) gmKillCreature(gm *Player, c Creature) {
	switch t := c.(type) {
	case *Player:
		v := t.stats.View()
		t.ReduceHP(float64(v.MaxHP)+float64(v.MaxCP)+1, gm)
	default:
		c.ReduceHP(float64(c.MaxHP())+1, gm)
	}
}

// gmRes is //res (Mobius AdminRes): the target, or the GM, gets up where it
// lies with the exp its death took; //res <name> resurrects that player,
// //res <radius> the players around the GM. NPCs are revived by //res only
// once the decay can be cancelled (the monster death feature).
func (s *Server) gmRes(p *Player, args []string) {
	target := p.Target
	if len(args) > 0 {
		if t, ok := s.PlayerByName(args[0]); ok {
			target = t
		} else {
			radius, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				p.SendMessage("Enter a valid player name or radius.")
				return
			}
			var players []*Player
			for v := range s.world.VisiblePlayersInRange(p, int32(radius)) {
				if vp, ok := v.(*Player); ok {
					players = append(players, vp)
				}
			}
			for _, vp := range players {
				s.gmResurrect(vp)
			}
			p.SendMessage(fmt.Sprintf("Resurrected all players within a %d unit radius.", radius))
			return
		}
	}
	if target == nil {
		target = p
	}
	if t, ok := target.(*Player); ok {
		s.gmResurrect(t)
	}
}

// gmResurrect is AdminRes.doResurrect for a player: the lost exp back, then
// the revive.
func (s *Server) gmResurrect(t *Player) {
	if !t.dead {
		return
	}
	s.RestoreExp(t, 100)
	s.RevivePlayer(t)
}

// gmTargetPlayer returns the GM's target when it is a player.
func gmTargetPlayer(p *Player) (*Player, bool) {
	t, ok := p.Target.(*Player)
	return t, ok && t != nil
}

// gmSetLevel is //set_level <level> (Mobius AdminLevel): the target player
// gets the level, 1 to MaxLevel, with the exp of its start; above it the
// level is set and the exp removed (with the target's messages), below it
// the exp is added like a reward (level-ups and all). HP, MP and CP are
// filled and the target's UserInfo and CharInfo go out.
func (s *Server) gmSetLevel(p *Player, args []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgIncorrectTarget)
		return
	}
	maxLevel := s.MaxLevel()
	usage := fmt.Sprintf("You must specify level between 1 and %d.", maxLevel)
	if len(args) == 0 {
		p.SendMessage(usage)
		return
	}
	level, err := strconv.ParseInt(args[0], 10, 8)
	if err != nil || level < 1 || int(level) > maxLevel {
		p.SendMessage(usage)
		return
	}
	have, want := t.char.Exp, s.ExpForLevel(int(level))
	switch {
	case have > want:
		t.stats.SetLevel(min(int(level), maxLevel-1))
		t.char.Level = int32(t.Level()) //nolint:gosec // levels are small
		s.RemoveExpAndSP(t, have-want, 0, true)
		p.SendMessage(fmt.Sprintf("Removed %d exp.", have-want))
	case have < want:
		s.AddExpAndSP(t, stats.Reward{Exp: want - have, ShownExp: want - have})
		p.SendMessage(fmt.Sprintf("Added %d exp.", want-have))
	}
	v := t.stats.View()
	t.SetCurrentHPMP(float64(v.MaxHP), float64(v.MaxMP))
	t.SetCurrentCP(float64(v.MaxCP))
	t.BroadcastUserInfo()
}

// gmAddLevel is //add_level <n> (Mobius AdminLevel): the target player's
// level changes by n (AddLevel). Without a player target nothing happens.
func (s *Server) gmAddLevel(p *Player, args []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		return
	}
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	n, err := strconv.ParseInt(arg, 10, 8)
	if err != nil {
		p.SendMessage("Wrong Number Format")
		return
	}
	s.AddLevel(t, int(n))
}

// gmAddExpSP is //add_exp_sp <exp> <sp> (Mobius AdminExpSp): the target
// player is told and gets the exp and SP as a reward. Mobius then opens
// its exp/SP panel, or says "Invalid target." once more.
func (s *Server) gmAddExpSP(p *Player, args []string) {
	defer s.gmExpSPPanel(p)
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		p.SendMessage("Usage: //add_exp_sp exp sp")
		return
	}
	exp, sp, ok := gmExpSPArgs(args)
	if !ok {
		p.SendMessage("Usage: //add_exp_sp exp sp")
		return
	}
	sp = min(sp, maxSP)
	if exp == 0 && sp == 0 {
		return
	}
	t.SendMessage(fmt.Sprintf("Admin is adding you %d xp and %d sp.", exp, sp))
	s.AddExpAndSP(t, stats.Reward{Exp: exp, SP: sp, ShownExp: exp, ShownSP: sp})
	p.SendMessage(fmt.Sprintf("Added %d xp and %d sp to %s.", exp, sp, t.Name()))
}

// gmRemoveExpSP is //remove_exp_sp <exp> <sp> (Mobius AdminExpSp): the
// target player is told and loses the exp and SP (RemoveExpAndSP, with its
// messages).
func (s *Server) gmRemoveExpSP(p *Player, args []string) {
	defer s.gmExpSPPanel(p)
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		p.SendMessage("Usage: //remove_exp_sp exp sp")
		return
	}
	exp, sp, ok := gmExpSPArgs(args)
	if !ok {
		p.SendMessage("Usage: //remove_exp_sp exp sp")
		return
	}
	if exp == 0 && sp == 0 {
		return
	}
	t.SendMessage(fmt.Sprintf("Admin is removing you %d xp and %d sp.", exp, sp))
	s.RemoveExpAndSP(t, exp, sp, true)
	p.SendMessage(fmt.Sprintf("Removed %d xp and %d sp from %s.", exp, sp, t.Name()))
}

// gmExpSPPanel is Mobius's exp/SP panel after //add_exp_sp and
// //remove_exp_sp (AdminExpSp.addExpSp, admin_char.go), or its "Invalid
// target.".
func (s *Server) gmExpSPPanel(p *Player) { s.adminExpSPPage(p) }

// gmExpSPArgs parses "<exp> <sp>" as two longs.
func gmExpSPArgs(args []string) (exp, sp int64, ok bool) {
	if len(args) != 2 {
		return 0, 0, false
	}
	exp, err1 := strconv.ParseInt(args[0], 10, 64)
	sp, err2 := strconv.ParseInt(args[1], 10, 64)
	return exp, sp, err1 == nil && err2 == nil
}

// gmSetKarma is //setkarma <karma> (Mobius AdminEditChar): the target
// player's karma, told with "Your Karma has been changed to $s1.". Without
// a player target nothing happens, as in Mobius.
func (s *Server) gmSetKarma(p *Player, args []string) {
	karma, err := strconv.ParseInt(strings.Join(args, " "), 10, 32)
	if err != nil {
		p.SendMessage("Usage: //setkarma <new_karma_value>")
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		return
	}
	if karma < 0 {
		p.SendMessage("You must enter a value for karma greater than or equal to 0.")
		return
	}
	old := t.Karma()
	s.SetKarma(t, int32(karma))
	t.SendSystemMessage(sysMsgKarmaChanged, serverpackets.NumberParam(int32(karma)))
	p.SendMessage(fmt.Sprintf("Successfully Changed karma for %s from (%d) to (%d).", t.Name(), old, karma))
}

// gmSetPvPFlag is //set_pvp_flag (Mobius AdminEditChar): the target
// player's flag goes from 0 to 1, or from 1 or 2 to 0 or 1, without the
// flag timer. "Invalid target." without a player target.
func (s *Server) gmSetPvPFlag(p *Player, _ []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	flag := t.PvPFlag - 1
	if flag < 0 {
		flag = -flag
	}
	s.UpdatePvPFlag(t, flag)
}
