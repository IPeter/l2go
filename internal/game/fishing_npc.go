package game

import (
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// The fishermen (Mobius entity/actor/instance/Fisherman, a Merchant): their
// fishing skills (FishSkillList, RequestAcquireSkillInfo and
// RequestAcquireSkill of type FISHING) and the championship pages
// (FishingChampionship, FishingReward). Their shops are the merchant's.

// fishermanCommand runs the Fisherman's own bypasses (Mobius
// Fisherman.onBypassFeedback); false hands the command on to the merchant's.
func (s *Server) fishermanCommand(p *Player, n *Npc, command string) bool {
	on := s.fishing.champ != nil && s.fishing.champ.Config().Enabled
	switch {
	case strings.EqualFold(command, "FishSkillList"):
		s.showFishSkillList(p)
	case strings.HasPrefix(command, "FishingChampionship"):
		if on {
			s.showChampScreen(p, n) // fishing_champ.go
		} else {
			s.sendPage(p, s.pageFile(p, n.ID(), fishPageNoChampion))
		}
	case strings.HasPrefix(command, "FishingReward"):
		if on {
			s.claimFishingPrize(p, n)
		} else {
			s.sendPage(p, s.pageFile(p, n.ID(), fishPageNoChampion))
		}
	default:
		return false
	}
	return true
}

// availableFishingSkills is Mobius getAvailableFishingSkills: the entries of
// the fishing tree for p's race (or any race), learned from an NPC, up to
// p's level, that are level 1 of an unknown skill or the level after the
// known one.
func (s *Server) availableFishingSkills(p *Player) []*data.SkillLearn {
	var out []*data.SkillLearn
	race := data.Race(p.Race()) //nolint:gosec // races are small
	for i := range s.fishing.skillTree {
		l := &s.fishing.skillTree[i]
		if len(l.Races) > 0 && !slices.Contains(l.Races, race) {
			continue
		}
		if !l.LearnedByNpc || p.Level() < l.GetLevel {
			continue
		}
		if old := p.SkillLevel(l.SkillID); old > 0 {
			if old == l.Level-1 {
				out = append(out, l)
			}
		} else if l.Level == 1 {
			out = append(out, l)
		}
	}
	return out
}

// showFishSkillList is Mobius Fisherman.showFishSkillList: AcquireSkillList
// of type FISHING with the skills p can learn now; with none, "Come back
// when you have reached Level S1." when a later level teaches more, else
// "There are no other skills to learn.".
func (s *Server) showFishSkillList(p *Player) {
	var entries []serverpackets.AcquireSkillListEntry
	for _, l := range s.availableFishingSkills(p) {
		if _, ok := s.Skill(l.SkillID, l.Level); !ok {
			continue
		}
		entries = append(entries, serverpackets.AcquireSkillListEntry{
			ID:           int32(l.SkillID), //nolint:gosec // skill IDs are small
			NextLevel:    int32(l.Level),   //nolint:gosec // skill levels are small
			MaxLevel:     int32(l.Level),   //nolint:gosec // skill levels are small
			SPCost:       int32(l.SP),      //nolint:gosec // SP costs fit in int32
			Requirements: 1,
		})
	}
	if len(entries) > 0 {
		p.Send(serverpackets.NewAcquireSkillListPacket(serverpackets.AcquireSkillList{
			Type:   serverpackets.AcquireSkillFishing,
			Skills: entries,
		}))
		return
	}
	minLevel := 0
	for i := range s.fishing.skillTree {
		l := &s.fishing.skillTree[i]
		if l.LearnedByNpc && p.Level() < l.GetLevel && (minLevel == 0 || minLevel > l.GetLevel) {
			minLevel = l.GetLevel
		}
	}
	if minLevel > 0 {
		p.SendSystemMessage(sysMsgLearnComeBackAtS1, serverpackets.NumberParam(int32(minLevel))) //nolint:gosec // levels are small
		return
	}
	p.SendSystemMessage(sysMsgLearnNoOtherSkill)
}

// fishingSkillLearn is Mobius getFishingSkill: the fishing tree's entry of
// a skill level, nil if none.
func (s *Server) fishingSkillLearn(id, level int) *data.SkillLearn {
	return findSkillLearn(s.fishing.skillTree, id, level)
}

// fishingSkillInfo is RequestAcquireSkillInfo for a fishing skill:
// AcquireSkillInfo with its SP and items.
func (s *Server) fishingSkillInfo(p *Player, id, level int) {
	l := s.fishingSkillLearn(id, level)
	if l == nil {
		return
	}
	info := serverpackets.AcquireSkillInfo{
		SkillID: int32(id),    //nolint:gosec // skill IDs are small
		Level:   int32(level), //nolint:gosec // skill levels are small
		SPCost:  int32(l.SP),  //nolint:gosec // SP costs fit in int32
		Type:    serverpackets.AcquireSkillFishing,
	}
	for _, it := range l.Items {
		info.Items = append(info.Items, serverpackets.AcquireSkillItem{ItemID: int32(it.ID), Count: it.Count}) //nolint:gosec // item IDs are small
	}
	p.Send(serverpackets.NewAcquireSkillInfoPacket(info))
}

// learnFishingSkill is RequestAcquireSkill for a fishing skill: the entry
// is checked and paid for as a class skill is (checkPlayerSkill), then
// given.
func (s *Server) learnFishingSkill(p *Player, n *Npc, id, level int) {
	l := s.fishingSkillLearn(id, level)
	if l == nil {
		return
	}
	sk, ok := s.Skill(id, level)
	if !ok {
		return
	}
	class := learningClass(p)
	if s.checkLearnSkill(p, n, l, class) {
		s.giveLearnedSkill(p, n, sk, class)
	}
}

// showLearnList is RequestAcquireSkill.showSkillList: a fisherman shows its
// fishing skills, any other trainer the class skills.
func (s *Server) showLearnList(p *Player, n *Npc, class data.PlayerClass) {
	if s.npcIs(n, "Fisherman") {
		s.showFishSkillList(p)
		return
	}
	s.showSkillList(p, n, class)
}
