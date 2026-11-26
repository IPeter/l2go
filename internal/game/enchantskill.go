package game

import (
	"strconv"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/subclass"
)

// Skill enchanting at the trainers (M11e): the "EnchantSkillList" bypass
// (Mobius Folk.showEnchantSkillList), RequestExEnchantSkillInfo (D0 06) and
// RequestExEnchantSkill (D0 07). The rules are subclass.AvailableEnchants,
// CheckEnchant and EnchantSucceeds.

// System messages of skill enchanting.
const (
	sysMsgEnchantNone      serverpackets.SystemMessageID = 1438 // "There is no skill that enables enchant."
	sysMsgEnchantNoItems   serverpackets.SystemMessageID = 1439 // "You do not have all of the items needed to enchant that skill."
	sysMsgEnchantSucceeded serverpackets.SystemMessageID = 1440 // "Skill enchant was successful! $s1 has been enchanted."
	sysMsgEnchantFailed    serverpackets.SystemMessageID = 1441 // "Skill enchant failed. The skill will be initialized."
	sysMsgEnchantNoSP      serverpackets.SystemMessageID = 1443 // "You do not have enough SP to enchant that skill."
	sysMsgEnchantNoExp     serverpackets.SystemMessageID = 1444 // "You do not have enough experience (Exp) to enchant that skill."
)

// enchantBookRequirement is the requirement type ExEnchantSkillInfo sends
// with the Secret Book of Giants.
const enchantBookRequirement = 4

// bypassEnchantSkillList handles "EnchantSkillList" of a Folk NPC (Mobius
// EnchantSkillList bypass handler).
func (s *Server) bypassEnchantSkillList(p *Player, n *Npc) {
	if !s.npcIs(n, "Folk") {
		return
	}
	s.showEnchantSkillList(p, n)
}

// availableEnchants returns the enchant levels p may learn now
// (getAvailableEnchantSkills).
func (s *Server) availableEnchants(p *Player) []*data.EnchantSkill {
	return subclass.AvailableEnchants(s.subclass.enchant, p.Level(), p.SkillLevel)
}

// showEnchantSkillList is Mobius Folk.showEnchantSkillList: the no-teach
// page when n does not teach p's class; a plain page below the third class;
// otherwise ExEnchantSkillList with every enchant level p may learn whose
// skill exists, or, with none, "There is no skill that enables enchant.",
// "come back at level 74" below it or "no other skills" (and
// AcquireSkillDone). ActionFailed ends it.
func (s *Server) showEnchantSkillList(p *Player, n *Npc) {
	class := data.PlayerClass(p.ClassID()) //nolint:gosec // class IDs are small
	if !trainerCanTeach(n.Template.ID, class) {
		s.showNoTeach(p, n)
		return
	}
	if class.Level() < 3 {
		s.sendPage(p, newPage(n.ID(), "<html><body>You must have 3rd class change quest completed.</body></html>"))
		return
	}
	var entries []serverpackets.EnchantSkillEntry
	for _, e := range s.availableEnchants(p) {
		if _, ok := s.Skill(e.ID, e.Level); !ok {
			continue
		}
		entries = append(entries, serverpackets.EnchantSkillEntry{
			ID: int32(e.ID), NextLevel: int32(e.Level), SP: int32(e.SP), Exp: int64(e.Exp), //nolint:gosec // data values fit
		})
	}
	if len(entries) == 0 {
		p.SendSystemMessage(sysMsgEnchantNone)
		if p.Level() < subclass.EnchantComeBackLevel {
			p.SendSystemMessage(sysMsgLearnComeBackAtS1, serverpackets.NumberParam(subclass.EnchantComeBackLevel))
		} else {
			p.SendSystemMessage(sysMsgLearnNoOtherSkill)
		}
		p.Send(serverpackets.NewAcquireSkillDonePacket())
	} else {
		p.Send(serverpackets.NewExEnchantSkillListPacket(entries))
	}
	p.SendActionFailed()
}

// enchantTrainer returns the Folk NPC p talked to last when p is within its
// interaction range (or is a GM), else nil (Mobius getLastFolkNPC with
// isInsideRadius2D).
func (s *Server) enchantTrainer(p *Player) *Npc {
	n := s.Npc(p.LastNpc)
	if n == nil || n.Template == nil || !s.npcIs(n, "Folk") {
		return nil
	}
	if !p.IsGM() && p.Position().Distance2D(n.Position()) > float64(s.npcInteractionRange()) {
		return nil
	}
	return n
}

// offeredEnchant returns the enchant level of a skill that n offers p now:
// the skill level exists, n teaches p's class and the level is available.
func (s *Server) offeredEnchant(p *Player, n *Npc, id, level int) *data.EnchantSkill {
	if _, ok := s.Skill(id, level); !ok || !trainerCanTeach(n.Template.ID, data.PlayerClass(p.ClassID())) { //nolint:gosec // class IDs are small
		return nil
	}
	for _, e := range s.availableEnchants(p) {
		if e.ID == id && e.Level == level {
			return e
		}
	}
	return nil
}

