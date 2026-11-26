package game

import (
	"errors"
	"strconv"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
)

// Learning skills (Plan 10 Phase 7.5, decision K12), as Mobius does it with
// AltGameSkillLearn and AutoLearnSkills off:
//
//   - The trainers' "SkillList" bypass (Mobius SkillList and
//     Folk.showSkillList) lists the next level of each skill the player's
//     own class learns from an NPC (AcquireSkillList), or says when to come
//     back; an NPC that does not teach the class shows its no-teach page.
//   - RequestAcquireSkillInfo answers with the SP and spellbooks a level
//     takes; RequestAcquireSkill checks the level, SP, prerequisites and
//     items again, takes them and gives the skill. Clan skills (PLEDGE) are
//     in clan_skills.go; fishing skills (FISHING) wait for Milestone 11.
//   - A new character starts with the level-1 skills of its class tree
//     (NewCharacterSkills, Mobius CharacterCreate). Login and every level
//     change give the auto-get skills (RewardSkills); losing levels lowers
//     the skills learned more than 9 levels above the player
//     (checkPlayerSkills; Expertise without that margin).
//   - GM commands: //add_skill, //remove_skill, //give_all_skills, //buff,
//     //stopallbuffs and //removereuse (skill_learn_gm.go).
//
// The classes each trainer teaches are in trainer_classes.go.

func init() { registerFeature("skill learning", setupSkillLearning) }

// SkillLearnedHook runs when p learned sk at a trainer, after the skill was
// added and SkillList asked for, before the trainer's list shows again
// (where Mobius RequestAcquireSkill calls Player.updateShortcuts).
type SkillLearnedHook func(p *Player, sk *skills.Skill)

// learnFeature is the learning state on the Server (Server.learn).
type learnFeature struct {
	learned []SkillLearnedHook
}

// OnSkillLearned registers fn to run whenever a player learns a skill at a
// trainer. Phase 7.6 updates the shortcuts of the skill from here (Mobius
// Player.updateShortcuts). Call it from a feature's setup.
func (s *Server) OnSkillLearned(fn SkillLearnedHook) {
	if fn != nil {
		s.learn.learned = append(s.learn.learned, fn)
	}
}

// System messages of learning.
const (
	sysMsgLearnNoMaterials  serverpackets.SystemMessageID = 276 // "You do not have the necessary materials or prerequisites to learn this skill."
	sysMsgLearnEarnedS1     serverpackets.SystemMessageID = 277 // "You have earned $s1."
	sysMsgLearnNotEnoughSP  serverpackets.SystemMessageID = 278 // "You do not have enough SP to learn this skill."
	sysMsgLearnComeBackAtS1 serverpackets.SystemMessageID = 607 // "You do not have any further skills to learn. Come back when you have reached Level $s1."
	sysMsgLearnNoOtherSkill serverpackets.SystemMessageID = 750 // "There are no other skills to learn."
)

const (
	// learnDivineInspiration and learnExpertise are Mobius CommonSkill
	// DIVINE_INSPIRATION and EXPERTISE.
	learnDivineInspiration = 1405
	learnExpertise         = 239
	// autoLearnDivineInspiration and divineSPBookNeeded are Player.ini
	// AutoLearnDivineInspiration (False) and DivineInspirationSpBookNeeded
	// (True).
	autoLearnDivineInspiration = false
	divineSPBookNeeded         = true
	// deleveledSkillMargin is how many levels a player may be below a
	// skill's learn level before checkPlayerSkills lowers it.
	deleveledSkillMargin = 9
	// The skills that enlarge storage (1368-1372) make RequestAcquireSkill
	// send ExStorageMaxCount.
	firstStorageSkill = 1368
	lastStorageSkill  = 1372
)

// setupSkillLearning registers the learning packets and GM commands.
func setupSkillLearning(s *Server) error {
	return errors.Join(
		s.Register(clientpackets.OpRequestAcquireSkillInfo, s.handleRequestAcquireSkillInfo),
		s.Register(clientpackets.OpRequestAcquireSkill, s.handleRequestAcquireSkill),
		setupSkillLearningGM(s),
	)
}

// learnerLevels is how availableSkills sees the learner: its level, whether
// it is a GM, and the level it knows of a skill (0 for none).
type learnerLevels struct {
	level int
	gm    bool
	known func(id int) int
}

