package game

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The skill and buff GM commands of Plan 10 Phase 7.5 (Mobius AdminSkill and
// AdminBuffs): //add_skill, //remove_skill, //give_all_skills, //buff,
// //getbuffs, //getbuffs_ps, //stopbuff, //stopallbuffs and //removereuse,
// with Mobius's answers.

// gmSysMsgThatIsIncorrectTarget is "That is the incorrect target." (Mobius
// THAT_IS_THE_INCORRECT_TARGET).
const gmSysMsgThatIsIncorrectTarget serverpackets.SystemMessageID = 144

// setupSkillLearningGM registers the skill and buff GM commands.
func setupSkillLearningGM(s *Server) error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("add_skill", admin, s.gmAddSkill),
		s.RegisterGMCommand("remove_skill", admin, s.gmRemoveSkill),
		s.RegisterGMCommand("give_all_skills", admin, s.gmGiveAllSkills),
		s.RegisterGMCommand("buff", admin, s.gmBuff),
		s.RegisterGMCommand("getbuffs", admin, s.gmGetBuffs(false)),
		s.RegisterGMCommand("getbuffs_ps", admin, s.gmGetBuffs(true)),
		s.RegisterGMCommand("stopbuff", admin, s.gmStopBuff),
		s.RegisterGMCommand("stopallbuffs", admin, s.gmStopAllBuffs),
		s.RegisterGMCommand("removereuse", admin, s.gmRemoveReuse),
	)
}

// gmSysMessage is Mobius Player.sendSysMessage with GMStartupBuilderHide on
// (its default): general chat from "SYS".
func gmSysMessage(p *Player, text string) {
	p.Send(serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{
		Type:       serverpackets.ChatGeneral,
		SenderName: "SYS",
		Text:       text,
	}))
}

// javaParseIntError is the message of Java's NumberFormatException for a
// string Integer.parseInt refuses.
func javaParseIntError(s string) string { return `For input string: "` + s + `"` }

// javaParseInt is Integer.parseInt: a decimal int with an optional sign.
func javaParseInt(s string) (int, bool) {
	v, err := strconv.ParseInt(s, 10, 32)
	return int(v), err == nil
}

