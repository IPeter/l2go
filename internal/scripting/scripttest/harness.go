// Package scripttest runs Lua scripts in Go tests: load scripts, fire
// events, and check what the handlers decided and logged.
//
//	h := scripttest.Load(t, "../../scripts", "events/character_create.lua")
//	res := h.CharacterCreate(scripting.CharacterCreate{Name: "Announcer"})
//	if !res.Rejected { ... }
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

// callTimeout bounds one Fire, so a stuck engine fails the test instead of
// hanging it.
const callTimeout = 5 * time.Second

// Harness is a running script engine for one test.
type Harness struct {
	t      testing.TB
	dir    string
	engine *scripting.Engine
	logs   *recorder
}

// Load loads the scripts in dir. With no files, every script runs, as on
// the server; otherwise only the given scripts (slash paths relative to
// dir) run, and lib/ stays available to require. The test fails if a
// script does not load. The engine stops when the test ends.
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
	rec := newRecorder(t)
	opts.Logger = slog.New(rec)
	e, err := scripting.New(t.Context(), opts)
	if err != nil {
		t.Fatalf("loading scripts: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = e.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return &Harness{t: t, dir: opts.Dir, engine: e, logs: rec}
}

// Engine returns the engine, for example to Reload it.
func (h *Harness) Engine() *scripting.Engine { return h.engine }

// Dir returns the scripts folder.
func (h *Harness) Dir() string { return h.dir }

// Fire runs the handlers of ev and returns the result. The test fails if
// the event cannot run or a handler fails.
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
	res, err := h.engine.Call(ctx, ev)
	if err != nil {
		h.t.Fatalf("%s: %v", ev.Name, err)
	}
	return res
}

// CharacterCreate fires events.on_character_create.
func (h *Harness) CharacterCreate(ev scripting.CharacterCreate) scripting.Result {
	h.t.Helper()
	return h.Fire(ev.Event())
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
// test log. Handlers made by WithAttrs share the records.
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
