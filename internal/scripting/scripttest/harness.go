// Package scripttest runs Lua scripts in Go tests: load scripts, fire
// events, talk to scripted NPCs, and check what the handlers decided,
// said and logged.
//
//	h := scripttest.Load(t, "../../scripts", "events/character_create.lua")
//	res := h.CharacterCreate(scripting.CharacterCreate{Name: "Announcer"})
//	if !res.Rejected { ... }
//
//	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 5}
//	guard := &scripttest.Npc{ObjectID: 2, ID: 30733, Name: "Guard"}
//	reply, ok := h.FirstTalk(hero, guard)
//
// The harness calls the engine on the test's goroutine, the way the world
// loop does on the server. Do not call it from other goroutines.
package scripttest

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

// callTimeout bounds one call into the engine, on top of the engine's own
// handler timeout.
const callTimeout = 5 * time.Second

// Harness is a script engine for one test.
type Harness struct {
	t      testing.TB
	dir    string
	engine *scripting.Engine
	logs   *recorder
	out    *output
}

// Load loads the scripts in dir. With no files, every script runs, as on
// the server; otherwise only the given scripts (slash paths relative to
// dir) run, and lib/ stays available to require. The test fails if a
// script does not load. The engine is closed when the test ends.
func Load(t testing.TB, dir string, files ...string) *Harness {
	t.Helper()
	opts := scripting.Options{Dir: dir}
	if len(files) > 0 {
		opts.Include = func(path string) bool { return slices.Contains(files, path) }
	}
	return start(t, opts)
}

// LoadSource writes files (slash path relative to the scripts folder →
// Lua source) to a temporary scripts folder and loads all of them.
func LoadSource(t testing.TB, files map[string]string) *Harness {
	t.Helper()
	return start(t, scripting.Options{Dir: WriteScripts(t, files)})
}

// WriteScripts writes files (slash path → Lua source) to a new temporary
// folder and returns its path.
func WriteScripts(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		WriteScript(t, dir, name, src)
	}
	return dir
}

// WriteScript writes one script below dir, creating folders as needed.
func WriteScript(t testing.TB, dir, name, src string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}

func start(t testing.TB, opts scripting.Options) *Harness {
	t.Helper()
	h := &Harness{t: t, dir: opts.Dir, logs: newRecorder(t), out: &output{}}
	opts.Logger = slog.New(h.logs)
	opts.Output = h.out
	e, err := scripting.New(t.Context(), opts)
	if err != nil {
		t.Fatalf("loading scripts: %v", err)
	}
	t.Cleanup(e.Close)
	h.engine = e
	return h
}

// Engine returns the engine, for calls the harness has no helper for.
func (h *Harness) Engine() *scripting.Engine { return h.engine }

// Dir returns the scripts folder.
func (h *Harness) Dir() string { return h.dir }

// Reload loads the scripts again and swaps them in, as the server does
// after a change. On a load error the old scripts stay and the error is
// returned.
func (h *Harness) Reload() error {
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	p, err := h.engine.Prepare(ctx)
	if err != nil {
		return err
	}
	h.engine.Swap(p)
	return nil
}

// Fire runs the handlers of ev and returns the result. The test fails if
// a handler fails or the event cannot run.
func (h *Harness) Fire(ev scripting.Event) scripting.Result {
	h.t.Helper()
	res := h.FireAllowErrors(ev)
	if res.Err != nil {
		h.t.Fatalf("%s handler failed: %v", ev.Name, res.Err)
	}
	return res
}

// FireAllowErrors is Fire for tests that expect handler failures: those
// are returned in Result.Err instead of failing the test.
func (h *Harness) FireAllowErrors(ev scripting.Event) scripting.Result {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return h.engine.Invoke(ctx, ev)
}

// CharacterCreate fires events.on_character_create.
func (h *Harness) CharacterCreate(ev scripting.CharacterCreate) scripting.Result {
	h.t.Helper()
	return h.Fire(ev.Event())
}

// Chat fires events.on_chat. Use Player.Ref for msg.Player.
func (h *Harness) Chat(msg scripting.ChatMessage) scripting.Result {
	h.t.Helper()
	return h.Fire(msg.Event())
}

// FirstTalk runs the first-talk handler of npc's template, as when player
// clicks npc. The test fails if the handler fails.
func (h *Harness) FirstTalk(player *Player, npc *Npc) (scripting.Reply, bool) {
	h.t.Helper()
	return h.npcCall("first talk", func(ctx context.Context) (scripting.Reply, bool) {
		return h.engine.FirstTalk(ctx, npc.ID, player.Ref(), npc.Ref())
	})
}

// Talk runs the on_talk handler script registered for npc's template, as
// when player clicks "Script <script>" at npc. The test fails if the
// handler fails.
func (h *Harness) Talk(script string, player *Player, npc *Npc) (scripting.Reply, bool) {
	h.t.Helper()
	return h.npcCall("talk", func(ctx context.Context) (scripting.Reply, bool) {
		return h.engine.Talk(ctx, script, npc.ID, player.Ref(), npc.Ref())
	})
}