// learnerOf returns p as availableSkills sees it, with its known skills.
func learnerOf(p *Player) learnerLevels {
	return learnerLevels{level: p.Level(), gm: p.IsGM(), known: p.SkillLevel}
}

// availableSkills is Mobius SkillTreeData.getAvailableSkills: the entries of
// class's complete tree that the learner can take next, in the tree's
// order. An entry counts when the learner's level reaches it, it is learned
// from an NPC (or is auto-get with autoGet, or from a Forgotten Scroll with
// byFS), it takes no items unless withItems (Forgotten Scroll skills
// always count), and it is level 1 of an unknown skill or the level after
// the known one. Divine Inspiration is left out with autoGet for a non-GM
// (AutoLearnDivineInspiration is off). The race is not checked, as in
// Mobius.
func (s *Server) availableSkills(who learnerLevels, class data.PlayerClass, byFS, autoGet, withItems bool) []*data.SkillLearn {
	d := s.skills.data
	if d == nil {
		return nil
	}
	tree, ok := d.SkillTree(int(class))
	if !ok || len(tree) == 0 {
		s.log.Warn("game: no skill tree for the class", "class", int(class))
		return nil
	}
	var out []*data.SkillLearn
	for i := range tree {
		l := &tree[i]
		if l.SkillID == learnDivineInspiration && !autoLearnDivineInspiration && autoGet && !who.gm {
			continue
		}
		if ((!autoGet || !l.AutoGet) && !l.LearnedByNpc && (!byFS || !l.LearnedByFS)) || who.level < l.GetLevel {
			continue
		}
		if !withItems && len(l.Items) > 0 && !l.LearnedByFS {
			continue
		}
		if old := who.known(l.SkillID); old > 0 {
			if old == l.Level-1 {
				out = append(out, l)
			}
		} else if l.Level == 1 {
			out = append(out, l)
		}
	}
	return out
}

// minLevelForNewSkill is Mobius getMinLevelForNewSkill: the lowest level
// above level at which class's tree teaches a skill at an NPC, 0 for none.
func (s *Server) minLevelForNewSkill(level int, class data.PlayerClass) int {
	d := s.skills.data
	if d == nil {
		return 0
	}
	tree, _ := d.SkillTree(int(class))
	minLevel := 0
	for i := range tree {
		l := &tree[i]
		if l.LearnedByNpc && level < l.GetLevel && (minLevel == 0 || minLevel > l.GetLevel) {
			minLevel = l.GetLevel
		}
	}
	return minLevel
}

// classSkillLearn is Mobius getClassSkill: the entry of class's complete
// tree for a skill level, nil if the tree does not teach it.
func (s *Server) classSkillLearn(id, level int, class data.PlayerClass) *data.SkillLearn {
	d := s.skills.data
	if d == nil {
		return nil
	}
	tree, _ := d.SkillTree(int(class))
	return findSkillLearn(tree, id, level)
}

// findSkillLearn returns the entry of tree for a skill level, nil if none.
func findSkillLearn(tree []data.SkillLearn, id, level int) *data.SkillLearn {
	for i := range tree {
		if tree[i].SkillID == id && tree[i].Level == level {
			return &tree[i]
		}
	}
	return nil
}

// learningClass is the class whose skills p learns at trainers (Mobius
// getLearningClass): with AltGameSkillLearn off, p's own class.
func learningClass(p *Player) data.PlayerClass {
	return data.PlayerClass(p.ClassID()) //nolint:gosec // class IDs are small
}

// bypassSkillList handles the "SkillList" bypass of an NPC (Mobius
// SkillList with AltGameSkillLearn off): the skills of p's own class.
func (s *Server) bypassSkillList(p *Player, n *Npc) {
	s.showSkillList(p, n, learningClass(p))
}

