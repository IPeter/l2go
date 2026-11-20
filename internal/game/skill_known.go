package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// Known skills (Mobius Creature._skills) with Mobius addSkill and
// removeSkill, the skills of NPC templates, and the auto-get skills of
// players (Player.rewardSkills, K12).

// chmOrder keeps int keys in the order a Java ConcurrentHashMap<Integer, V>
// iterates them: by bin of the spread hash, and within a bin in the order
// the map's puts, removes and resizes left them. Mobius sends SkillList and
// SkillCoolTime in that order. The zero value is empty.
type chmOrder struct {
	bins  [][]int
	count int
}

// chmSpread is ConcurrentHashMap.spread of an Integer key.
func chmSpread(k int) int {
	h := uint32(int32(k)) //nolint:gosec // Java int bits
	return int((h ^ (h >> 16)) & 0x7fffffff)
}

// put adds a key that is not in the map, at the tail of its bin, then grows
// the table at 0.75 of its size as addCount does.
func (o *chmOrder) put(k int) {
	if o.bins == nil {
		o.bins = make([][]int, 16)
	}
	i := chmSpread(k) & (len(o.bins) - 1)
	o.bins[i] = append(o.bins[i], k)
	o.count++
	for n := len(o.bins); o.count >= n-n/4; n = len(o.bins) {
		o.grow()
	}
}

// grow doubles the table as ConcurrentHashMap.transfer does: each bin splits
// into a low and a high bin; the nodes before the bin's last run are
// prepended (so reversed), the last run keeps its order.
func (o *chmOrder) grow() {
	n := len(o.bins)
	next := make([][]int, 2*n)
	for i, bin := range o.bins {
		if len(bin) == 0 {
			continue
		}
		runBit := chmSpread(bin[0]) & n
		last := 0
		for j := 1; j < len(bin); j++ {
			if b := chmSpread(bin[j]) & n; b != runBit {
				runBit, last = b, j
			}
		}
		var lo, hi []int
		if runBit == 0 {
			lo = slices.Clone(bin[last:])
		} else {
			hi = slices.Clone(bin[last:])
		}
		for j := 0; j < last; j++ {
			if chmSpread(bin[j])&n == 0 {
				lo = slices.Insert(lo, 0, bin[j])
			} else {
				hi = slices.Insert(hi, 0, bin[j])
			}
		}
		next[i], next[i+n] = lo, hi
	}
	o.bins = next
}

// remove takes a key out of its bin.
func (o *chmOrder) remove(k int) {
	if o.bins == nil {
		return
	}
	i := chmSpread(k) & (len(o.bins) - 1)
	if j := slices.Index(o.bins[i], k); j >= 0 {
		o.bins[i] = slices.Delete(o.bins[i], j, j+1)
		o.count--
	}
}

// keys returns the keys in iteration order.
func (o *chmOrder) keys() []int {
	out := make([]int, 0, o.count)
	for _, bin := range o.bins {
		out = append(out, bin...)
	}
	return out
}

// knownSkills are a creature's skills by ID, in Mobius's iteration order.
// The zero value is empty and allocates nothing.
type knownSkills struct {
	byID  map[int]*skills.Skill
	order chmOrder
}

// get returns the known level of a skill, nil if unknown.
func (k *knownSkills) get(id int) *skills.Skill { return k.byID[id] }

// level returns the known level of a skill, 0 if unknown.
func (k *knownSkills) level(id int) int {
	if sk := k.byID[id]; sk != nil {
		return sk.Level
	}
	return 0
}

// put stores sk and returns the level it replaced.
func (k *knownSkills) put(sk *skills.Skill) *skills.Skill {
	if k.byID == nil {
		k.byID = make(map[int]*skills.Skill)
	}
	old := k.byID[sk.ID]
	k.byID[sk.ID] = sk
	if old == nil {
		k.order.put(sk.ID)
	}
	return old
}

// remove forgets a skill and returns it, nil if unknown.
func (k *knownSkills) remove(id int) *skills.Skill {
	old := k.byID[id]
	if old != nil {
		delete(k.byID, id)
		k.order.remove(id)
	}
	return old
}

// all returns the known skills in iteration order.
func (k *knownSkills) all() []*skills.Skill {
	ids := k.order.keys()
	out := make([]*skills.Skill, len(ids))
	for i, id := range ids {
		out[i] = k.byID[id]
	}
	return out
}

// addSkill is Mobius Creature.addSkill: sk replaces the level the creature
// knew (whose stats and, for a passive, effects end), and a passive starts
// its effects. It returns the replaced level, nil for a new skill.
func (b *skillBase) addSkill(sk *skills.Skill) *skills.Skill {
	if sk == nil {
		return nil
	}
	old := b.known.put(sk)
	if old != nil && old.IsPassive() {
		b.el.StopSkill(skills.FinishNormal, old.ID)
	}
	if sk.IsPassive() {
		sk.AddPassive(b.srv.skills.env, b.self)
	}
	return old
}

// removeSkill is Mobius Creature.removeSkill(id, cancelEffect): the skill is
// forgotten, a cast of it stops (Phase 7.3: casting checks the known level),
// and with cancelEffect, or for a toggle or passive, its effects end.
func (b *skillBase) removeSkill(id int, cancelEffect bool) *skills.Skill {
	old := b.known.remove(id)
	if old != nil && (cancelEffect || old.IsToggle() || old.IsPassive()) {
		b.el.StopSkill(skills.FinishRemoved, old.ID)
	}
	return old
}

