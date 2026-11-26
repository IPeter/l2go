package game

import (
	"html"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/quest"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The Lua NPC hooks (decision N9): first talk, the quest link ("Script"),
// and the "Script <name> <event>" links, with Mobius's reply rules
// (Quest.showResult).

// noQuestHTML is Mobius's text when data/html/noquest.htm is missing or
// empty (Quest.DEFAULT_NO_QUEST_MSG).
const noQuestHTML = "<html><body>You are either not on a quest that involves this NPC, or you don't meet this NPC's minimum quest requirements.</body></html>"

// scriptFirstTalk runs the first-talk script of n for p. A reply shows; no
// reply shows the NPC's own first page. A failed script shows a GM the
// error instead (Mobius showError); players get the NPC's own page.
func (s *Server) scriptFirstTalk(p *Player, n *Npc) {
	reply, ok := s.scripts.FirstTalk(s.background(), n.Template.ID, p, n.ScriptRef())
	switch {
	case ok:
		s.showScriptReply(p, n, reply)
	case reply.Err != nil && p.IsGM():
		s.showScriptError(p, reply.Err)
	default:
		s.showFirstPage(p, n)
	}
}

// showScriptReply shows what a script handler returned, by Mobius
// showResult: a page file (from the script's folder, then the HTML folder)
// with the handler's placeholders, %objectId% and %playername% filled, or
// inline HTML with %playername% filled, each followed by ActionFailed; text
// goes to the chat as a system message. n may be nil.
func (s *Server) showScriptReply(p *Player, n *Npc, r scripting.Reply) {
	var npc world.ObjectID
	if n != nil {
		npc = n.ID()
	}
	switch r.Kind {
	case scripting.ReplyPage:
		content, ok := s.readScriptPage(p, r.Dir, s.scriptRelDir(r.Script), r.Content)
		if !ok {
			// Mobius showHtmlFile sends nothing for a missing page.
			s.log.Warn("game: script page not found", append(p.logArgs(), "script", r.Script, "page", r.Content)...)
			return
		}
		content = scripting.ReplaceVars(content, r.Vars)
		if n != nil {
			content = strings.ReplaceAll(content, "%objectId%", strconv.Itoa(int(npc)))
		}
		s.sendPage(p, newPage(npc, content).replace("%playername%", p.Name()))
		p.SendActionFailed()
	case scripting.ReplyHTML:
		s.sendPage(p, newPage(npc, r.Content).replace("%playername%", p.Name()))
		p.SendActionFailed()
	case scripting.ReplyText:
		p.SendMessage(r.Content)
	}
}

// showScriptError shows a GM why a script failed (Mobius Quest.showError
// shows GMs the stack trace). Players see nothing; the error is logged by
// the engine.
func (s *Server) showScriptError(p *Player, err error) {
	if !p.IsGM() || err == nil {
		return
	}
	page := "<html><body><title>Script error</title>" + html.EscapeString(err.Error()) + "</body></html>"
	s.sendPage(p, newPage(0, page).replace("%playername%", p.Name()))
	p.SendActionFailed()
}

// bypassTrim trims s like the Java String.trim: every character up to U+0020
// at both ends.
func bypassTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
}

// bypassScript handles the "Script" bypass (Mobius ScriptLink): "Script"
// opens n's quest window, "Script <name>" talks to the script, and
// "Script <name> <event>" sends the event to the script.
func (s *Server) bypassScript(p *Player, n *Npc, command string) {
	var rest string
	if len(command) > len("Script") {
		rest = bypassTrim(command[len("Script"):])
	}
	name, event, hasEvent := strings.Cut(rest, " ")
	switch {
	case rest == "":
		if n != nil {
			s.showQuestWindow(p, n)
		}
	case !hasEvent:
		if n != nil {
			s.showScriptTalk(p, n, name)
		}
	default:
		s.scriptEvent(p, name, bypassTrim(event))
	}
}

// showQuestWindow opens n's quest window (Mobius
// ScriptLink.showQuestWindow(player, npc)): the quests (IDs 1 to 19999)
// that talk at n and of which p has a state, and those n starts, when what
// they require exists (q:requires). With none, Mobius's "no quest" page
// shows; with one, it answers; with more, the player chooses, the quests
// by ID with "(In Progress)" and "(Done)".
func (s *Server) showQuestWindow(p *Player, n *Npc) {
	var options []scripting.QuestInfo
	if s.scripts != nil {
		seen := make(map[string]bool)
		consider := func(name string, needState bool) {
			if seen[name] {
				return
			}
			q, ok := s.scripts.Quest(name)
			if !ok || q.ID < 1 || q.ID > quest.MaxListID || !questAvailable(q) {
				return
			}
			if needState && s.QuestState(p, name) == nil {
				return
			}
			seen[name] = true
			options = append(options, q)
		}
		for _, name := range s.scripts.TalkScripts(n.Template.ID) {
			consider(name, true)
		}
		for _, name := range s.scripts.StartQuests(n.Template.ID) {
			consider(name, false)
		}
	}
	slices.SortFunc(options, func(a, b scripting.QuestInfo) int { return a.ID - b.ID })
	switch len(options) {
	case 0:
		s.showScriptTalk(p, n, "")
	case 1:
		s.showScriptTalk(p, n, options[0].Name)
	default:
		s.showQuestChooseWindow(p, n, options)
	}
}

