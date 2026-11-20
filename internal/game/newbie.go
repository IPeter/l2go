package game

import "github.com/IPeter/l2go/internal/data"

// The newbie buffer (Plan 10 Phase 7.7, decision K17): the "SupportMagic"
// link of the Newbie Guides (Mobius handlers.bypass.npc.SupportMagic). A
// newbie of level 8 to 25 who has not made the 3rd class transfer gets the
// beginner buffs of its level band; the guide targets the player and casts
// each one through the NPC cast path, so everyone around sees its
// MagicSkillUse. The skills have no hit time: each lands at once.

// The level limits of the support magic.
const (
	supportMagicLowestLevel  = 8
	supportMagicHighestLevel = 25
)

// The refusal pages of the support magic, relative to the HTML folder.
const (
	supportMagicNovicePage    = "default/SupportMagicNovice.htm"
	supportMagicHighLevelPage = "default/SupportMagicHighLevel.htm"
	supportMagicLowLevelPage  = "default/SupportMagicLowLevel.htm"
)

// supportMagicThirdClassMessage is Mobius's (custom) refusal of a 3rd class
// character.
const supportMagicThirdClassMessage = "Only adventurers who have not completed their 3rd class transfer may receive these buffs."

// skillLifeCubic is the beginners' Life Cubic, levels 16 to 19, which the
// player casts on itself in Mobius. It needs cubics (Milestone 11).
const skillLifeCubic = 4338

// supportMagicBuff is one buff of the newbie buffer: level 1 of the skill,
// for players of minLevel to maxLevel.
type supportMagicBuff struct {
	skill              int
	minLevel, maxLevel int
}

// The buffs in Mobius's cast order: Wind Walk and Shield for everyone, then
// those of a beginner mage (CategoryType.BEGINNER_MAGE) or a fighter.
var (
	supportMagicCommon = []supportMagicBuff{
		{4322, 8, 24},  // Wind Walk for Beginners
		{4323, 11, 23}, // Shield for Beginners
	}
	supportMagicMage = []supportMagicBuff{
		{4328, 12, 22}, // Blessed Soul for Beginners
		{4329, 13, 21}, // Acumen for Beginners
		{4330, 14, 20}, // Concentration for Beginners
		{4331, 15, 19}, // Empower for Beginners
	}
	supportMagicFighter = []supportMagicBuff{
		{4324, 12, 22}, // Blessed Body for Beginners
		{4325, 13, 21}, // Vampiric Rage for Beginners
		{4326, 14, 20}, // Regeneration for Beginners
		{4327, 15, 19}, // Haste for Beginners
	}
)

// supportMagicSkills returns the IDs of the buffs a player of the level gets,
// in cast order; mage is BEGINNER_MAGE.
func supportMagicSkills(level int, mage bool) []int {
	lists := [][]supportMagicBuff{supportMagicCommon, supportMagicFighter}
	if mage {
		lists[1] = supportMagicMage
	}
	var ids []int
	for _, list := range lists {
		for _, b := range list {
			if level >= b.minLevel && level <= b.maxLevel {
				ids = append(ids, b.skill)
			}
		}
	}
	return ids
}

// bypassSupportMagic handles "SupportMagic" at n (Mobius SupportMagic): a
// player who is not a newbie gets the novice page, one above level 25 or
// below level 8 the level pages, a 3rd class one a message; any other gets
// the buffs of its level band, cast by n. Loop-only.
func (s *Server) bypassSupportMagic(p *Player, n *Npc) {
	if !p.IsNewbie() {
		s.supportMagicPage(p, n, supportMagicNovicePage)
		return
	}
	level := p.Level()
	switch {
	case level > supportMagicHighestLevel:
		s.supportMagicPage(p, n, supportMagicHighLevelPage)
		return
	case level < supportMagicLowestLevel:
		s.supportMagicPage(p, n, supportMagicLowLevelPage)
		return
	case data.PlayerClass(p.ClassID()).Level() == 3: //nolint:gosec // class IDs are small
		p.SendMessage(supportMagicThirdClassMessage)
		return
	}
	for _, id := range supportMagicSkills(level, p.InCategory(int(data.CategoryBeginnerMage))) {
		sk, ok := s.Skill(id, 1)
		if !ok {
			s.log.Warn("game: newbie buffer skill missing", "skill", id)
			continue
		}
		s.NpcCast(n, p, sk)
	}
	if level >= 16 && level <= 19 {
		s.log.Debug("game: Life Cubic needs cubics (Milestone 11), skipped", append(p.logArgs(), "skill", skillLifeCubic)...)
	}
}

// supportMagicPage shows p a refusal page of n (Mobius
// Npc.showChatWindow(player, filename)): the page with %objectId% filled,
// then ActionFailed.
func (s *Server) supportMagicPage(p *Player, n *Npc, rel string) {
	s.sendPage(p, s.pageFile(p, n.ID(), rel).replaceObjectID(n.ID()))
	p.SendActionFailed()
}
