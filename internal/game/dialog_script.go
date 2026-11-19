package game

import (
	"html"
	"strconv"
	"strings"

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
		content, ok := s.readScriptPage(p, r.Dir, r.Content)
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
// ScriptLink.showQuestWindow): the scripts that talk at n are its quests.
// With none, Mobius's "no quest" page shows; with one, it answers; with
// more, the player chooses.
func (s *Server) showQuestWindow(p *Player, n *Npc) {
	var names []string
	if s.scripts != nil {
		names = s.scripts.TalkScripts(n.Template.ID)
	}
	switch len(names) {
	case 0:
		s.showScriptTalk(p, n, "")
	case 1:
		s.showScriptTalk(p, n, names[0])
	default:
		// Mobius showQuestChooseWindow.
		var b strings.Builder
		b.WriteString("<html><body>")
		for _, name := range names {
			b.WriteString(`<a action="bypass npc_` + strconv.Itoa(int(n.ID())) + `_Script ` + name + `">[` + name + `]</a><br>`)
		}
		b.WriteString("</body></html>")
		s.insertObjectIDAndShow(p, n, b.String())
	}
}

// showScriptTalk runs the talk handler of the script name at n (Mobius
// ScriptLink.showQuestWindow(player, npc, questId)). An unknown script
// shows the "no quest" page.
func (s *Server) showScriptTalk(p *Player, n *Npc, name string) {
	if name != "" && s.scripts != nil && s.scripts.HasScript(name) {
		reply, ok := s.scripts.Talk(s.background(), name, n.Template.ID, p, n.ScriptRef())
		if ok || reply.Err == nil {
			s.dialogState(p).lastQuestNpc = n.ID()
		}
		if ok {
			s.showScriptReply(p, n, reply)
		} else {
			s.showScriptError(p, reply.Err)
		}
	} else {
		s.insertObjectIDAndShow(p, n, s.noQuestPage(p))
	}
	p.SendActionFailed()
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
	if n == nil || p.Position().Distance2D(n.Position()) > float64(s.settings.NPCs.InteractionRange) {
		return nil
	}
	return n
}