// gmShowSkillsPage is AdminSkill.showMainPage: the skill editor page of the
// GM's target player, or "Invalid target.".
func (s *Server) gmShowSkillsPage(gm *Player) {
	t, ok := gmTargetPlayer(gm)
	if !ok {
		gm.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	s.sendPage(gm, s.pageFile(gm, 0, "admin/charskills.htm").
		replace("%name%", t.Name()).
		replace("%level%", strconv.Itoa(t.Level())).
		replace("%class%", s.className(t.ClassID())))
}

// className is the class's name from classList.xml (Mobius ClassListData).
func (s *Server) className(id int) string {
	if c, ok := s.data.ClassTemplate(id); ok {
		return c.Name
	}
	return strconv.Itoa(id)
}

// gmAddSkill is //add_skill <id> [level] (Mobius AdminSkill.adminAddSkill):
// the target player gets the skill level (level 1 without one), saved, with
// "Admin gave you the skill ..."; the GM is told and both get SkillList.
// The skill editor page follows.
func (s *Server) gmAddSkill(gm *Player, args []string) {
	t, ok := gmTargetPlayer(gm)
	if !ok {
		gm.SendSystemMessage(gmSysMsgInvalidTarget)
		s.gmShowSkillsPage(gm)
		return
	}
	if len(args) != 1 && len(args) != 2 {
		s.gmShowSkillsPage(gm)
		return
	}
	var sk *skills.Skill
	id, okID := javaParseInt(args[0])
	level, okLevel := 1, true
	if len(args) == 2 {
		level, okLevel = javaParseInt(args[1])
	}
	if okID && okLevel {
		sk, _ = s.Skill(id, level)
	}
	if sk == nil {
		gmSysMessage(gm, "Error: there is no such skill.")
		s.gmShowSkillsPage(gm)
		return
	}
	t.SendMessage("Admin gave you the skill " + sk.Name + ".")
	s.AddSkill(t, sk, true)
	t.SendSkillList()
	gmSysMessage(gm, "You gave the skill "+sk.Name+" to "+t.Name()+".")
	gm.SendSkillList()
	s.gmShowSkillsPage(gm)
}

// gmRemoveSkill is //remove_skill <id> (Mobius AdminSkill.adminRemoveSkill):
// the target player loses the skill (its effects end; the save drops it)
// and is told; the GM is told and gets its own SkillList, as in Mobius. The
// page to remove skills follows.
func (s *Server) gmRemoveSkill(gm *Player, args []string) {
	if len(args) == 0 {
		gmSysMessage(gm, "Usage: //remove_skill <skill_id>")
		return
	}
	id, ok := javaParseInt(strings.Join(args, " "))
	if !ok {
		gmSysMessage(gm, "Usage: //remove_skill <skill_id>")
		return
	}
	t, ok := gmTargetPlayer(gm)
	if !ok {
		gm.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	sk, found := s.Skill(id, t.SkillLevel(id))
	if !found {
		gmSysMessage(gm, "Error: there is no such skill.")
	} else {
		t.SendMessage("Admin removed the skill " + sk.Name + " from your skills list.")
		s.RemoveSkill(t, id, true, true)
		gmSysMessage(gm, "You removed the skill "+sk.Name+" from "+t.Name()+".")
		gm.SendSkillList()
	}
	s.gmRemoveSkillsPage(gm, 0)
}

// gmRemoveSkillsPage is AdminSkill.removeSkillsPage: the target player's
// skills, 10 to a page, each a link to //remove_skill.
func (s *Server) gmRemoveSkillsPage(gm *Player, page int) {
	t, ok := gmTargetPlayer(gm)
	if !ok {
		gm.SendSystemMessage(gmSysMsgThatIsIncorrectTarget)
		return
	}
	const perPage = 10
	known := t.Skills()
	maxPages := len(known) / perPage
	if len(known) > perPage*maxPages {
		maxPages++
	}
	page = min(page, maxPages)
	start := perPage * page
	end := len(known)
	if end-start > perPage {
		end = start + perPage
	}
	var b strings.Builder
	b.WriteString(`<html><body><table width=260><tr><td width=40><button value="Main" action="bypass admin_admin" width=40 height=15 back="sek.cbui94" fore="sek.cbui92"></td><td width=180><center>Character Selection Menu</center></td><td width=40><button value="Back" action="bypass -h admin_show_skills" width=40 height=15 back="sek.cbui94" fore="sek.cbui92"></td></tr></table><br><br><center>Editing <font color="LEVEL">`)
	b.WriteString(t.Name())
	b.WriteString(`</font></center><br><table width=270><tr><td>Lv: `)
	b.WriteString(strconv.Itoa(t.Level()) + " " + s.className(t.ClassID()))
	b.WriteString(`</td></tr></table><br><table width=270><tr><td>Note: Do not forget that modifying players skills can</td></tr><tr><td>ruin the game...</td></tr></table><br><center>Click on the skill you wish to remove:</center><br><center><table width=270><tr>`)
	for x := range maxPages {
		b.WriteString(`<td><a action="bypass -h admin_remove_skills ` + strconv.Itoa(x) + `">Page ` + strconv.Itoa(x+1) + `</a></td>`)
	}
	b.WriteString(`</tr></table></center><br><table width=270><tr><td width=80>Name:</td><td width=60>Level:</td><td width=40>Id:</td></tr>`)
	for i := start; i < end; i++ {
		sk := known[i]
		id := strconv.Itoa(sk.ID)
		b.WriteString(`<tr><td width=80><a action="bypass -h admin_remove_skill ` + id + `">` + sk.Name + `</a></td><td width=60>` + strconv.Itoa(sk.Level) + `</td><td width=40>` + id + `</td></tr>`)
	}
	b.WriteString(`</table><br><center><table>Remove skill by ID :<tr><td>Id: </td><td><edit var="id_to_remove" width=110></td></tr></table></center><center><button value="Remove skill" action="bypass -h admin_remove_skill $id_to_remove" width=110 height=15 back="sek.cbui94" fore="sek.cbui92"></center><br><center><button value="Back" action="bypass -h admin_current_player" width=40 height=15 back="sek.cbui94" fore="sek.cbui92"></center></body></html>`)
	s.sendPage(gm, newPage(0, b.String()))
}

// gmGiveAllSkills is //give_all_skills (Mobius AdminSkill.adminGiveAllSkills
// → Player.giveAvailableSkills): the target player gets every skill of its
// class it can have at its level, auto-get skills included, spellbook
// skills and Forgotten Scroll skills not; the GM is told how many skills
// were new. Mobius matches the command only without arguments.
func (s *Server) gmGiveAllSkills(gm *Player, args []string) {
	if len(args) > 0 {
		return
	}
	t, ok := gmTargetPlayer(gm)
	if !ok {
		gm.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	n := s.giveAvailableSkills(t, false, false)
	gmSysMessage(gm, "You gave "+strconv.Itoa(n)+" skills to "+t.Name())
	t.SendSkillList()
}

// giveAvailableSkills is Mobius Player.giveAvailableSkills(byFS, autoGet
// true, withItems): every skill level getAllAvailableSkills finds that p
// does not know yet is added and saved (a toggle of it that runs is stopped
// first). It returns how many of them p did not know at all.
func (s *Server) giveAvailableSkills(p *Player, byFS, withItems bool) int {
	all := s.allAvailableSkills(p, byFS, withItems)
	ids := make([]int, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	slices.Sort(ids) // Mobius iterates a HashMap of small Integer keys
	count := 0
	for _, id := range ids {
		sk := all[id]
		if p.KnownSkill(id) == sk {
			continue
		}
		if p.SkillLevel(id) == 0 {
			count++
		}
		if sk.IsToggle() && p.Effects().IsAffectedBySkill(id) {
			p.Effects().StopSkill(skills.FinishRemoved, id)
		}
		s.AddSkill(p, sk, true)
	}
	return count
}

// allAvailableSkills is Mobius SkillTreeData.getAllAvailableSkills: p's
// skills that its class may have (isSkillAllowed), then, round after round,
// the levels availableSkills offers on top of them, until a round offers
// nothing new. It returns the resulting level of each skill by ID.
func (s *Server) allAvailableSkills(p *Player, byFS, withItems bool) map[int]*skills.Skill {
	result := make(map[int]*skills.Skill)
	for _, sk := range p.Skills() {
		if s.skillAllowed(p, sk) {
			result[sk.ID] = sk
		}
	}
	who := learnerLevels{level: p.Level(), gm: p.IsGM(), known: func(id int) int {
		if sk := result[id]; sk != nil {
			return sk.Level
		}
		return 0
	}}
	class := learningClass(p)
	var prev []*data.SkillLearn
	for {
		learnable := s.availableSkills(who, class, byFS, true, withItems)
		if len(learnable) == 0 || slices.Equal(learnable, prev) {
			break
		}
		for _, l := range learnable {
			if sk, ok := s.Skill(l.SkillID, l.Level); ok {
				result[sk.ID] = sk
			}
		}
		prev = learnable
	}
	return result
}

// skillAllowed is Mobius SkillTreeData.isSkillAllowed: a skill excluded from
// the check, a GM's Game Master skill, a level (capped at the skill's
// highest) of p's complete class tree, or a fishing skill of p's race or of
// every race.
func (s *Server) skillAllowed(p *Player, sk *skills.Skill) bool {
	d := s.skills.data
	if sk.ExcludedFromCheck || d == nil {
		return true
	}
	if p.IsGM() && (inSkillTree(d.OtherSkillTree(data.TreeGameMaster), sk.ID, sk.Level) || inSkillTree(d.OtherSkillTree(data.TreeGameMasterAura), sk.ID, sk.Level)) {
		return true
	}
	level := min(sk.Level, d.MaxLevel(sk.ID))
	if tree, ok := d.SkillTree(p.ClassID()); ok && findSkillLearn(tree, sk.ID, level) != nil {
		return true
	}
	race := data.Race(p.char.Race) //nolint:gosec // races are small
	for _, l := range d.OtherSkillTree(data.TreeFishing) {
		if l.SkillID == sk.ID && l.Level == level && (len(l.Races) == 0 || slices.Contains(l.Races, race)) {
			return true
		}
	}
	return false
}

// inSkillTree reports whether tree teaches the skill level.
func inSkillTree(tree []data.SkillLearn, id, level int) bool {
	return findSkillLearn(tree, id, level) != nil
}

// gmBuff is //buff <id> [level] (Mobius AdminBuffs): the skill's effects
// land on the GM's target creature, with the GM as their caster; without a
// level the skill's highest.
func (s *Server) gmBuff(gm *Player, args []string) {
	target, ok := gm.Target.(Creature)
	if !ok || target == nil || s.skillCreatureOf(target) == nil {
		gm.SendSystemMessage(gmSysMsgThatIsIncorrectTarget)
		return
	}
	if len(args) == 0 {
		gmSysMessage(gm, "Skill Id and level are not specified.")
		gmSysMessage(gm, "Usage: //buff <skillId> <skillLevel>")
		return
	}
	id, ok := javaParseInt(args[0])
	if !ok {
		gmSysMessage(gm, "Failed buffing: "+javaParseIntError(args[0]))
		gmSysMessage(gm, "Usage: //buff <skillId> <skillLevel>")
		return
	}
	level := 0
	if d := s.skills.data; d != nil {
		level = d.MaxLevel(id)
	}
	if len(args) > 1 {
		if level, ok = javaParseInt(args[1]); !ok {
			gmSysMessage(gm, "Failed buffing: "+javaParseIntError(args[1]))
			gmSysMessage(gm, "Usage: //buff <skillId> <skillLevel>")
			return
		}
	}
	sk, found := s.Skill(id, level)
	if !found {
		gmSysMessage(gm, "Skill with id: "+strconv.Itoa(id)+", level: "+strconv.Itoa(level)+" not found.")
		return
	}
	gmSysMessage(gm, "Admin buffing "+sk.Name+" ("+strconv.Itoa(id)+","+strconv.Itoa(level)+")")
	sk.ApplyTo(s.skills.env, s.skillCreatureOf(gm), s.skillCreatureOf(target))
}

// gmGetBuffs is //getbuffs and //getbuffs_ps [player name or object ID]
// [page] (Mobius AdminBuffs): the effect page of a creature, the GM's
// target without an argument.
func (s *Server) gmGetBuffs(passive bool) GMCommand {
	return func(gm *Player, args []string) {
		if len(args) == 0 {
			target, ok := gm.Target.(Creature)
			if !ok || target == nil || s.skillBaseOf(target) == nil {
				gm.SendSystemMessage(gmSysMsgThatIsIncorrectTarget)
				return
			}
			s.gmShowBuffs(gm, target, passive)
			return
		}
		var target Creature
		if t, ok := s.PlayerByName(args[0]); ok {
			target = t
		} else if id, ok := javaParseInt(args[0]); ok {
			target = s.gmCreatureByID(id)
		}
		if target == nil {
			gmSysMessage(gm, "No online player or creature matches "+args[0]+".")
			return
		}
		s.gmShowBuffs(gm, target, passive)
	}
}

// gmCreatureByID returns the player or NPC with the object ID, nil if none.
func (s *Server) gmCreatureByID(id int) Creature {
	o, ok := s.world.Find(world.ObjectID(id)) //nolint:gosec // object IDs fit in int32
	if !ok {
		return nil
	}
	if c, ok := o.(Creature); ok && s.skillBaseOf(c) != nil {
		return c
	}
	return nil
}

// gmShowBuffs is AdminBuffs.showBuffs: the target's effects (or passives)
// on admin/getbuff_skills.htm, one row per skill with its time left and a
// button to stop it. Mobius pages them 7 at a time; this page lists them
// all.
func (s *Server) gmShowBuffs(gm *Player, target Creature, passive bool) {
	b := s.skillBaseOf(target)
	if b == nil {
		return
	}
	infos := b.el.Effects()
	if passive {
		infos = b.el.Passives()
	}
	var order []*skills.Skill
	times := make(map[*skills.Skill]int)
	for _, info := range infos {
		sk := info.Skill()
		if sk == nil {
			continue
		}
		if old, ok := times[sk]; ok {
			times[sk] = max(info.AbnormalTime(), old)
			continue
		}
		order = append(order, sk)
		times[sk] = info.Time()
	}
	objID := strconv.Itoa(int(target.ID()))
	var body strings.Builder
	for _, sk := range order {
		nameWidth, timeWidth := "650", "200"
		if sk.IsPassive() {
			nameWidth, timeWidth = "850", "1"
		}
		body.WriteString(`<table width="295" bgcolor="000000"><tr><td height=40 width=40><img src="`)
		body.WriteString(sk.Icon)
		body.WriteString(`" width=32 height=32></td><td width=`)
		body.WriteString(nameWidth)
		body.WriteString(`><font color="B09878">`)
		if passive {
			body.WriteString(`<a action="bypass admin_viewskilleffects_ps `)
		} else {
			body.WriteString(`<a action="bypass admin_viewskilleffects `)
		}
		body.WriteString(objID + " " + strconv.Itoa(sk.ID) + " " + strconv.Itoa(sk.Level) + `">` + sk.Name)
		if !strings.Contains(sk.Name, "Lv") && !strings.Contains(sk.Name, "Level") {
			body.WriteString(" Lv. " + strconv.Itoa(sk.Level))
		}
		body.WriteString(" (" + strconv.Itoa(sk.ID) + `)</a></font></td><td fixwidth=`)
		body.WriteString(timeWidth)
		body.WriteString(` height=32 valign="center" align="center">`)
		switch {
		case sk.IsPassive():
		case sk.IsToggle():
			body.WriteString("T")
		default:
			body.WriteString(strconv.Itoa(times[sk]) + "s")
		}
		body.WriteString(`</td><td><table><tr><td></td></tr><tr><td fixwidth=360><button value="X" action="bypass admin_stopbuff `)
		body.WriteString(objID + " " + strconv.Itoa(sk.ID))
		body.WriteString(`" width=30 height=15 back="sek.cbui94" fore="sek.cbui92"></td></tr></table></td></tr></table><img src="L2UI.SquareGray" width=295 height=1>`)
	}
	buffsText, passives := "Show Passives", "_ps"
	if passive {
		buffsText, passives = "Hide Passives", ""
	}
	s.sendPage(gm, s.pageFile(gm, 0, "admin/getbuff_skills.htm").
		replace("%pages%", "").
		replace("%buffsText%", buffsText).
		replace("%passives%", passives).
		replace("%targetName%", target.Name()).
		replace("%targetObjId%", objID).
		replace("%buffs%", body.String()).
		replace("%skillSize%", strconv.Itoa(len(order))))
}

// gmStopBuff is //stopbuff <objectId> <skillId> (Mobius AdminBuffs): the
// creature's effects of the skill end, and its effect page shows again.
func (s *Server) gmStopBuff(gm *Player, args []string) {
	const usage = "Usage: //stopbuff <objectId> <skillId>"
	if len(args) < 2 {
		gmSysMessage(gm, "Failed removing effect: null")
		gmSysMessage(gm, usage)
		return
	}
	objID, ok1 := javaParseInt(args[0])
	skillID, ok2 := javaParseInt(args[1])
	if !ok1 || !ok2 {
		bad := args[0]
		if ok1 {
			bad = args[1]
		}
		gmSysMessage(gm, "Failed removing effect: "+javaParseIntError(bad))
		gmSysMessage(gm, usage)
		return
	}
	target := s.gmCreatureByID(objID)
	if target == nil || skillID <= 0 {
		return
	}
	if el := s.skillBaseOf(target).el; el.IsAffectedBySkill(skillID) {
		el.StopSkill(skills.FinishRemoved, skillID)
		gmSysMessage(gm, "Removed skill ID: "+strconv.Itoa(skillID)+" effects from "+target.Name()+" ("+strconv.Itoa(objID)+").")
	}
	s.gmShowBuffs(gm, target, false)
}

// gmStopAllBuffs is //stopallbuffs <objectId> (Mobius AdminBuffs): every
// effect of the creature but its passives ends (a player's UserInfo and
// CharInfo follow), the GM is told, and the creature's effect page shows.
// An unknown object ID does nothing.
func (s *Server) gmStopAllBuffs(gm *Player, args []string) {
	const usage = "Usage: //stopallbuffs <objectId>"
	if len(args) == 0 {
		gmSysMessage(gm, "Failed removing all effects: null")
		gmSysMessage(gm, usage)
		return
	}
	objID, ok := javaParseInt(args[0])
	if !ok {
		gmSysMessage(gm, "Failed removing all effects: "+javaParseIntError(args[0]))
		gmSysMessage(gm, usage)
		return
	}
	target := s.gmCreatureByID(objID)
	if target == nil {
		return
	}
	s.skillBaseOf(target).el.StopAll()
	if t, isPlayer := target.(*Player); isPlayer {
		t.BroadcastUserInfo() // Player.stopAllEffects: updateAndBroadcastStatus
	}
	gmSysMessage(gm, "Removed all effects from "+target.Name()+" ("+strconv.Itoa(objID)+")")
	s.gmShowBuffs(gm, target, false)
}

// gmRemoveReuse is //removereuse [player name] (Mobius AdminBuffs): every
// skill reuse of the player, or of the GM's target player, ends; it gets
// its SkillCoolTime and the GM is told.
func (s *Server) gmRemoveReuse(gm *Player, args []string) {
	var t *Player
	if len(args) > 0 {
		var ok bool
		if t, ok = s.PlayerByName(args[0]); !ok {
			gmSysMessage(gm, "The player "+args[0]+" is not online.")
			return
		}
	} else {
		var ok bool
		if t, ok = gmTargetPlayer(gm); !ok {
			gm.SendSystemMessage(gmSysMsgThatIsIncorrectTarget)
			return
		}
	}
	s.ClearSkillReuse(t)
	t.Send(s.SkillCoolTimePacket(t))
	gmSysMessage(gm, "Skill reuse was removed from "+t.Name()+".")
}
