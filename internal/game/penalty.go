package game

import (
	"github.com/IPeter/l2go/internal/data"
)

// The grade penalty (decision K16, Mobius refreshExpertisePenalty with
// ExpertisePenalty = True). The weight penalty (refreshOverloaded) is the
// item feature's, in items.go.

const (
	// skillExpertise is the Expertise skill whose level is the grade a
	// player may wear (Mobius getExpertiseLevel: getSkillLevel(239)).
	skillExpertise = 239
	// skillGradePenalty is CommonSkill.GRADE_PENALTY.
	skillGradePenalty = 4267
)

// refreshExpertisePenalty is Mobius Player.refreshExpertisePenalty: the
// highest grade above the Expertise level among the worn weapons and the
// other worn items (arrows aside; a weapon of a grade not above the weapon
// penalty so far counts as armor, a Mobius quirk), less the Expertise level,
// 0-4 each. When either changes, skill 4267 level 1 comes or goes (level 1
// whatever the gap, as in Mobius) and the player gets EtcStatusUpdate.
// Mobius sends no SkillList for it.
func (s *Server) refreshExpertisePenalty(p *Player) {
	if p.inv.Inventory == nil {
		return
	}
	expertise := p.SkillLevel(skillExpertise)
	weapon, armor := 0, 0
	for _, it := range p.inv.Items() {
		t := it.Template()
		if !it.Equipped() || (t.Etc != nil && t.Etc.Type == data.EtcArrow) {
			continue
		}
		grade := t.Grade.Level()
		if grade <= expertise {
			continue
		}
		if t.Weapon != nil && grade > weapon {
			weapon = grade
		} else if grade > armor {
			armor = grade
		}
	}
	weapon = min(max(weapon-expertise, 0), 4)
	armor = min(max(armor-expertise, 0), 4)
	st := &p.itemSkill
	if st.weaponPenalty == weapon && st.armorPenalty == armor {
		return
	}
	st.weaponPenalty, st.armorPenalty = weapon, armor
	if weapon > 0 || armor > 0 {
		if sk, ok := s.Skill(skillGradePenalty, 1); ok {
			s.AddSkill(p, sk, false)
		}
	} else {
		s.RemoveSkill(p, skillGradePenalty, false, true)
	}
	if p.InGame() {
		p.Send(etcStatusPacket(p))
	}
}

// RefreshExpertisePenalty checks p's grade penalty again, as Mobius does on
// a level change and after learning Expertise. Loop-only.
func (s *Server) RefreshExpertisePenalty(p *Player) { s.refreshExpertisePenalty(p) }

// GradePenalty reports whether p wears a weapon or armor above its
// Expertise (the EtcStatusUpdate icon). Loop-only.
func (p *Player) GradePenalty() bool {
	return p.itemSkill.weaponPenalty > 0 || p.itemSkill.armorPenalty > 0
}

// CharmOfCourage reports whether p carries the effect of a Charm of
// Courage. Loop-only.
func (p *Player) CharmOfCourage() bool { return p.itemSkill.charm }
