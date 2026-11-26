package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
)

// Clan skills (Plan 12 Phase 7 item 6): the leader learns them at a village
// master ("learn_clan_skills", Mobius VillageMaster.showPledgeSkillList, and
// the PLEDGE branches of RequestAcquireSkillInfo and RequestAcquireSkill)
// for clan reputation (the tree's levelUpSp) and, with LifeCrystalNeeded,
// items. Clan.addNewSkill stores the skill and gives it to the members
// online; addSkillEffects and removeSkillEffects give and take them on
// login, joining and leaving (by the social class the tree names), and
// skillsStatus disables them while the reputation is below 0.
//
// As in Mobius, a disabled clan skill is shown disabled in SkillList and
// cannot be used; Mobius keeps a disabled passive's stats, and so does
// L2Go. Only whole-clan skills (unit -2) are learned in Interlude; skills
// stored for a unit are given to its members as Mobius does.

func init() {
	registerFeature("clan skills", setupClanSkills)
	registerVillageMasterCommand("learn_clan_skills", func(s *Server, p *Player, _ *Npc, _ []string) {
		s.showPledgeSkillList(p)
	})
	clanSkillListHook = (*Server).clanSkillListPacket
	clanSkillEffectHook = func(s *Server, p *Player, add bool) {
		if add {
			s.giveClanSkills(p)
		} else {
			s.takeClanSkills(p)
		}
	}
	clanSkillsHook = (*Server).setClanSkillsDisabled
}

// System messages of the clan skills (Mobius SystemMessageId).
const (
	clanSkillMsgItemDisappeared serverpackets.SystemMessageID = 301  // "$s2 $s1 has disappeared."
	clanSkillMsgAdded           serverpackets.SystemMessageID = 1788 // "The clan skill $s1 has been added."
	clanSkillMsgNotEnoughRepute serverpackets.SystemMessageID = 1852 // "The attempt to acquire the skill has failed because of an insufficient Clan Reputation Score."
)

const (
	// clanSkillMaxClanLevel is the last clan level.
	clanSkillMaxClanLevel = 8
	// clanSkillFirstLevel is the clan level of the first clan skills.
	clanSkillFirstLevel = 5
)

// clanSkillsFeature is the clan skill tree on the Server (Server.clanSkills).
// Set up once, then read-only.
type clanSkillsFeature struct {
	// tree is the pledge tree (clan and residence skills) in the order
	// Mobius iterates its ConcurrentHashMap of skill hash codes.
	tree []data.SkillLearn
	// inTree holds the skill levels of the tree (Mobius isClanSkill).
	inTree map[data.SkillRef]bool
}

// setupClanSkills orders the pledge tree as Mobius iterates it.
func setupClanSkills(s *Server) error {
	f := &s.clanSkills
	f.inTree = make(map[data.SkillRef]bool)
	d := s.skills.data
	if d == nil {
		return nil
	}
	file := d.OtherSkillTree(data.TreePledge)
	var order chmOrder
	byHash := make(map[int]int, len(file))
	for i := range file {
		h := file[i].SkillID*1021 + file[i].Level // SkillData.getSkillHashCode
		byHash[h] = i
		order.put(h)
		f.inTree[file[i].Ref()] = true
	}
	for _, h := range order.keys() {
		f.tree = append(f.tree, file[byHash[h]])
	}
	return nil
}

// pledgeSkillLearn is Mobius getPledgeSkill: the pledge tree's entry of a
// skill level, nil if none.
func (s *Server) pledgeSkillLearn(id, level int) *data.SkillLearn {
	return findSkillLearn(s.clanSkills.tree, id, level)
}

// isClanSkill is Mobius Skill.isClanSkill: the pledge tree teaches the
// skill level.
func (s *Server) isClanSkill(sk *skills.Skill) bool {
	return sk != nil && s.clanSkills.inTree[data.SkillRef{ID: sk.ID, Level: sk.Level}]
}

// clanSkillDisabled reports whether SkillList shows sk disabled because it
// is a clan skill of a clan with a reputation below 0 (Mobius
// sendSkillList).
func (p *Player) clanSkillDisabled(sk *skills.Skill) bool {
	c := p.clan.clan
	return c != nil && c.Reputation < 0 && p.srv != nil && p.srv.isClanSkill(sk)
}