// showSkillList is Mobius Folk.showSkillList: n's no-teach page when it does
// not teach class; otherwise AcquireSkillList with the next level of each
// skill p can learn from an NPC now, with its SP. With nothing to learn:
// "Come back when you have reached Level S1." when a later level teaches
// more, a plain message to a first-class player, else "There are no other
// skills to learn." and AcquireSkillDone.
func (s *Server) showSkillList(p *Player, n *Npc, class data.PlayerClass) {
	if !trainerCanTeach(n.Template.ID, class) {
		s.showNoTeach(p, n)
		return
	}
	var entries []serverpackets.AcquireSkillListEntry
	for _, l := range s.availableSkills(learnerOf(p), class, false, false, true) {
		if _, ok := s.Skill(l.SkillID, l.Level); !ok {
			continue
		}
		entries = append(entries, serverpackets.AcquireSkillListEntry{
			ID:        int32(l.SkillID), //nolint:gosec // skill IDs are small
			NextLevel: int32(l.Level),   //nolint:gosec // skill levels are small
			MaxLevel:  int32(l.Level),   //nolint:gosec // skill levels are small
			SPCost:    int32(l.SP),      //nolint:gosec // SP costs fit in int32
		})
	}
	if len(entries) > 0 {
		p.Send(serverpackets.NewAcquireSkillListPacket(serverpackets.AcquireSkillList{
			Type:   serverpackets.AcquireSkillClass,
			Skills: entries,
		}))
		return
	}
	switch minLevel := s.minLevelForNewSkill(p.Level(), class); {
	case minLevel > 0:
		p.SendSystemMessage(sysMsgLearnComeBackAtS1, serverpackets.NumberParam(int32(minLevel))) //nolint:gosec // levels are small
	case data.PlayerClass(p.ClassID()).Level() == 1: //nolint:gosec // class IDs are small
		p.SendMessage("There are no other skills to learn. Please come back after 2nd class change.")
	default:
		p.SendSystemMessage(sysMsgLearnNoOtherSkill)
		p.Send(serverpackets.NewAcquireSkillDonePacket())
	}
}

// showNoTeach is Mobius Npc.showNoTeachHtml: a warehouse keeper's
// "warehouse/<id>-noteach.htm", a trainer's "trainer/<id>-noteach.htm" (or
// its HealerTrainer script page), with %objectId% filled; a missing page
// gives Mobius's "I cannot teach you any skills." text. Other NPCs send an
// empty page, as Mobius does.
func (s *Server) showNoTeach(p *Player, n *Npc) {
	id := strconv.Itoa(n.Template.ID)
	content, ok := "", true
	switch {
	case s.npcIs(n, "Warehouse"):
		content, ok = s.htmlFile(p, "warehouse/"+id+"-noteach.htm")
	case s.npcIs(n, "Trainer"):
		content, ok = s.htmlFile(p, "trainer/"+id+"-noteach.htm")
		if !ok {
			content, ok = s.htmlFile(p, "scripts/ai/others/HealerTrainer/"+id+"-noteach.html")
		}
	}
	if !ok {
		s.log.Warn("game: NPC without a no-teach page", "npc", n.Template.ID)
		s.sendPage(p, newPage(n.ID(), "<html><body>I cannot teach you any skills.<br>You must find your current class teachers.</body></html>"))
		return
	}
	s.sendPage(p, newPage(n.ID(), content).replaceObjectID(n.ID()))
}

// lastTrainer returns the NPC p clicked last (Mobius getLastFolkNPC) when p
// may still talk to it, or is a GM; otherwise nil.
func (s *Server) lastTrainer(p *Player) *Npc {
	n := s.Npc(p.LastNpc)
	if n == nil || n.Template == nil || (!s.npcCanInteract(p, n) && !p.IsGM()) {
		return nil
	}
	return n
}

// handleRequestAcquireSkillInfo handles RequestAcquireSkillInfo (0x6B), a
// click on a skill of the trainer's list (Mobius RequestAcquireSkillInfo):
// for a class skill the trainer teaches, AcquireSkillInfo with the SP and
// the items the level takes. Fishing and clan skills wait for their
// milestones.
func (s *Server) handleRequestAcquireSkillInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAcquireSkillInfo(body)
	if err != nil {
		s.log.Debug("game: bad RequestAcquireSkillInfo", append(p.logArgs(), "err", err)...)
		return
	}
	if req.SkillID <= 0 || req.Level <= 0 {
		return
	}
	n := s.lastTrainer(p)
	if n == nil {
		return
	}
	id, level := int(req.SkillID), int(req.Level)
	if _, ok := s.Skill(id, level); !ok {
		s.log.Warn("game: RequestAcquireSkillInfo for an unknown skill", append(p.logArgs(), "skill", id, "level", level)...)
		return
	}
	if req.Type == clientpackets.AcquireSkillPledge {
		s.clanSkillInfo(p, id, level) // clan_skills.go
		return
	}
	if req.Type != clientpackets.AcquireSkillClass {
		s.log.Debug("game: fishing skills are not learned yet", append(p.logArgs(), "skill", id, "type", req.Type)...)
		return
	}
	class := learningClass(p)
	l := s.classSkillLearn(id, level, class)
	if l == nil || !trainerCanTeach(n.Template.ID, class) {
		return
	}
	info := serverpackets.AcquireSkillInfo{
		SkillID: req.SkillID,
		Level:   req.Level,
		SPCost:  int32(l.SP), //nolint:gosec // SP costs fit in int32
		Type:    serverpackets.AcquireSkillClass,
	}
	for _, it := range l.Items {
		info.Items = append(info.Items, serverpackets.AcquireSkillItem{ItemID: int32(it.ID), Count: it.Count}) //nolint:gosec // item IDs are small
	}
	p.Send(serverpackets.NewAcquireSkillInfoPacket(info))
}