// KnownSkill returns the level of a skill the player knows, nil if it does
// not know it (Mobius getKnownSkill). Loop-only.
func (p *Player) KnownSkill(id int) *skills.Skill { return p.skill.known.get(id) }

// SkillLevel returns the level of a known skill, 0 if unknown (Mobius
// getSkillLevel). Loop-only.
func (p *Player) SkillLevel(id int) int { return p.skill.known.level(id) }

// Skills returns the player's known skills in Mobius's iteration order (the
// order of SkillList). Loop-only.
func (p *Player) Skills() []*skills.Skill { return p.skill.known.all() }

// AddSkill gives p a skill level, Mobius Player.addSkill(skill, store): it
// replaces the level p knew; a passive's modifiers and effects follow at
// once (UserInfo at the end of the tick when the numbers changed). With
// store the skill is saved with the character (character_skills, K13; the
// save is write-behind). It returns the replaced level, nil for a new
// skill. The caller sends SkillList (SendSkillList), as Mobius callers do.
// Loop-only.
func (s *Server) AddSkill(p *Player, sk *skills.Skill, store bool) *skills.Skill {
	old := p.skill.addSkill(sk)
	if sk != nil && store {
		p.skill.storeSkill(sk.ID, sk.Level)
		s.RequestSave(p)
	}
	return old
}

// RemoveSkill takes a skill from p, Mobius Player.removeSkill(skill, store,
// cancelEffect): its toggle or passive effects end, its other effects too
// with cancelEffect. With store the skill also leaves the saved skills and
// the shortcut bar (shortcut.go). It returns the removed level, nil if p
// did not know it. The caller sends SkillList. Loop-only.
func (s *Server) RemoveSkill(p *Player, id int, store, cancelEffect bool) *skills.Skill {
	old := p.skill.removeSkill(id, cancelEffect || store)
	if store && p.skill.unstoreSkill(id) {
		s.RequestSave(p)
	}
	if store {
		s.deleteSkillShortcuts(p, id)
	}
	return old
}

// AddNpcSkill gives n a skill level (Creature.addSkill): passives change its
// stats at once. It returns the replaced level. Loop-only.
func (s *Server) AddNpcSkill(n *Npc, sk *skills.Skill) *skills.Skill { return n.skill.addSkill(sk) }

// RemoveNpcSkill takes a skill from n (Creature.removeSkill(id, true)).
// Loop-only.
func (s *Server) RemoveNpcSkill(n *Npc, id int) *skills.Skill { return n.skill.removeSkill(id, true) }

// NpcSkills returns the skills n knows: its template's (Mobius copies them
// into every NPC) and those added since. Loop-only.
func (n *Npc) Skills() []*skills.Skill { return n.skill.known.all() }

// addTemplateSkills gives a new NPC its template's skills, as the Mobius
// Creature constructor does for NPCs; the passives (race traits 4416, the
// stat multipliers 4408-4415, ...) change its stats before it spawns. A
// skill level above the highest takes the highest (SkillData.getSkill).
func (s *Server) addTemplateSkills(n *Npc) {
	c := s.skills.catalog
	if c == nil {
		return
	}
	for _, ref := range n.Template.Skills {
		if sk, ok := c.SkillOrMax(ref.ID, ref.Level); ok {
			n.skill.addSkill(sk)
		}
	}
}

// RewardSkills is Mobius Player.rewardSkills with AutoLearnSkills off: the
// auto-get skills of p's class tree up to its level, saved; for a non-GM
// the skills too far above its level drop (checkPlayerSkills,
// skill_learn.go); then SkillList. Login and every level change run it.
// Loop-only.
func (s *Server) RewardSkills(p *Player) {
	changed := s.giveAutoGetSkills(p)
	if !p.IsGM() && s.checkPlayerSkills(p) {
		changed = true
	}
	if changed {
		s.RequestSave(p)
	}
	p.SendSkillList()
}

// giveAutoGetSkills is Mobius giveAvailableAutoGetSkills: every auto-get
// entry of the complete class tree for p's race and level that is above the
// level p knows, collected first and then added (and stored) in tree order.
// It reports whether a skill was added.
func (s *Server) giveAutoGetSkills(p *Player) bool {
	d, c := s.skills.data, s.skills.catalog
	if d == nil || c == nil {
		return false
	}
	tree, ok := d.SkillTree(p.ClassID())
	if !ok {
		return false
	}
	race := data.Race(p.char.Race) //nolint:gosec // races are small
	var learn []*skills.Skill
	for i := range tree {
		l := &tree[i]
		if !l.AutoGet || p.Level() < l.GetLevel || (len(l.Races) > 0 && !slices.Contains(l.Races, race)) {
			continue
		}
		if old := p.skill.known.get(l.SkillID); old != nil && old.Level >= l.Level {
			continue
		}
		sk, ok := c.Skill(l.SkillID, l.Level)
		if !ok {
			s.log.Warn("game: skipping a missing auto-get skill", append(p.logArgs(), "skill", l.SkillID, "level", l.Level)...)
			continue
		}
		learn = append(learn, sk)
	}
	for _, sk := range learn {
		p.skill.addSkill(sk)
		p.skill.storeSkill(sk.ID, sk.Level)
	}
	return len(learn) > 0
}