// clanSkillLevel is the level of the whole-clan skill id c has, 0 for none.
func clanSkillLevel(c *Clan, id int) int {
	for _, sk := range c.Skills {
		if sk.Unit == storage.ClanSkillAllUnit && int(sk.SkillID) == id {
			return int(sk.Level)
		}
	}
	return 0
}

// clanSkillsOfUnit are c's skills of a unit (storage.ClanSkillAllUnit for
// the whole clan), in their order.
func clanSkillsOfUnit(c *Clan, unit int32) []storage.ClanSkill {
	var out []storage.ClanSkill
	for _, sk := range c.Skills {
		if sk.Unit == unit {
			out = append(out, sk)
		}
	}
	return out
}

// clanSkill returns the skill level of a stored clan skill, false when the
// skill data does not have it.
func (s *Server) clanSkill(cs storage.ClanSkill) (*skills.Skill, bool) {
	return s.Skill(int(cs.SkillID), int(cs.Level))
}

// availableClanSkills is Mobius getAvailablePledgeSkills: the entries of the
// pledge tree but the residence skills, up to c's level, that are level 1 of
// a skill c does not have or the level after the one it has.
func (s *Server) availableClanSkills(c *Clan) []*data.SkillLearn {
	var out []*data.SkillLearn
	for i := range s.clanSkills.tree {
		l := &s.clanSkills.tree[i]
		if l.ResidenceSkill || c.Level < l.GetLevel {
			continue
		}
		if old := clanSkillLevel(c, l.SkillID); old > 0 {
			if old+1 == l.Level {
				out = append(out, l)
			}
		} else if l.Level == 1 {
			out = append(out, l)
		}
	}
	return out
}

// clanSkillListPacket is PledgeSkillList of c (Mobius PledgeSkillList: the
// whole-clan skills, in the order of its map of skill IDs), with each
// skill's display ID and level.
func (s *Server) clanSkillListPacket(c *Clan) []byte {
	all := clanSkillsOfUnit(c, storage.ClanSkillAllUnit)
	var order chmOrder
	byID := make(map[int]storage.ClanSkill, len(all))
	for _, cs := range all {
		if _, dup := byID[int(cs.SkillID)]; !dup {
			order.put(int(cs.SkillID))
		}
		byID[int(cs.SkillID)] = cs
	}
	var list []serverpackets.PledgeSkill
	for _, id := range order.keys() {
		cs := byID[id]
		ps := serverpackets.PledgeSkill{ID: cs.SkillID, Level: cs.Level}
		if sk, ok := s.clanSkill(cs); ok {
			ps = serverpackets.PledgeSkill{ID: int32(sk.DisplayID), Level: int32(sk.DisplayLevel)} //nolint:gosec // skill IDs and levels are small
		}
		list = append(list, ps)
	}
	return serverpackets.NewPledgeSkillListPacket(list)
}

// ----- the village master -----

// showPledgeSkillList is Mobius VillageMaster.showPledgeSkillList: a clan
// leader gets AcquireSkillList of the clan skills it can learn, with their
// reputation cost and social class; without any, "Come back when you have
// reached Level S1." below clan level 8 (5 below level 5, else the next
// level), the NoMoreSkills page at level 8. Others get the NotClanLeader
// page. ActionFailed ends it.
func (s *Server) showPledgeSkillList(p *Player) {
	if !p.IsClanLeader() {
		s.sendPage(p, s.pageFile(p, 0, "villagemaster/NotClanLeader.htm"))
		p.SendActionFailed()
		return
	}
	c := p.clan.clan
	var entries []serverpackets.AcquireSkillListEntry
	for _, l := range s.availableClanSkills(c) {
		req := int32(0)
		if l.SocialClass != nil {
			req = int32(*l.SocialClass)
		}
		entries = append(entries, serverpackets.AcquireSkillListEntry{
			ID:           int32(l.SkillID), //nolint:gosec // skill IDs are small
			NextLevel:    int32(l.Level),   //nolint:gosec // skill levels are small
			MaxLevel:     int32(l.Level),   //nolint:gosec // skill levels are small
			SPCost:       int32(l.SP),      //nolint:gosec // reputation costs fit in int32
			Requirements: req,
		})
	}
	switch {
	case len(entries) > 0:
		p.Send(serverpackets.NewAcquireSkillListPacket(serverpackets.AcquireSkillList{
			Type:   serverpackets.AcquireSkillPledge,
			Skills: entries,
		}))
	case c.Level < clanSkillMaxClanLevel:
		level := clanSkillFirstLevel
		if c.Level >= clanSkillFirstLevel {
			level = c.Level + 1
		}
		p.SendSystemMessage(sysMsgLearnComeBackAtS1, serverpackets.NumberParam(int32(level))) //nolint:gosec // clan levels are small
	default:
		s.sendPage(p, s.pageFile(p, 0, "villagemaster/NoMoreSkills.htm"))
	}
	p.SendActionFailed()
}