// handleRequestAcquireSkill handles RequestAcquireSkill (0x6C), learning a
// skill at the trainer (Mobius RequestAcquireSkill): a skill ID below 1 or
// a level outside 1-1000 is an illegal action; then the trainer must still
// be in reach and the level must be one of the learning class's tree. A
// class skill is checked and paid for (checkLearnSkill), then given
// (giveLearnedSkill). Fishing and clan skills wait for their milestones.
func (s *Server) handleRequestAcquireSkill(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestAcquireSkill(body)
	if err != nil {
		s.log.Debug("game: bad RequestAcquireSkill", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Level < 1 || req.Level > 1000 || req.SkillID < 1 {
		s.illegalAction(p, "wrong RequestAcquireSkill data: skill "+strconv.Itoa(int(req.SkillID))+" level "+strconv.Itoa(int(req.Level)))
		return
	}
	n := s.lastTrainer(p)
	if n == nil {
		return
	}
	id, level := int(req.SkillID), int(req.Level)
	sk, ok := s.Skill(id, level)
	if !ok {
		s.log.Warn("game: RequestAcquireSkill for an unknown skill", append(p.logArgs(), "skill", id, "level", level)...)
		return
	}
	if req.Type == clientpackets.AcquireSkillPledge {
		s.learnClanSkill(p, sk) // clan_skills.go
		return
	}
	if req.Type != clientpackets.AcquireSkillClass {
		s.log.Debug("game: fishing skills are not learned yet", append(p.logArgs(), "skill", id, "type", req.Type)...)
		return
	}
	class := learningClass(p)
	l := s.classSkillLearn(id, level, class)
	if l == nil {
		return
	}
	if s.checkLearnSkill(p, n, l, class) {
		s.giveLearnedSkill(p, n, sk, class)
	}
}

// checkLearnSkill is Mobius RequestAcquireSkill.checkPlayerSkill: p's level
// must reach the entry's ("You do not meet the skill level requirements.");
// p must have the SP (else the message and the list again), the
// prerequisite skills and the items (the list again when items are
// missing). The items are destroyed ("... has disappeared.") and the SP
// taken, with UserInfo. It reports whether p may have the skill.
func (s *Server) checkLearnSkill(p *Player, n *Npc, l *data.SkillLearn, class data.PlayerClass) bool {
	if l.GetLevel > p.Level() {
		p.SendMessage("You do not meet the skill level requirements.")
		s.log.Warn("game: learning a skill below its level", append(p.logArgs(), "skill", l.SkillID, "level", l.Level, "needs", l.GetLevel)...)
		return false
	}
	sp := l.SP
	if sp > 0 && sp > int(p.SP()) {
		p.SendSystemMessage(sysMsgLearnNotEnoughSP)
		s.showSkillList(p, n, class)
		return false
	}
	if !divineSPBookNeeded && l.SkillID == learnDivineInspiration {
		return true
	}
	for _, pre := range l.PreRequisites {
		if p.SkillLevel(pre.ID) < pre.Level {
			p.SendSystemMessage(sysMsgLearnNoMaterials)
			return false
		}
	}
	if len(l.Items) > 0 {
		for _, it := range l.Items {
			if p.inv.Count(int32(it.ID), -1, true) < it.Count { //nolint:gosec // item IDs are small
				p.SendSystemMessage(sysMsgLearnNoMaterials)
				s.showSkillList(p, n, class)
				return false
			}
		}
		for _, it := range l.Items {
			if !s.DestroyItemByItemID(p, int32(it.ID), it.Count, true) { //nolint:gosec // item IDs are small
				s.log.Warn("game: a learned skill's item went missing", append(p.logArgs(), "skill", l.SkillID, "level", l.Level, "item", it.ID)...)
			}
		}
	}
	if sp > 0 {
		s.removeSP(p, int64(sp))
		p.UpdateUserInfo()
	}
	return true
}

// giveLearnedSkill is Mobius RequestAcquireSkill.giveSkill: "You have
// earned S1.", the skill added and saved, SkillList, the OnSkillLearned
// hooks (the shortcuts), the trainer's list again, and ExStorageMaxCount for
// the storage skills.
func (s *Server) giveLearnedSkill(p *Player, n *Npc, sk *skills.Skill, class data.PlayerClass) {
	p.SendSystemMessage(sysMsgLearnEarnedS1, skills.SkillName(sk))
	s.AddSkill(p, sk, true)
	p.SendSkillList()
	for _, fn := range s.learn.learned {
		fn(p, sk)
	}
	s.showSkillList(p, n, class)
	if sk.ID >= firstStorageSkill && sk.ID <= lastStorageSkill {
		p.Send(s.storageMaxCount(p))
	}
}

// checkPlayerSkills is Mobius Player.checkPlayerSkills
// (DecreaseSkillOnDelevel): every known skill whose level p's class tree
// teaches more than 9 levels above p's level (Expertise: any level above)
// drops to the highest level p may have, or goes when there is none. Both
// are saved. It reports whether a skill changed.
func (s *Server) checkPlayerSkills(p *Player) bool {
	d := s.skills.data
	if d == nil {
		return false
	}
	tree, ok := d.SkillTree(p.ClassID())
	if !ok {
		return false
	}
	changed := false
	for _, sk := range p.Skills() {
		l := findSkillLearn(tree, sk.ID, sk.Level%100)
		if l == nil {
			continue
		}
		margin := deleveledSkillMargin
		if sk.ID == learnExpertise {
			margin = 0
		}
		if p.Level() < l.GetLevel-margin {
			s.decreaseSkillLevel(p, tree, sk, margin)
			changed = true
		}
	}
	return changed
}

// decreaseSkillLevel is Mobius deacreaseSkillLevel: sk becomes the highest
// level of its tree entries p's level allows with the margin, saved; without
// one it is removed (and its effects end).
func (s *Server) decreaseSkillLevel(p *Player, tree []data.SkillLearn, sk *skills.Skill, margin int) {
	next := -1
	for i := range tree {
		l := &tree[i]
		if l.SkillID == sk.ID && next < l.Level && p.Level() >= l.GetLevel-margin {
			next = l.Level
		}
	}
	if next == -1 {
		s.log.Info("game: removing a skill after a level loss", append(p.logArgs(), "skill", sk.ID, "level", sk.Level)...)
		s.RemoveSkill(p, sk.ID, true, true)
		return
	}
	lower, ok := s.Skill(sk.ID, next)
	if !ok {
		return
	}
	s.log.Info("game: lowering a skill after a level loss", append(p.logArgs(), "skill", sk.ID, "level", sk.Level, "to", next)...)
	s.AddSkill(p, lower, true)
}

// NewCharacterSkills returns the skills a new character of class classID
// starts with, as Mobius CharacterCreate adds them: every level-1 entry of
// the class tree a level-1 player can take, auto-get skills included, in the
// tree's order. The session layer stores them with the character it
// creates. Safe on any goroutine (it reads only the skill data).
//
// Phase 7.6 adds the initial shortcuts (InitialShortcutData) next to this.
func (s *Server) NewCharacterSkills(classID int) []storage.CharacterSkill {
	none := func(int) int { return 0 }
	var out []storage.CharacterSkill
	learner := learnerLevels{level: 1, known: none}
	for _, l := range s.availableSkills(learner, data.PlayerClass(classID), false, true, true) { //nolint:gosec // class IDs are small
		if _, ok := s.Skill(l.SkillID, l.Level); !ok {
			continue
		}
		out = append(out, storage.CharacterSkill{
			ClassIndex: classIndex,
			ID:         int32(l.SkillID), //nolint:gosec // skill IDs are small
			Level:      int32(l.Level),   //nolint:gosec // skill levels are small
		})
	}
	return out
}
