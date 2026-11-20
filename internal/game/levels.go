package game

import (
	"math"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// Experience, SP and levels of players: Mobius PlayableStat and PlayerStat
// (addExp, removeExp, addLevel, addSp, removeSp, addExpAndSp,
// removeExpAndSp). Exp and SP live in the character (Player.char), so they
// are saved with it; the level is the stat set's.
//
// The API for the kill rewards, quests and GM commands:
//
//   - Server.AddExpAndSP adds a reward (stats.Reward from
//     CombatRules.SoloReward) with "You have earned ..." and UserInfo.
//   - Server.AddExp and Server.RemoveExp change the exp silently, with the
//     level-ups and delevels they cause; AddExp also burns karma.
//   - Server.RemoveExpAndSP takes exp and SP with their messages.
//   - Server.AddLevel changes the level (level-up: full HP/MP/CP,
//     SocialAction 15, "Your level has increased!"; UserInfo).
//   - Server.MaxLevel and Server.ExpForLevel read the experience table.

// playerMaximumLevel is Mobius MaximumPlayerLevel (Player.ini): the
// experience table's maxLevel + 1 is capped to it.
const playerMaximumLevel = 80

// maxSP is the most SP a player holds (Mobius MaxSp; SP are an int here).
const maxSP = math.MaxInt32

// System messages of exp and levels.
const (
	sysMsgEarnedExp      serverpackets.SystemMessageID = 45   // "You have earned $s1 experience."
	sysMsgEarnedExpAndSP serverpackets.SystemMessageID = 95   // "You have earned $s1 experience and $s2 SP."
	sysMsgLevelIncreased serverpackets.SystemMessageID = 96   // "Your level has increased!"
	sysMsgAcquiredSP     serverpackets.SystemMessageID = 331  // "You have acquired $s1 SP."
	sysMsgSPDecreased    serverpackets.SystemMessageID = 538  // "Your SP has decreased by $s1."
	sysMsgExpDecreased   serverpackets.SystemMessageID = 539  // "Your Experience has decreased by $s1."
	sysMsgKarmaChanged   serverpackets.SystemMessageID = 1282 // "Your Karma has been changed to $s1."
)

// MaxLevel is Mobius ExperienceData.getMaxLevel: the table's maxLevel + 1,
// at most MaximumPlayerLevel (80). Players reach MaxLevel() - 1 at most.
// Safe on any goroutine.
func (s *Server) MaxLevel() int {
	return min(s.data.Experience.MaxLevel+1, playerMaximumLevel)
}

// ExpForLevel is Mobius getExpForLevel: the total exp of level, the exp of
// level 80 above it, 0 below level 1. Safe on any goroutine.
func (s *Server) ExpForLevel(level int) int64 {
	level = min(level, playerMaximumLevel)
	exp, _ := s.data.ExpForLevel(level)
	return exp
}

// Exp returns the player's experience.
func (p *Player) Exp() int64 { return p.char.Exp }

// SP returns the player's SP.
func (p *Player) SP() int32 { return p.char.SP }

// levelForExp is the level loop of PlayableStat.addExp and removeExp: the
// last level whose exp exp reaches.
func (s *Server) levelForExp(exp int64) int {
	maxLevel := s.MaxLevel()
	for tmp := 1; tmp <= maxLevel; tmp++ {
		if exp >= s.ExpForLevel(tmp) {
			continue
		}
		return tmp - 1
	}
	return 1 // Mobius keeps the minimum level when no level is above exp
}

// AddExpAndSP adds a reward to p (Mobius PlayerStat.addExpAndSp): r.Exp
// through AddExp (level-ups, karma burn), r.SP up to the SP cap, then "You
// have earned $s1 experience and $s2 SP." with r.ShownExp and r.ShownSP
// (or the message of exp or SP alone when only one of them was added) and
// UserInfo. A GM command or quest gives a plain reward:
// stats.Reward{Exp: e, SP: sp, ShownExp: e, ShownSP: sp}. Loop-only.
func (s *Server) AddExpAndSP(p *Player, r stats.Reward) {
	expAdded := s.AddExp(p, r.Exp)
	spAdded := s.addSP(p, r.SP)
	switch {
	case !expAdded && spAdded:
		p.SendSystemMessage(sysMsgAcquiredSP, serverpackets.LongNumberParam(r.SP))
	case expAdded && !spAdded:
		p.SendSystemMessage(sysMsgEarnedExp, serverpackets.LongNumberParam(r.Exp))
	case r.Exp > 0 || r.SP > 0:
		p.SendSystemMessage(sysMsgEarnedExpAndSP, serverpackets.LongNumberParam(r.ShownExp), serverpackets.LongNumberParam(r.ShownSP))
	}
	p.UpdateUserInfo()
}

// AddExp adds exp to p as Mobius PlayerStat.addExp does: nothing when the
// result would be negative or p is at the top of the last level; at most up
// to one below the exp of MaxLevel; the level follows the exp (AddLevel).
// Then, when p has karma outside a PvP zone, the exp burns karma
// (stats.CombatRules.KarmaLost with the karmaLoss.xml modifier of p's
// level) and p gets "Your Karma has been changed to $s1.". It sends no exp
// message and reports whether exp may be gained (always). Loop-only.
func (s *Server) AddExp(p *Player, amount int64) bool {
	s.changeExp(p, amount)
	if p.char.Karma > 0 && !p.Zones.Inside(ZonePvP) {
		mod, ok := s.data.KarmaLoss.At(p.Level())
		if ok && mod > 0 {
			if lost := s.combat.KarmaLost(amount, mod); lost > 0 {
				s.SetKarma(p, p.char.Karma-int32(min(lost, math.MaxInt32))) //nolint:gosec // capped
				p.SendSystemMessage(sysMsgKarmaChanged, serverpackets.NumberParam(p.char.Karma))
			}
		}
	}
	return true
}

// changeExp is PlayableStat.addExp.
func (s *Server) changeExp(p *Player, amount int64) {
	exp := p.char.Exp
	top := s.ExpForLevel(s.MaxLevel())
	if exp+amount < 0 || (amount > 0 && exp == top-1) {
		return
	}
	if exp+amount >= top {
		amount = top - 1 - exp
	}
	p.char.Exp = exp + amount
	if level := s.levelForExp(p.char.Exp); level != p.Level() && level >= 1 {
		s.AddLevel(p, level-p.Level())
	}
}

// RemoveExp takes exp from p silently, as Mobius PlayableStat.removeExp
// does: a loss bigger than the exp leaves 1 exp, and the level follows
// (AddLevel). Death exp loss uses it. Loop-only.
func (s *Server) RemoveExp(p *Player, amount int64) {
	if p.char.Exp-amount < 0 {
		amount = p.char.Exp - 1
	}
	p.char.Exp -= amount
	if level := s.levelForExp(p.char.Exp); level != p.Level() && level >= 1 {
		s.AddLevel(p, level-p.Level())
	}
}

// RemoveExpAndSP takes exp and SP from p (Mobius
// PlayerStat.removeExpAndSp): only positive amounts are taken, and nothing
// happens when neither is. With sendMessage p gets "Your Experience has
// decreased by $s1." and "Your SP has decreased by $s1.", and its
// StatusUpdate when it lost a level; then UserInfo. Loop-only.
func (s *Server) RemoveExpAndSP(p *Player, exp, sp int64, sendMessage bool) bool {
	level := p.Level()
	if exp <= 0 && sp <= 0 {
		return false
	}
	if exp > 0 {
		s.RemoveExp(p, exp)
	}
	if sp > 0 {
		s.removeSP(p, sp)
	}
	if sendMessage {
		p.SendSystemMessage(sysMsgExpDecreased, serverpackets.LongNumberParam(exp))
		p.SendSystemMessage(sysMsgSPDecreased, serverpackets.LongNumberParam(sp))
		if p.Level() < level {
			p.BroadcastStatus()
		}
	}
	p.UpdateUserInfo()
	return true
}

// addSP is PlayableStat.addSp: false for a negative amount or a player at
// the SP cap; otherwise the SP grow up to the cap.
func (s *Server) addSP(p *Player, amount int64) bool {
	cur := int64(p.char.SP)
	if amount < 0 || cur >= maxSP {
		return false
	}
	p.char.SP = int32(min(cur+amount, maxSP)) //nolint:gosec // capped
	return true
}

// removeSP is PlayableStat.removeSp: down to 0.
func (s *Server) removeSP(p *Player, amount int64) {
	cur := int64(p.char.SP)
	p.char.SP = int32(max(cur-amount, 0)) //nolint:gosec // within the old value
}

// AddLevel changes p's level by delta (Mobius PlayerStat.addLevel, which
// //add_level calls): refused beyond MaxLevel() - 1. The exp moves to the
// start of the new level when it is outside it. A level-up fills HP, MP and
// CP, shows SocialAction 15 to p and the players around and sends "Your
// level has increased!". Either way the weight penalty is checked and p
// gets its UserInfo. It reports whether the level went up. Skills follow
// in M7. Loop-only.
func (s *Server) AddLevel(p *Player, delta int) bool {
	maxLevel := s.MaxLevel()
	old := p.Level()
	if old+delta > maxLevel-1 {
		return false
	}
	level := max(old+delta, 1)
	increased := level > old
	p.stats.SetLevel(level)
	p.char.Level = int32(p.Level()) //nolint:gosec // levels are small
	if p.char.Exp >= s.ExpForLevel(p.Level()+1) || s.ExpForLevel(p.Level()) > p.char.Exp {
		p.char.Exp = s.ExpForLevel(p.Level())
	}
	if increased {
		v := p.stats.View()
		p.SetCurrentHP(float64(v.MaxHP))
		p.SetCurrentMP(float64(v.MaxMP))
		p.SetCurrentCP(float64(v.MaxCP))
		if p.InWorld() {
			s.world.Broadcast(p, serverpackets.NewSocialActionPacket(serverpackets.SocialAction{
				ObjectID: p.ObjectID(),
				ActionID: serverpackets.SocialActionLevelUp,
			}), true)
		}
		p.SendSystemMessage(sysMsgLevelIncreased)
	}
	s.refreshOverloaded(p)
	if p.state == stateInGame {
		p.UpdateUserInfo()
	}
	return increased
}

// mathRound is Java's Math.round for the values here: the closest long,
// ties up.
func mathRound(x float64) int64 {
	if x != x {
		return 0
	}
	return int64(math.Floor(x + 0.5))
}
