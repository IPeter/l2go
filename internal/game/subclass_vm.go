package game

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/subclass"
)

// The village master's subclass pages (Mobius VillageMaster.onBypassFeedback,
// "Subclass <choice> ..."): 0 the menu, 1 and 4 add a subclass, 2 and 5
// switch the class, 3, 6 and 7 replace a subclass. The pages are Mobius's
// villagemaster/SubClass*.htm.

// System messages of the subclass pages.
const (
	sysMsgSubclassAdded         serverpackets.SystemMessageID = 1269 // "The new sub class has been added."
	sysMsgSubclassTransferred   serverpackets.SystemMessageID = 1270 // "The transfer of sub class has been completed."
	sysMsgSubclassSkillInUse    serverpackets.SystemMessageID = 1295 // "Sub classes may not be created or changed while a skill is in use."
	sysMsgSubclassOverweight    serverpackets.SystemMessageID = 1894 // "A sub-class cannot be created or changed while you are over your weight limit."
	sysMsgSubclassInventoryFull serverpackets.SystemMessageID = 2033 // "A sub-class cannot be created or changed because you have exceeded your inventory limit."
)

// subclassPages is the folder of the subclass pages.
const subclassPages = "villagemaster/"

// subclassChangeSkill is the effect a subclass switch or replacement shows
// (MagicSkillUse 5103, as a class change).
const subclassChangeSkill = classChangeSkill

// vmSubclass runs a "Subclass" command of a village master's page.
func (s *Server) vmSubclass(p *Player, n *Npc, args []string) {
	command := strings.Join(append([]string{"Subclass"}, args...), " ")
	if p.skill.IsCastingNow() || allSkillsDisabled(&p.skill.skillBase) {
		p.SendSystemMessage(sysMsgSubclassSkillInUse)
		return
	}
	if h := s.subclass.hooks.LeaveOlympiad; h != nil {
		h(p)
	}
	id := n.ID()
	if s.SummonOf(p) != nil {
		s.sendPage(p, s.pageFile(p, id, subclassPages+"SubClass_NoSummon.htm"))
		return
	}
	if !s.inventoryUnder90(p) {
		p.SendSystemMessage(sysMsgSubclassInventoryFull)
		return
	}
	if !p.admin.diet && p.WeightPenalty() >= 2 {
		p.SendSystemMessage(sysMsgSubclassOverweight)
		return
	}
	cmd := subclass.ParseCommand(command)
	if cmd.Bad {
		s.log.Warn("game: wrong numeric values for a subclass command", append(p.logArgs(), "bypass", command)...)
	}
	cfg := s.subclass.cfg
	master := subclass.Master{Race: n.Template.Race, Teach: s.masterTeachType(n)}
	var pg *htmlPage
	switch cmd.Choice {
	case 0:
		pg = s.pageFile(p, id, subclassPages+"SubClass.htm")
	case 1:
		if p.TotalSubClasses() >= cfg.MaxSubclass {
			pg = s.pageFile(p, id, subclassPages+"SubClass_Fail.htm")
			break
		}
		avail := subclass.Available(s.classCategories(), subclassOwner(p), master, cfg)
		if len(avail) == 0 {
			p.SendMessage("There are no sub classes available at this time.")
			return
		}
		var b strings.Builder
		for _, c := range avail {
			name := s.className(int(c))
			b.WriteString(`<a action="bypass npc_%objectId%_Subclass 4 ` + strconv.Itoa(int(c)) + `" msg="1268;` + name + `">` + name + `</a><br>`)
		}
		pg = s.pageFile(p, id, subclassPages+"SubClass_Add.htm").replace("%list%", b.String())
	case 2:
		if p.TotalSubClasses() == 0 {
			pg = s.pageFile(p, id, subclassPages+"SubClass_ChangeNo.htm")
			break
		}
		var b strings.Builder
		b.WriteString(`<a action="bypass -h npc_%objectId%_Subclass 5 0">` + s.className(p.BaseClassID()) + `</a><br>`)
		for _, sub := range p.Subclasses() {
			b.WriteString(`<a action="bypass -h npc_%objectId%_Subclass 5 ` + strconv.Itoa(int(sub.ClassIndex)) + `">` + s.className(int(sub.ClassID)) + `</a><br>`)
		}
		pg = s.pageFile(p, id, subclassPages+"SubClass_Change.htm").replace("%list%", b.String())
	case 3:
		subs := p.Subclasses()
		switch {
		case len(subs) == 0:
			pg = s.pageFile(p, id, subclassPages+"SubClass_ModifyEmpty.htm")
		case len(subs) > 3:
			var b strings.Builder
			for i, sub := range subs {
				b.WriteString("Sub-class " + strconv.Itoa(i+1) + `<br><a action="bypass -h npc_%objectId%_Subclass 6 ` + strconv.Itoa(int(sub.ClassIndex)) + `">` + s.className(int(sub.ClassID)) + `</a><br>`)
			}
			pg = s.pageFile(p, id, subclassPages+"SubClass_ModifyCustom.htm").replace("%list%", b.String())
		default:
			pg = s.pageFile(p, id, subclassPages+"SubClass_Modify.htm")
			for i := int32(1); i <= 3; i++ {
				key := "%sub" + strconv.Itoa(int(i)) + "%"
				if sub := p.subclassOf(i); sub != nil {
					pg.replace(key, s.className(int(sub.ClassID)))
				} else {
					pg.replace(`<a action="bypass npc_%objectId%_Subclass 6 `+strconv.Itoa(int(i))+`">`+key+`</a><br>`, "")
				}
			}
		}
	case 4:
		if !s.subclassAddAllowed(p, cmd.One) {
			pg = s.pageFile(p, id, subclassPages+"SubClass_Fail.htm")
			break
		}
		index := int32(p.TotalSubClasses() + 1) //nolint:gosec // at most 3
		if !s.addSubclass(p, cmd.One, index) {
			return
		}
		s.setActiveClass(p, index, true)
		pg = s.pageFile(p, id, subclassPages+"SubClass_AddOk.htm")
		p.SendSystemMessage(sysMsgSubclassAdded)
	case 5:
		if p.ClassIndex() == cmd.One {
			pg = s.pageFile(p, id, subclassPages+"SubClass_Current.htm")
			break
		}
		if cmd.One < 0 || cmd.One > subclass.MaxIndex || !s.setActiveClass(p, int32(cmd.One), true) { //nolint:gosec // checked above
			return
		}
		p.SendSystemMessage(sysMsgSubclassTransferred)
		s.subclassEffect(p)
		return
	case 6:
		if cmd.One < 1 || cmd.One > cfg.MaxSubclass {
			return
		}
		avail := subclass.Available(s.classCategories(), subclassOwner(p), master, cfg)
		if len(avail) == 0 {
			p.SendMessage("There are no sub classes available at this time.")
			return
		}
		var b strings.Builder
		for _, c := range avail {
			b.WriteString(`<a action="bypass npc_%objectId%_Subclass 7 ` + strconv.Itoa(cmd.One) + ` ` + strconv.Itoa(int(c)) + `" msg="1445;">` + s.className(int(c)) + `</a><br>`)
		}
		page := "SubClass_ModifyChoice.htm"
		if cmd.One <= 3 {
			page = "SubClass_ModifyChoice" + strconv.Itoa(cmd.One) + ".htm"
		}
		pg = s.pageFile(p, id, subclassPages+page).replace("%list%", b.String())
	case 7:
		if !subclass.ValidNew(s.classCategories(), subclassOwner(p), cmd.Two) {
			return
		}
		index := int32(cmd.One) //nolint:gosec // a page's small number
		replacedActive := p.classIndex() == index
		if !s.modifySubclass(p, index, cmd.Two) {
			// Mobius: "modifySubClass() removed subclass from memory".
			s.setActiveClass(p, baseClassIndex, !replacedActive)
			p.SendMessage("The sub class could not be added, you have been reverted to your base class.")
			return
		}
		s.AbortCast(p)
		p.skill.el.StopAllExceptStayAfterDeath()
		p.skill.el.StopAllNotStayOnSubclassChange()
		s.StopCubics(p)
		s.setActiveClass(p, index, !replacedActive)
		pg = s.pageFile(p, id, subclassPages+"SubClass_ModifyOk.htm").replace("%name%", s.className(cmd.Two))
		p.SendSystemMessage(sysMsgSubclassAdded)
		s.subclassEffect(p)
	default:
		return
	}
	s.sendPage(p, pg.replaceObjectID(id))
}

