package scripting

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// ReplyKind is how the game shows an NPC handler's answer. The rules are
// Mobius's (Quest.showResult).
type ReplyKind int

const (
	// ReplyNone: the handler returned "". It did what it needed (for
	// example npc:say) and there is nothing to show.
	ReplyNone ReplyKind = iota
	// ReplyPage: Content is a page file name ending in .htm or .html,
	// relative to Dir (the script's folder). The game layer may also look
	// the name up in its HTML folder, as Mobius does for data/ paths. The
	// name is a clean relative path (fs.ValidPath, no backslashes).
	ReplyPage
	// ReplyHTML: Content is HTML starting with <html, sent as is.
	ReplyHTML
	// ReplyText: Content is plain text. Mobius shows it as a system
	// message (player.sendMessage); the game layer decides.
	ReplyText
)

func (k ReplyKind) String() string {
	switch k {
	case ReplyNone:
		return "none"
	case ReplyPage:
		return "page"
	case ReplyHTML:
		return "html"
	case ReplyText:
		return "text"
	}
	return fmt.Sprintf("ReplyKind(%d)", int(k))
}

// Reply is a script's answer to a talk or a "Script <name> <event>" link.
type Reply struct {
	Kind    ReplyKind
	Content string
	// Vars are the values of %name% placeholders the handler returned as
	// its second value. They are already applied to HTML and text; for a
	// page file, the game layer applies them (ReplaceVars) after loading
	// it, together with its own placeholders such as %objectId%.
	Vars map[string]string
	// Script is the name of the script that answered.
	Script string
	// Dir is the folder of the file that created the script. Page names
	// are relative to it.
	Dir string
	// Err is set when the handler failed: a Lua error, a timeout or a bad
	// return value. It is already logged. Mobius shows script errors to
	// GMs only.
	Err error
}

// HasFirstTalk reports whether a script answers the first talk of the NPC
// template npcID. If not, the game shows the NPC's default dialog without
// calling FirstTalk (Mobius NpcClick).
func (e *Engine) HasFirstTalk(npcID int) bool {
	if e.closed {
		return false
	}
	owner, ok := e.vm.firstTalk[npcID]
	if !ok {
		return false
	}
	_, ok = owner.script.handlerFor(hookFirstTalk, npcID)
	return ok
}

// FirstTalk runs the s:on_first_talk handler of the NPC template npcID:
// player clicked npc, which stands within reach. ok is false when the game
// should show the NPC's default dialog: no script owns the NPC, the
// handler returned nil, or it failed (Reply.Err).
func (e *Engine) FirstTalk(ctx context.Context, npcID int, player PlayerRef, npc NpcRef) (reply Reply, ok bool) {
	if err := e.stopped(ctx); err != nil {
		return Reply{Err: err}, false
	}
	owner, found := e.vm.firstTalk[npcID]
	if !found {
		return Reply{}, false
	}
	reg, found := owner.script.handlerFor(hookFirstTalk, npcID)
	if !found {
		return Reply{}, false
	}
	defer e.settle()
	v := e.vm
	return v.npcCall(ctx, e.opts.Timeout, owner.script, reg.handler, "on_first_talk", func() []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, player)}
	})
}

// Talk runs the talk handler of script for the NPC template npcID: player
// clicked the "Script <script>" link (Mobius's quest link) at npc, or chose
// the quest in the quest window. A quest's start NPCs count as talk NPCs.
// ok is false when there is nothing to show: no such script or handler,
// the handler returned nil, or it failed (Reply.Err).
func (e *Engine) Talk(ctx context.Context, script string, npcID int, player PlayerRef, npc NpcRef) (reply Reply, ok bool) {
	if err := e.stopped(ctx); err != nil {
		return Reply{Err: err}, false
	}
	s, found := e.vm.scripts[script]
	if !found {
		return Reply{}, false
	}
	reg, found := s.handlerFor(hookTalk, npcID)
	if !found {
		return Reply{Script: s.name, Dir: s.dir}, false
	}
	defer e.settle()
	v := e.vm
	return v.npcCall(ctx, e.opts.Timeout, s, reg.handler, "on_talk", func() []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, player)}
	})
}

// TalkScripts returns the names of the scripts with a talk handler for the
// NPC template npcID (talk or start NPCs), in load order. The game layer
// lists them when a player opens the NPC's quest link without a script
// name.
func (e *Engine) TalkScripts(npcID int) []string {
	if e.closed {
		return nil
	}
	var names []string
	for _, s := range e.vm.byID[hookTalk][npcID] {
		if _, ok := s.handlerFor(hookTalk, npcID); ok {
			names = append(names, s.name)
		}
	}
	return names
}