// questAvailable reports whether the features a quest requires exist
// (decision Q10).
func questAvailable(q scripting.QuestInfo) bool {
	for _, f := range q.Requires {
		if !questFeatures[f] {
			return false
		}
	}
	return true
}

// questFeatures are the q:requires features the game has: clans since M9
// (Plan 12 Phase 7), the Four Sepulchers since M10 (Plan 13 Phase 10.7,
// scripts/ai/areas/ImperialTomb/FourSepulchers).
var questFeatures = map[string]bool{"clans": true, "sepulchers": true}

// showQuestChooseWindow lets p choose one of quests (Mobius
// showQuestChooseWindow).
func (s *Server) showQuestChooseWindow(p *Player, n *Npc, quests []scripting.QuestInfo) {
	var b strings.Builder
	b.WriteString("<html><body>")
	for _, q := range quests {
		state := ""
		if st := s.QuestState(p, q.Name); st != nil {
			switch {
			case st.IsStarted():
				state = " (In Progress)"
			case st.IsCompleted():
				state = " (Done)"
			}
		}
		label := q.Description
		if q.Custom {
			label = q.Name
		}
		b.WriteString(`<a action="bypass npc_` + strconv.Itoa(int(n.ID())) + `_Script ` + q.Name + `">[` + label + state + `]</a><br>`)
	}
	b.WriteString("</body></html>")
	s.insertObjectIDAndShow(p, n, b.String())
}

// showScriptTalk runs the talk handler of the script name at n (Mobius
// ScriptLink.showQuestWindow(player, npc, questId)). A quest (IDs 1 to
// 19999) is refused with 80% weight (weight penalty 3) or a 90% full
// inventory, and a quest not yet taken with more than 25 active quests
// (fullquest.html). An unknown script shows the "no quest" page.
func (s *Server) showScriptTalk(p *Player, n *Npc, name string) {
	var q scripting.QuestInfo
	found := name != "" && s.scripts != nil
	if found {
		q, found = s.scripts.Quest(name)
	}
	if !found {
		s.insertObjectIDAndShow(p, n, s.noQuestPage(p))
		p.SendActionFailed()
		return
	}
	listed := q.ID >= 1 && q.ID <= quest.MaxListID
	if listed && (p.WeightPenalty() >= 3 || !s.inventoryUnder90(p)) {
		p.SendSystemMessage(sysMsgQuestInventoryFull)
		return
	}
	if listed && s.QuestState(p, name) == nil && len(s.questsOf(p).book.Active()) > questMaxActive {
		s.sendPage(p, s.pageFile(p, n.ID(), "fullquest.html"))
		return
	}
	reply, ok := s.scripts.Talk(s.background(), name, n.Template.ID, p, n.ScriptRef())
	switch {
	case ok:
		s.dialogState(p).lastQuestNpc = n.ID()
		s.showScriptReply(p, n, reply)
	case reply.Err != nil:
		s.showScriptError(p, reply.Err)
	default:
		s.dialogState(p).lastQuestNpc = n.ID()
	}
	p.SendActionFailed()
}

// inventoryUnder90 is Mobius isInventoryUnder90(true): the items, quest
// items included, fill at most 90% of the inventory's slots.
func (s *Server) inventoryUnder90(p *Player) bool {
	if p.inv.Inventory == nil {
		return true
	}
	return float64(p.inv.Len()) <= float64(p.stats.View().InventoryLimit)*0.9
}

// scriptRelDir returns the folder of a script relative to the scripts
// folder, "" if there is no such script.
func (s *Server) scriptRelDir(name string) string {
	if s.scripts == nil || name == "" {
		return ""
	}
	if q, ok := s.scripts.Quest(name); ok {
		return q.RelDir
	}
	return ""
}

// noQuestPage returns data/html/noquest.htm, or Mobius's text when it is
// missing (Quest.getNoQuestMsg).
func (s *Server) noQuestPage(p *Player) string {
	if content, ok := s.htmlFile(p, "noquest.htm"); ok && content != "" {
		return content
	}
	return noQuestHTML
}

// scriptEvent sends event to the script name, at the NPC p talked to last
// if it is within interaction range (2D), else at the NPC of its last
// script talk (Mobius Player.processScriptEvent). Without such an NPC, or
// without the script, nothing happens.
func (s *Server) scriptEvent(p *Player, name, event string) {
	if s.scripts == nil || event == "" || !s.scripts.HasScript(name) {
		return
	}
	n := s.npcInRange2D(p, p.LastNpc)
	if n == nil {
		n = s.npcInRange2D(p, s.dialogState(p).lastQuestNpc)
	}
	if n == nil {
		return
	}
	reply, ok := s.scripts.Event(s.background(), name, event, p, n.ScriptRef())
	if ok {
		s.showScriptReply(p, n, reply)
		return
	}
	s.showScriptError(p, reply.Err)
}

// npcInRange2D returns the NPC with object ID id when it is in the world
// within interaction range of p on the X/Y plane, or nil.
func (s *Server) npcInRange2D(p *Player, id world.ObjectID) *Npc {
	if id == 0 {
		return nil
	}
	n := s.Npc(id)
	if n == nil || p.Position().Distance2D(n.Position()) > float64(s.npcInteractionRange()) {
		return nil
	}
	return n
}
