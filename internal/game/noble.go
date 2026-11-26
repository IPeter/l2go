package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// Noblesse (M11e, Mobius Player.setNoble): the noble flag lives in the
// character (storage.Character.Noble, Mobius characters.nobless) and the
// skills of nobleSkillTree.xml come with it. As in Mobius they are never
// stored: the flag gives them again at character selection
// (loadSubclass), after a class switch (regiveTemporarySkills) and whenever
// it is set. The Olympiad (M11h) reads IsNoble.

// IsNoble reports whether the player is a noble (scripting.PlayerNoble).
func (p *Player) IsNoble() bool { return p.char.Noble }

// nobleFlag is IsNoble for the info packets.
func (p *Player) nobleFlag() bool { return p.char.Noble }

// SetNoble is Mobius setNoble: the noble skills are given or taken (not
// stored; a removed one's effects end), the flag is set and saved soon
// after, and the skill list is sent. Quests (Possessor of a Precious Soul
// 4), //setnoble and the admin panel's class page call it. Loop-only.
func (p *Player) SetNoble(noble bool) {
	s := p.srv
	for _, sk := range s.nobleSkills() {
		if noble {
			s.AddSkill(p, sk, false)
		} else {
			s.RemoveSkill(p, sk.ID, false, true)
		}
	}
	p.char.Noble = noble
	s.RequestSave(p)
	p.SendSkillList()
}

// nobleSkills returns the skills of nobleSkillTree.xml (Mobius
// getNobleSkillTree), in the data's order; a level the skill data does not
// have is left out.
func (s *Server) nobleSkills() []*skills.Skill {
	d, c := s.skills.data, s.skills.catalog
	if d == nil || c == nil {
		return nil
	}
	var out []*skills.Skill
	for _, l := range d.OtherSkillTree(data.TreeNoble) {
		if sk, ok := c.Skill(l.SkillID, l.Level); ok {
			out = append(out, sk)
		}
	}
	return out
}

// nobleSkillIDs returns the skill IDs of the noble tree, for the migration
// of stored characters (subclass.Migrate). Safe on any goroutine.
func (s *Server) nobleSkillIDs() []int {
	d := s.skills.data
	if d == nil {
		return nil
	}
	var ids []int
	for _, l := range d.OtherSkillTree(data.TreeNoble) {
		ids = append(ids, l.SkillID)
	}
	return ids
}

// giveNobleSkills gives a noble its skills without packets (Mobius
// setNoble(true) at load and regiveTemporarySkills).
func (s *Server) giveNobleSkills(p *Player) {
	for _, sk := range s.nobleSkills() {
		s.AddSkill(p, sk, false)
	}
}