// Event runs the s:on_event handler of script: player clicked a
// "Script <script> <event>" link, near npc. npc may be nil, and so may
// player in later uses (timers). ok is false when there is nothing to
// show: no such script or handler, the handler returned nil, or it failed
// (Reply.Err).
func (e *Engine) Event(ctx context.Context, script, event string, player PlayerRef, npc NpcRef) (reply Reply, ok bool) {
	if err := e.stopped(ctx); err != nil {
		return Reply{Err: err}, false
	}
	s, found := e.vm.scripts[script]
	if !found {
		return Reply{}, false
	}
	reg, found := s.handlerFor(hookEvent, 0)
	if !found {
		return Reply{Script: s.name, Dir: s.dir}, false
	}
	defer e.settle()
	v := e.vm
	return v.npcCall(ctx, e.opts.Timeout, s, reg.handler, "on_event", func() []lua.LValue {
		return []lua.LValue{lua.LString(event), v.newNpc(v.L, npc), v.newPlayer(v.L, player)}
	})
}

// HasScript reports whether a script with this name exists, for the game
// layer's check of "Script <name>" links.
func (e *Engine) HasScript(name string) bool {
	if e.closed {
		return false
	}
	_, ok := e.vm.scripts[name]
	return ok
}

// npcCall runs one NPC handler of s with the arguments args builds (inside
// the run, so its refs expire with it) and turns what it returned into a
// Reply.
func (v *vm) npcCall(ctx context.Context, timeout time.Duration, s *npcScript, h handler, what string, args func() []lua.LValue) (Reply, bool) {
	base := Reply{Script: s.name, Dir: s.dir}
	ctx, end := v.begin(ctx, timeout)
	defer end()
	// events.reject belongs to events; inside an NPC handler it is an
	// error, even when this run is nested in an event's run.
	prev := v.event
	v.event = nil
	defer func() { v.event = prev }()

	ret, vars, err := v.callWith(h.fn, args)
	if err != nil {
		base.Err = handlerError(ctx, err, h, s.name+" "+what+" handler", timeout)
		v.logger.Warn("script handler failed", "script", s.name, "handler", h.where(), "err", base.Err)
		return base, false
	}
	reply, ok, problem := toReply(base, ret, vars)
	if problem != "" {
		base.Err = &Error{File: h.file, Line: h.line, Msg: fmt.Sprintf("%s %s handler %s", s.name, what, problem)}
		v.logger.Warn("script handler failed", "script", s.name, "handler", h.where(), "err", base.Err)
		return base, false
	}
	return reply, ok
}

// toReply turns a handler's return values into a Reply by Mobius's rules:
// nil is no reply, "" is ReplyNone, a name ending in .htm or .html is a
// page, a string starting with <html is HTML, other strings are text. A
// table as the second value gives placeholder values; anything else there
// is ignored (string.gsub returns a count). problem describes a wrong
// return value.
func toReply(r Reply, ret, second lua.LValue) (reply Reply, ok bool, problem string) {
	if ret == lua.LNil {
		return r, false, ""
	}
	str, isString := ret.(lua.LString)
	if !isString {
		return r, false, fmt.Sprintf("returned a %s; return a page name, HTML, text or nil", ret.Type())
	}
	var vars map[string]string
	if t, isTable := second.(*lua.LTable); isTable {
		var err error
		if vars, err = varsFromLua(t); err != nil {
			return r, false, "returned bad placeholders: " + err.Error()
		}
	}
	s := string(str)
	switch {
	case s == "":
		r.Kind = ReplyNone
	case strings.HasSuffix(s, ".htm") || strings.HasSuffix(s, ".html"):
		if !fs.ValidPath(s) || strings.Contains(s, `\`) {
			return r, false, fmt.Sprintf("returned the page name %q; use a relative path without . or .. parts", s)
		}
		r.Kind, r.Content, r.Vars = ReplyPage, s, vars
	case strings.HasPrefix(s, "<html"):
		r.Kind, r.Content, r.Vars = ReplyHTML, ReplaceVars(s, vars), vars
	default:
		r.Kind, r.Content, r.Vars = ReplyText, ReplaceVars(s, vars), vars
	}
	return r, true, ""
}