// Event runs the on_event handler of script, as when player clicks
// "Script <script> <event>" near npc (which may be nil). The test fails if
// the handler fails.
func (h *Harness) Event(script, event string, player *Player, npc *Npc) (scripting.Reply, bool) {
	h.t.Helper()
	return h.npcCall("event "+event, func(ctx context.Context) (scripting.Reply, bool) {
		return h.engine.Event(ctx, script, event, player.Ref(), npc.Ref())
	})
}

func (h *Harness) npcCall(what string, call func(context.Context) (scripting.Reply, bool)) (scripting.Reply, bool) {
	h.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	reply, ok := call(ctx)
	if reply.Err != nil {
		h.t.Fatalf("%s handler failed: %v", what, reply.Err)
	}
	return reply, ok
}

// Player is a fake player. Change its fields between calls as needed.
type Player struct {
	ObjectID int32
	Name     string
	Level    int
	ClassID  int
	Race     int
}

// Ref returns the player as the engine sees it; nil for a nil Player. It
// reads the fields at call time.
func (p *Player) Ref() scripting.PlayerRef {
	if p == nil {
		return nil
	}
	return playerRef{p}
}

type playerRef struct{ p *Player }

func (r playerRef) ObjectID() int32 { return r.p.ObjectID }
func (r playerRef) Name() string    { return r.p.Name }
func (r playerRef) Level() int      { return r.p.Level }
func (r playerRef) ClassID() int    { return r.p.ClassID }
func (r playerRef) Race() int       { return r.p.Race }

// Npc is a fake NPC. ID is its template ID.
type Npc struct {
	ObjectID int32
	ID       int
	Name     string
}

// Ref returns the NPC as the engine sees it; nil for a nil Npc.
func (n *Npc) Ref() scripting.NpcRef {
	if n == nil {
		return nil
	}
	return npcRef{n}
}

type npcRef struct{ n *Npc }

func (r npcRef) ObjectID() int32 { return r.n.ObjectID }
func (r npcRef) ID() int         { return r.n.ID }
func (r npcRef) Name() string    { return r.n.Name }

// Speech is one npc:say.
type Speech struct {
	NpcID    int   // template ID
	ObjectID int32 // the NPC's object ID
	Text     string
}

// Said returns what NPCs said so far, in order.
func (h *Harness) Said() []Speech {
	h.out.mu.Lock()
	defer h.out.mu.Unlock()
	return slices.Clone(h.out.said)
}

// ClearSaid forgets what NPCs said so far.
func (h *Harness) ClearSaid() {
	h.out.mu.Lock()
	defer h.out.mu.Unlock()
	h.out.said = nil
}

// output is the harness's scripting.Output. It is safe for concurrent use,
// for tests that drive Engine() from a server's goroutines.
type output struct {
	mu   sync.Mutex
	said []Speech
}

func (o *output) NpcSay(npc scripting.NpcRef, text string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.said = append(o.said, Speech{NpcID: npc.ID(), ObjectID: npc.ObjectID(), Text: text})
}

// Log is one record written by the engine or a script.
type Log struct {
	Level slog.Level
	Msg   string
	Attrs map[string]any
}

// Logs returns the records logged so far.
func (h *Harness) Logs() []Log { return h.logs.all() }

// ClearLogs forgets the records logged so far.
func (h *Harness) ClearLogs() { h.logs.clear() }

// recorder is a slog.Handler that keeps every record and copies it to the
// test log. Handlers made by WithAttrs share the records. It is safe for
// concurrent use (the deprecated Engine.Call logs from server goroutines).
type recorder struct {
	t     testing.TB
	attrs []slog.Attr

	mu   *sync.Mutex
	logs *[]Log
}

func newRecorder(t testing.TB) *recorder {
	return &recorder{t: t, mu: new(sync.Mutex), logs: new([]Log)}
}

func (r *recorder) Enabled(context.Context, slog.Level) bool { return true }

func (r *recorder) Handle(_ context.Context, rec slog.Record) error {
	l := Log{Level: rec.Level, Msg: rec.Message, Attrs: make(map[string]any)}
	for _, a := range r.attrs {
		l.Attrs[a.Key] = a.Value.Any()
	}
	rec.Attrs(func(a slog.Attr) bool {
		l.Attrs[a.Key] = a.Value.Any()
		return true
	})
	r.t.Logf("%s %s %v", l.Level, l.Msg, l.Attrs)
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.logs = append(*r.logs, l)
	return nil
}

func (r *recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	c := *r
	c.attrs = append(slices.Clip(r.attrs), attrs...)
	return &c
}

// WithGroup is not used by the engine; groups are flattened.
func (r *recorder) WithGroup(string) slog.Handler { return r }

func (r *recorder) all() []Log {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(*r.logs)
}

func (r *recorder) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.logs = nil
}