// handleEnchantSkillInfo handles RequestExEnchantSkillInfo (D0 06, Mobius
// RequestExEnchantSkillInfo): at level 76 or more, at a trainer of p's
// class, for an enchant level it offers, ExEnchantSkillInfo with the SP,
// exp, the rate of p's level and the Secret Book of Giants for the first
// level of a route.
func (s *Server) handleEnchantSkillInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestExEnchantSkill(body)
	if err != nil {
		s.log.Debug("game: bad RequestExEnchantSkillInfo", append(p.logArgs(), "err", err)...)
		return
	}
	if req.SkillID <= 0 || req.Level <= 0 || p.Level() < subclass.EnchantMinLevel {
		return
	}
	n := s.enchantTrainer(p)
	if n == nil {
		return
	}
	e := s.offeredEnchant(p, n, int(req.SkillID), int(req.Level))
	if e == nil {
		return
	}
	info := serverpackets.ExEnchantSkillInfo{
		ID: req.SkillID, Level: req.Level,
		SPCost: int32(e.SP), ExpCost: int64(e.Exp), Rate: int32(e.Rate(p.Level())), //nolint:gosec // data values fit
	}
	if s.subclass.cfg.EnchantBookNeeded && subclass.EnchantBookLevel(int(req.Level)) {
		info.Requirements = append(info.Requirements, serverpackets.EnchantSkillRequirement{
			Type: enchantBookRequirement, ItemID: subclass.SecretBookOfGiants, Count: 1,
		})
	}
	p.Send(serverpackets.NewExEnchantSkillInfoPacket(info))
}

// handleEnchantSkill handles RequestExEnchantSkill (D0 07, Mobius
// RequestExEnchantSkill): the checks of subclass.CheckEnchant; a level the
// trainer does not offer is an illegal action (a kick). With the costs met
// the book is used up when needed, then Rnd.get(100) < rate decides: a
// success gives the level (saved), takes the exp and SP with their
// messages and says so; a failure puts the skill back to its base level
// and keeps the exp and SP. Then the enchant list again, UserInfo,
// SkillList and the skill's shortcuts at the level it has now.
func (s *Server) handleEnchantSkill(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestExEnchantSkill(body)
	if err != nil {
		s.log.Debug("game: bad RequestExEnchantSkill", append(p.logArgs(), "err", err)...)
		return
	}
	n := s.enchantTrainer(p)
	if n == nil {
		return
	}
	id, level := int(req.SkillID), int(req.Level)
	r := subclass.EnchantRequest{
		SkillID: id, Level: level,
		ClassID: p.ClassID(), PlayerLevel: p.Level(), KnownLevel: p.SkillLevel(id),
		SP: int64(p.SP()), Exp: p.Exp(), LevelExp: s.ExpForLevel(p.Level()),
		HasBook: p.inv.Inventory != nil && p.inv.ByItemID(subclass.SecretBookOfGiants) != nil,
	}
	if r.KnownLevel < level && r.ClassID >= subclass.EnchantMinClassID && r.PlayerLevel >= subclass.EnchantMinLevel {
		r.Offered = s.offeredEnchant(p, n, id, level)
	}
	res, needBook := subclass.CheckEnchant(r, s.subclass.cfg)
	switch res {
	case subclass.EnchantIgnored:
		return
	case subclass.EnchantIllegal:
		p.SendMessage("You are trying to learn skill that you can't...")
		s.illegalAction(p, "enchant skill "+strconv.Itoa(id)+" to "+strconv.Itoa(level)+" without the trainer offering it")
		return
	case subclass.EnchantNoSP:
		p.SendSystemMessage(sysMsgEnchantNoSP)
		return
	case subclass.EnchantNoExp:
		p.SendSystemMessage(sysMsgEnchantNoExp)
		return
	case subclass.EnchantNoBook:
		p.SendSystemMessage(sysMsgEnchantNoItems)
		return
	}
	if needBook && !s.DestroyItemByItemID(p, subclass.SecretBookOfGiants, 1, true) {
		p.SendSystemMessage(sysMsgEnchantNoItems)
		return
	}
	e := r.Offered
	final := level
	if subclass.EnchantSucceeds(s.subclass.rand, e.Rate(p.Level())) {
		sk, _ := s.Skill(id, level)
		s.AddSkill(p, sk, true)
		s.RemoveExpAndSP(p, int64(e.Exp), int64(e.SP), true)
		p.SendSystemMessage(sysMsgEnchantSucceeded, serverpackets.SkillNameParam(int32(id), 1)) //nolint:gosec // skill IDs are small
		s.log.Info("skill enchanted", append(p.logArgs(), "skill", id, "level", level, "rate", e.Rate(p.Level()), "book", needBook)...)
	} else {
		if level > 100 {
			final = e.BaseLevel
			if sk, ok := s.Skill(id, final); ok {
				s.AddSkill(p, sk, true)
			}
			p.SendSkillList()
		}
		p.SendSystemMessage(sysMsgEnchantFailed)
		s.log.Info("skill enchant failed", append(p.logArgs(), "skill", id, "level", level, "rate", e.Rate(p.Level()), "book", needBook)...)
	}
	s.showEnchantSkillList(p, n)
	p.UpdateUserInfo()
	p.SendSkillList()
	s.UpdateSkillShortcuts(p, id, final)
}