// clanSkillInfo is the PLEDGE branch of RequestAcquireSkillInfo: for the
// clan leader, AcquireSkillInfo with the reputation cost and, with
// LifeCrystalNeeded, the items.
func (s *Server) clanSkillInfo(p *Player, id, level int) {
	l := s.pledgeSkillLearn(id, level)
	if l == nil || !p.IsClanLeader() {
		return
	}
	info := serverpackets.AcquireSkillInfo{
		SkillID: int32(id),    //nolint:gosec // skill IDs are small
		Level:   int32(level), //nolint:gosec // skill levels are small
		SPCost:  int32(l.SP),  //nolint:gosec // reputation costs fit in int32
		Type:    serverpackets.AcquireSkillPledge,
	}
	if s.gs.Clan.LifeCrystalNeeded {
		for _, it := range l.Items {
			info.Items = append(info.Items, serverpackets.AcquireSkillItem{ItemID: int32(it.ID), Count: it.Count}) //nolint:gosec // item IDs are small
		}
	}
	p.Send(serverpackets.NewAcquireSkillInfoPacket(info))
}

// learnClanSkill is the PLEDGE branch of RequestAcquireSkill: the clan
// leader pays the reputation and, with LifeCrystalNeeded, the items (each
// "... has disappeared."; a missing one stops it with "You do not have the
// necessary materials ..."), the clan learns sk (addNewClanSkill), the
// members online get PledgeSkillList, and the leader the list again. Too
// little reputation gives "The attempt to acquire the skill has failed ...".
//
// Mobius takes any level of the tree here; L2Go takes only one the clan
// could pick from the list (availableClanSkills), so a forged packet cannot
// skip levels or the clan level.
func (s *Server) learnClanSkill(p *Player, sk *skills.Skill) {
	l := s.pledgeSkillLearn(sk.ID, sk.Level)
	if l == nil || !p.IsClanLeader() {
		return
	}
	c := p.clan.clan
	learnable := false
	for _, a := range s.availableClanSkills(c) {
		if a == l {
			learnable = true
		}
	}
	if !learnable {
		s.log.Warn("game: learning a clan skill the clan cannot learn", append(p.logArgs(), "clan", c.Name, "skill", sk.ID, "level", sk.Level)...)
		return
	}
	cost := l.SP
	if c.Reputation < cost {
		p.SendSystemMessage(clanSkillMsgNotEnoughRepute)
		s.showPledgeSkillList(p)
		return
	}
	if s.gs.Clan.LifeCrystalNeeded {
		for _, it := range l.Items {
			if !s.DestroyItemByItemID(p, int32(it.ID), it.Count, false) { //nolint:gosec // item IDs are small
				p.SendSystemMessage(sysMsgLearnNoMaterials)
				s.showPledgeSkillList(p)
				return
			}
			p.SendSystemMessage(clanSkillMsgItemDisappeared,
				serverpackets.ItemNameParam(int32(it.ID)), serverpackets.NumberParam(int32(it.Count))) //nolint:gosec // small
		}
	}
	s.setClanReputation(c, -cost)
	p.SendSystemMessage(clanMsgReputationDeducted, serverpackets.NumberParam(int32(cost))) //nolint:gosec // reputation costs fit in int32
	s.addNewClanSkill(c, sk)
	broadcastToClan(c, s.clanSkillListPacket(c))
	s.SaveJoint([]*Player{p}, storage.Batch{Clans: []storage.Clan{c.document()}})
	s.log.Info("clan skill learned", append(p.logArgs(), "clan", c.Name, "skill", sk.ID, "level", sk.Level, "reputation", cost)...)
	s.showPledgeSkillList(p)
}