// subclassAddAllowed is "Subclass 4" before the addition: the count and
// level rules (subclass.CanAdd), the quests unless AltSubclassWithoutQuests
// (Fate's Whisper and Mimir's Elixir, or noblesse), and a valid choice
// (subclass.ValidNew).
func (s *Server) subclassAddAllowed(p *Player, classID int) bool {
	cfg := s.subclass.cfg
	o := subclassOwner(p)
	if !subclass.CanAdd(o, cfg) {
		return false
	}
	if !cfg.WithoutQuests && !subclass.QuestsDone(p.IsNoble(), func(name string) bool {
		st := s.QuestState(p, name)
		return st != nil && st.IsCompleted()
	}) {
		return false
	}
	return subclass.ValidNew(s.classCategories(), o, classID)
}

// subclassEffect shows the class change effect to p and the players around
// (MagicSkillUse 5103 without hit time).
func (s *Server) subclassEffect(p *Player) {
	if p.InWorld() {
		s.world.Broadcast(p, castUsePacket(p, p, subclassChangeSkill, 1, 0, 0), true)
	}
}

// masterTeachType is the class type a village master teaches (Mobius
// getVillageMasterTeachType).
func (s *Server) masterTeachType(n *Npc) subclass.TeachType {
	switch {
	case s.npcIs(n, "VillageMasterPriest"):
		return subclass.TeachPriest
	case s.npcIs(n, "VillageMasterMystic"):
		return subclass.TeachMystic
	}
	return subclass.TeachFighter
}