// addNewClanSkill is Mobius Clan.addNewSkill(skill) for the whole clan: sk
// replaces the level c had, and every member online gets it (not saved with
// the character), PledgeSkillListAdd, "The clan skill S1 has been added."
// and SkillList. The caller writes the clan. Loop-only.
func (s *Server) addNewClanSkill(c *Clan, sk *skills.Skill) {
	replaced := false
	for i := range c.Skills {
		if cs := &c.Skills[i]; cs.Unit == storage.ClanSkillAllUnit && int(cs.SkillID) == sk.ID {
			cs.Level = int32(sk.Level) //nolint:gosec // skill levels are small
			replaced = true
			break
		}
	}
	if !replaced {
		c.Skills = append(c.Skills, storage.ClanSkill{SkillID: int32(sk.ID), Level: int32(sk.Level), Unit: storage.ClanSkillAllUnit}) //nolint:gosec // small
	}
	added := clanSysMsg(clanSkillMsgAdded, serverpackets.SkillNameParam(int32(sk.ID), 1)) //nolint:gosec // skill IDs are small
	for _, q := range c.OnlineMembers() {
		// Skill.getMinPledgeClass is 0 for every clan skill of Interlude.
		s.AddSkill(q, sk, false)
		q.Send(serverpackets.NewPledgeSkillListAddPacket(int32(sk.ID), int32(sk.Level))) //nolint:gosec // small
		q.Send(added)
		q.SendSkillList()
	}
}

// ----- members -----

// giveClanSkills is Mobius Clan.addSkillEffects(player): the whole-clan
// skills whose social class p's pledge class reaches (pledge class + 1 at
// least the class's ordinal), then the skills of p's unit when it exists;
// with a reputation below 0 they are disabled at once. (Mobius returns
// before that check when p's unit does not exist, so a main clan member is
// never disabled at login; L2Go disables every member.)
func (s *Server) giveClanSkills(p *Player) {
	c := p.clan.clan
	if c == nil {
		return
	}
	class := p.PledgeClass() + 1
	for _, cs := range clanSkillsOfUnit(c, storage.ClanSkillAllUnit) {
		sk, ok := s.clanSkill(cs)
		if !ok {
			continue
		}
		if l := s.pledgeSkillLearn(sk.ID, sk.Level); l == nil || l.SocialClass == nil || class >= int(*l.SocialClass) {
			s.AddSkill(p, sk, false)
		}
	}
	if unit := p.PledgeType(); c.Unit(unit) != nil {
		for _, cs := range clanSkillsOfUnit(c, int32(unit)) { //nolint:gosec // unit types are small
			if sk, ok := s.clanSkill(cs); ok {
				s.AddSkill(p, sk, false)
			}
		}
	}
	if c.Reputation < 0 {
		s.setClanSkillsDisabled(p, true)
	}
}

// takeClanSkills is Mobius Clan.removeSkillEffects(player): the whole-clan
// skills, then those of the main clan (unit 0) or of p's unit.
func (s *Server) takeClanSkills(p *Player) {
	c := p.clan.clan
	if c == nil {
		return
	}
	for _, cs := range s.unitClanSkills(c, p) {
		s.RemoveSkill(p, int(cs.SkillID), false, true)
	}
}

// setClanSkillsDisabled is Mobius Clan.skillsStatus(player, disable): the
// clan skills of p (as takeClanSkills finds them) are disabled for good or
// enabled again.
func (s *Server) setClanSkillsDisabled(p *Player, disable bool) {
	c := p.clan.clan
	if c == nil {
		return
	}
	for _, cs := range s.unitClanSkills(c, p) {
		sk, ok := s.clanSkill(cs)
		if !ok {
			continue
		}
		if disable {
			s.DisableSkill(p, sk, 0)
		} else {
			s.EnableSkill(p, sk)
		}
	}
}

// unitClanSkills are the clan skills removeSkillEffects and skillsStatus
// touch for p: the whole-clan skills, and those of the main clan for a
// member of it or of p's unit when it exists.
func (s *Server) unitClanSkills(c *Clan, p *Player) []storage.ClanSkill {
	list := clanSkillsOfUnit(c, storage.ClanSkillAllUnit)
	unit := p.PledgeType()
	if unit == 0 || c.Unit(unit) != nil {
		list = append(list, clanSkillsOfUnit(c, int32(unit))...) //nolint:gosec // unit types are small
	}
	return list
}
