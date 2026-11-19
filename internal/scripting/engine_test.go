package scripting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// writeScripts writes files (slash path → source) to a new temporary
// scripts folder. (scripttest has the same helper, but importing it here
// would be an import cycle.)
func writeScripts(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		writeScript(t, dir, name, src)
	}
	return dir
}

func writeScript(t testing.TB, dir, name, src string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}

// start runs e until the test ends.
func start(t testing.TB, e *Engine) {
	t.Helper()
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
}

// newEngine loads files with opts (Dir is set to a temporary folder when
// empty) and runs the engine until the test ends.
func newEngine(t testing.TB, files map[string]string, opts Options) *Engine {
	t.Helper()
	if opts.Dir == "" {
		opts.Dir = writeScripts(t, files)
	}
	e, err := New(context.Background(), opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	start(t, e)
	return e
}

// loadErr loads files and returns the load error, failing if there is none.
func loadErr(t *testing.T, files map[string]string, opts Options) *LoadError {
	t.Helper()
	opts.Dir = writeScripts(t, files)
	e, err := New(context.Background(), opts)
	if err == nil {
		start(t, e)
		t.Fatal("New succeeded, want a load error")
	}
	le, ok := errors.AsType[*LoadError](err)
	if !ok {
		t.Fatalf("New error %v (%T) is not a *LoadError", err, err)
	}
	return le
}

func createEvent(name string) Event {
	return CharacterCreate{Account: "acc", Name: name}.Event()
}

func call(t testing.TB, e *Engine, name string) Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := e.Call(ctx, createEvent(name))
	if err != nil {
		t.Fatalf("Call(%q): %v", name, err)
	}
	return res
}

// rejectName is a script that rejects one character name.
func rejectName(name, reason string) string {
	return fmt.Sprintf(`events.on_character_create(function(ev)
  if ev.name == %q then
    return events.reject(%q)
  end
end)
`, name, reason)
}

func wantRejected(t *testing.T, res Result, reason string) {
	t.Helper()
	if res.Err != nil {
		t.Errorf("handler error: %v", res.Err)
	}
	if !res.Rejected || res.Reason != reason {
		t.Errorf("result = %+v, want rejected with %q", res, reason)
	}
}

func wantAccepted(t *testing.T, res Result) {
	t.Helper()
	if res.Err != nil {
		t.Errorf("handler error: %v", res.Err)
	}
	if res.Rejected {
		t.Errorf("result = %+v, want accepted", res)
	}
}

// logBuffer collects log output from several goroutines.
type logBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *logBuffer) logger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(l, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// records decodes the JSON log lines written so far.
func (l *logBuffer) records(t testing.TB) []map[string]any {
	t.Helper()
	l.mu.Lock()
	data := bytes.Clone(l.b.Bytes())
	l.mu.Unlock()
	var out []map[string]any
	for line := range bytes.Lines(data) {
		var rec map[string]any
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

// find returns the first record with msg.
func (l *logBuffer) find(t testing.TB, msg string) (map[string]any, bool) {
	t.Helper()
	for _, rec := range l.records(t) {
		if rec["msg"] == msg {
			return rec, true
		}
	}
	return nil, false
}

func waitFor(t testing.TB, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// --- Loading ---

func TestLoadValidScript(t *testing.T) {
	e := newEngine(t, map[string]string{
		"events/names.lua": rejectName("Bad", ReasonIncorrectName),
	}, Options{})
	wantRejected(t, call(t, e, "Bad"), ReasonIncorrectName)
	wantAccepted(t, call(t, e, "Good"))
}

func TestLoadEmptyFolder(t *testing.T) {
	e := newEngine(t, nil, Options{})
	wantAccepted(t, call(t, e, "Anyone"))
}

func TestLoadMissingFolder(t *testing.T) {
	_, err := New(context.Background(), Options{Dir: filepath.Join(t.TempDir(), "missing")})
	if err == nil {
		t.Fatal("New succeeded for a missing folder")
	}
}

func TestLoadSyntaxError(t *testing.T) {
	le := loadErr(t, map[string]string{
		"bad.lua": "local x = 1\nlocal y = = 2\n",
	}, Options{})
	if len(le.Errs) != 1 {
		t.Fatalf("errors = %v, want 1", le.Errs)
	}
	e, ok := errors.AsType[*Error](le.Errs[0])
	if !ok {
		t.Fatalf("error %v is not an *Error", le.Errs[0])
	}
	if !strings.HasSuffix(e.File, "bad.lua") || e.Line != 2 {
		t.Errorf("location = %s:%d, want bad.lua:2", e.File, e.Line)
	}
	if !strings.Contains(le.Error(), "bad.lua:2:") {
		t.Errorf("message %q has no file:line", le.Error())
	}
}

func TestLoadSyntaxErrorAtEOF(t *testing.T) {
	le := loadErr(t, map[string]string{
		"open.lua": "events.on_character_create(function(ev)\n  local x = 1\n",
	}, Options{})
	e, ok := errors.AsType[*Error](le.Errs[0])
	if !ok || !strings.HasSuffix(e.File, "open.lua") || e.Line == 0 {
		t.Errorf("error = %v, want open.lua with a line", le.Errs[0])
	}
}

func TestLoadRuntimeError(t *testing.T) {
	le := loadErr(t, map[string]string{
		"events/boom.lua": "local t = nil\nlocal x = t.field\n",
	}, Options{})
	e, ok := errors.AsType[*Error](le.Errs[0])
	if !ok {
		t.Fatalf("error %v is not an *Error", le.Errs[0])
	}
	if want := filepath.Join("events", "boom.lua"); !strings.HasSuffix(e.File, want) || e.Line != 2 {
		t.Errorf("location = %s:%d, want %s:2", e.File, e.Line, want)
	}
	if !strings.Contains(le.Error(), "boom.lua:2:") {
		t.Errorf("message %q has no file:line", le.Error())
	}
}

func TestLoadErrorCallsError(t *testing.T) {
	le := loadErr(t, map[string]string{"e.lua": "\n\nerror('custom failure')\n"}, Options{})
	e, _ := errors.AsType[*Error](le.Errs[0])
	if e == nil || e.Line != 3 || !strings.Contains(e.Msg, "custom failure") {
		t.Errorf("error = %v, want e.lua:3: custom failure", le.Errs[0])
	}
}

func TestLoadReportsEveryError(t *testing.T) {
	le := loadErr(t, map[string]string{
		"a.lua":     "this is not lua",
		"b.lua":     "local x = nil + 1",
		"good.lua":  rejectName("x", ReasonNameExists),
		"lib/l.lua": "return {",
	}, Options{})
	if len(le.Errs) != 3 {
		t.Fatalf("got %d errors, want 3: %v", len(le.Errs), le)
	}
	for _, name := range []string{"a.lua", "b.lua", filepath.Join("lib", "l.lua")} {
		if !strings.Contains(le.Error(), name) {
			t.Errorf("error does not mention %s:\n%v", name, le)
		}
	}
}

func TestLoadTimeout(t *testing.T) {
	begin := time.Now()
	le := loadErr(t, map[string]string{"loop.lua": "while true do end"}, Options{LoadTimeout: 50 * time.Millisecond})
	if !errors.Is(le, ErrTimeout) {
		t.Errorf("error %v does not wrap ErrTimeout", le)
	}
	if d := time.Since(begin); d > 3*time.Second {
		t.Errorf("load took %v", d)
	}
}

func TestLoadOrderAndSkippedFiles(t *testing.T) {
	logs := &logBuffer{}
	newEngine(t, map[string]string{
		"b.lua":         `log.info("b")`,
		"a.lua":         `log.info("a")`,
		"sub/c.lua":     `log.info("c")`,
		"lib/l.lua":     `log.info("lib ran")`,
		".hidden.lua":   `error("hidden file ran")`,
		".git/x.lua":    `error("hidden folder ran")`,
		"notes.txt":     `not lua`,
		"types/api.lua": "---@meta\nerror('types ran')",
	}, Options{Logger: logs.logger()})
	var got []string
	for _, rec := range logs.records(t) {
		if s, ok := rec["script"].(string); ok && s != "" {
			got = append(got, rec["msg"].(string))
		}
	}
	if want := []string{"a", "b", "c"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("scripts ran in order %v, want %v", got, want)
	}
}

func TestInclude(t *testing.T) {
	e := newEngine(t, map[string]string{
		"events/one.lua": rejectName("One", ReasonNameExists),
		"events/two.lua": rejectName("Two", ReasonNameExists),
		"broken.lua":     "not lua at all",
	}, Options{Include: func(path string) bool { return path == "events/one.lua" }})
	wantRejected(t, call(t, e, "One"), ReasonNameExists)
	wantAccepted(t, call(t, e, "Two"))
}

func TestCheck(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"a.lua":         rejectName("x", ReasonNameExists),
		"lib/l.lua":     "return {}",
		"types/api.lua": "---@meta\nfunction log.info(msg) end",
	})
	n, err := Check(context.Background(), Options{Dir: dir})
	if err != nil || n != 3 {
		t.Errorf("Check = %d, %v; want 3 files, nil", n, err)
	}

	writeScript(t, dir, "b.lua", "x(")
	if _, err := Check(context.Background(), Options{Dir: dir}); err == nil || !strings.Contains(err.Error(), "b.lua:1:") {
		t.Errorf("Check error = %v, want b.lua:1", err)
	}
}

// --- Events ---

func TestEventTable(t *testing.T) {
	e := newEngine(t, map[string]string{"check.lua": `
events.on_character_create(function(ev)
  assert(ev.account == "acc1", "account")
  assert(ev.name == "Hero", "name")
  assert(ev.race == 1 and ev.sex == 1 and ev.class_id == 18, "race, sex, class")
  assert(ev.hair_style == 2 and ev.hair_color == 3 and ev.face == 1, "appearance")
  return events.reject("name_exists")
end)
`}, Options{})
	res, err := e.Call(context.Background(), CharacterCreate{
		Account: "acc1", Name: "Hero", Race: 1, Sex: 1, ClassID: 18, HairStyle: 2, HairColor: 3, Face: 1,
	}.Event())
	if err != nil {
		t.Fatal(err)
	}
	wantRejected(t, res, ReasonNameExists)
}

func TestFirstRejectionWins(t *testing.T) {
	logs := &logBuffer{}
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_character_create(function(ev) return events.reject("name_exists") end)`,
		"b.lua": `events.on_character_create(function(ev) log.info("b ran"); return events.reject("incorrect_name") end)`,
	}, Options{Logger: logs.logger()})
	wantRejected(t, call(t, e, "x"), ReasonNameExists)
	if _, ok := logs.find(t, "b ran"); ok {
		t.Error("the handler after a rejection ran")
	}
}

func TestRejectWithoutReturn(t *testing.T) {
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_character_create(function(ev) events.reject("name_exists") end)`,
	}, Options{})
	wantRejected(t, call(t, e, "x"), ReasonNameExists)
}

func TestRejectUnknownReason(t *testing.T) {
	e := newEngine(t, map[string]string{"typo.lua": `
events.on_character_create(function(ev)
  return events.reject("incorect_name")
end)
`}, Options{})
	res := call(t, e, "x")
	if res.Rejected {
		t.Errorf("rejected with an unknown reason: %+v", res)
	}
	if res.Err == nil || !strings.Contains(res.Err.Error(), "typo.lua:3:") || !strings.Contains(res.Err.Error(), "unknown reason") {
		t.Errorf("Err = %v, want typo.lua:3 unknown reason", res.Err)
	}
}

func TestRejectOutsideHandler(t *testing.T) {
	le := loadErr(t, map[string]string{"top.lua": `events.reject("incorrect_name")`}, Options{})
	if !strings.Contains(le.Error(), "outside an event handler") {
		t.Errorf("error = %v", le)
	}
}

func TestRegisterOnlyWhileLoading(t *testing.T) {
	e := newEngine(t, map[string]string{"late.lua": `
events.on_character_create(function(ev)
  events.on_character_create(function() end)
end)
`}, Options{})
	res := call(t, e, "x")
	if res.Err == nil || !strings.Contains(res.Err.Error(), "while the scripts load") {
		t.Errorf("Err = %v, want a registration error", res.Err)
	}
}

func TestHandlerErrorDoesNotStopOthers(t *testing.T) {
	logs := &logBuffer{}
	e := newEngine(t, map[string]string{
		"a.lua": "events.on_character_create(function(ev)\n  local x = nil .. 'y'\nend)",
		"b.lua": rejectName("x", ReasonIncorrectName),
	}, Options{Logger: logs.logger()})
	res := call(t, e, "x")
	if !res.Rejected || res.Reason != ReasonIncorrectName {
		t.Errorf("result = %+v, want the second handler's rejection", res)
	}
	he, ok := errors.AsType[*Error](res.Err)
	if !ok || !strings.HasSuffix(he.File, "a.lua") || he.Line != 2 {
		t.Errorf("Err = %v, want a.lua:2", res.Err)
	}
	if _, ok := logs.find(t, "script handler failed"); !ok {
		t.Error("the handler error was not logged")
	}
}

func TestHandlerBadReturnValue(t *testing.T) {
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_character_create(function(ev) return "incorrect_name" end)`,
	}, Options{})
	res := call(t, e, "x")
	if res.Rejected || res.Err == nil || !strings.Contains(res.Err.Error(), "returned a string") {
		t.Errorf("result = %+v, want a 'returned a string' error", res)
	}
}

func TestUnknownEvent(t *testing.T) {
	e := newEngine(t, nil, Options{})
	res, err := e.Call(context.Background(), Event{Name: "no_such_event"})
	if err != nil || res.Err == nil {
		t.Errorf("Call = %+v, %v; want Result.Err", res, err)
	}
}

func TestDispatch(t *testing.T) {
	logs := &logBuffer{}
	e := newEngine(t, map[string]string{
		"a.lua": `events.on_character_create(function(ev) log.info("dispatched", "name", ev.name) end)`,
	}, Options{Logger: logs.logger()})
	if !e.Dispatch(createEvent("Async")) {
		t.Fatal("Dispatch = false")
	}
	waitFor(t, "the dispatched handler", func() bool {
		rec, ok := logs.find(t, "dispatched")
		return ok && rec["name"] == "Async"
	})
}

// --- Timeouts ---

func TestHandlerTimeout(t *testing.T) {
	const timeout = 20 * time.Millisecond
	loops := map[string]string{
		"plain":  "while true do end",
		"pcall":  "while true do pcall(function() while true do end end) end",
		"repeat": "local i = 0\nrepeat i = i + 1 until false",
	}
	for name, loop := range loops {
		t.Run(name, func(t *testing.T) {
			e := newEngine(t, map[string]string{"loop.lua": `
events.on_character_create(function(ev)
  if ev.name == "loop" then
    ` + loop + `
  end
  if ev.name == "bad" then
    return events.reject("incorrect_name")
  end
end)
`}, Options{Timeout: timeout})

			begin := time.Now()
			res := call(t, e, "loop")
			// Generous: -race and busy CI machines are slow.
			if d := time.Since(begin); d > timeout+time.Second {
				t.Errorf("an endless loop blocked the engine for %v (timeout %v)", d, timeout)
			}
			if !errors.Is(res.Err, ErrTimeout) {
				t.Errorf("Err = %v, want ErrTimeout", res.Err)
			}
			if he, ok := errors.AsType[*Error](res.Err); !ok || !strings.HasSuffix(he.File, "loop.lua") || he.Line == 0 {
				t.Errorf("Err = %v, want loop.lua with a line", res.Err)
			}
			if res.Rejected {
				t.Errorf("result = %+v, want accepted", res)
			}

			// The engine keeps working.
			wantRejected(t, call(t, e, "bad"), ReasonIncorrectName)
			wantAccepted(t, call(t, e, "good"))
		})
	}
}

func TestCallContextCanceled(t *testing.T) {
	e := newEngine(t, map[string]string{"a.lua": rejectName("x", ReasonNameExists)}, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.Call(ctx, createEvent("x")); !errors.Is(err, context.Canceled) {
		t.Errorf("Call with a canceled context = %v, want context.Canceled", err)
	}
	wantRejected(t, call(t, e, "x"), ReasonNameExists)
}

func TestStopped(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.lua": rejectName("x", ReasonNameExists)})
	e, err := New(context.Background(), Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.Run(ctx) }()
	wantRejected(t, call(t, e, "x"), ReasonNameExists)
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run = %v", err)
	}

	if _, err := e.Call(context.Background(), createEvent("x")); !errors.Is(err, ErrStopped) {
		t.Errorf("Call after stop = %v, want ErrStopped", err)
	}
	if e.Dispatch(createEvent("x")) {
		t.Error("Dispatch after stop = true")
	}
	if err := e.Reload(context.Background()); !errors.Is(err, ErrStopped) {
		t.Errorf("Reload after stop = %v, want ErrStopped", err)
	}
	if err := e.Run(context.Background()); err == nil {
		t.Error("second Run succeeded")
	}
}

// --- Sandbox ---

func TestSandboxForbidden(t *testing.T) {
	tests := map[string]string{
		"os.execute":  `os.execute("echo hacked")`,
		"os.exit":     `os.exit(1)`,
		"io.open":     `io.open("/etc/passwd")`,
		"io.write":    `io.write("x")`,
		"debug":       `debug.getinfo(1)`,
		"dofile":      `dofile("x.lua")`,
		"loadfile":    `loadfile("x.lua")`,
		"load":        `load(function() return nil end)`,
		"loadstring":  `loadstring("return 1")`,
		"module":      `module("x")`,
		"package":     `package.path = "x"`,
		"require os":  `require("os")`,
		"coroutine":   `coroutine.create(function() end)`,
		"channel":     `channel.make()`,
		"gc":          `collectgarbage()`,
		"string.dump": `string.dump(print)`,
		"huge rep":    `string.rep("x", 1e9)`,
		"huge rep2":   `local s = ("abc"):rep(1e6)`,
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			_ = loadErr(t, map[string]string{"evil.lua": src}, Options{})
		})
	}
}

func TestSandboxAllowed(t *testing.T) {
	newEngine(t, map[string]string{"ok.lua": `
assert(string.format("%d-%s", 1, "a") == "1-a")
assert(("Abc"):lower() == "abc")
assert(string.rep("ab", 3) == "ababab" and ("x"):rep(0) == "")
local t = {3, 1, 2}
table.sort(t)
assert(table.concat(t, ",") == "1,2,3")
assert(math.floor(2.5) == 2 and math.max(1, 5) == 5)
assert(select("#", 1, 2) == 2 and tostring(1) == "1" and tonumber("2") == 2)
assert(pcall(error, "x") == false)
local mt = setmetatable({}, {__index = function() return 7 end})
assert(mt.anything == 7)
for k, v in pairs({a = 1}) do assert(k == "a" and v == 1) end
for i, v in ipairs({5}) do assert(i == 1 and v == 5) end
print("print works")
`}, Options{})
}

func TestRequire(t *testing.T) {
	e := newEngine(t, map[string]string{
		"lib/names.lua": `
loads = (loads or 0) + 1
local M = {}
function M.is_bad(name) return name == "Bad" end
return M
`,
		"lib/flag.lua": `flag_loaded = true`,
		"a.lua": `local names = require("lib.names")
events.on_character_create(function(ev)
  if names.is_bad(ev.name) then return events.reject("incorrect_name") end
end)`,
		"b.lua": `local names = require("lib.names")
assert(require("lib.flag") == true, "a module returning nothing gives true")
assert(loads == 1, "lib/names.lua ran " .. tostring(loads) .. " times")`,
	}, Options{})
	wantRejected(t, call(t, e, "Bad"), ReasonIncorrectName)
	wantAccepted(t, call(t, e, "Good"))
}

func TestRequireScriptRunsOnce(t *testing.T) {
	// b.lua is required by a.lua before the loader reaches it; it must not
	// run again (its handler would be registered twice).
	logs := &logBuffer{}
	newEngine(t, map[string]string{
		"a.lua": `require("b")`,
		"b.lua": `log.info("b loaded")`,
	}, Options{Logger: logs.logger()})
	n := 0
	for _, rec := range logs.records(t) {
		if rec["msg"] == "b loaded" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("b.lua ran %d times, want 1", n)
	}
}

func TestRequireRejectsPaths(t *testing.T) {
	parent := t.TempDir()
	if err := os.WriteFile(filepath.Join(parent, "outside.lua"), []byte("return 1"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"parent":        `require("../outside")`,
		"dotted parent": `require("..outside")`,
		"absolute":      `require("/etc/passwd")`,
		"absolute lua":  `require("` + filepath.ToSlash(filepath.Join(parent, "outside")) + `")`,
		"slash":         `require("lib/util")`,
		"backslash":     `require("lib\\util")`,
		"types":         `require("types.l2go")`,
		"missing":       `require("lib.missing")`,
		"empty":         `require("")`,
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			le := loadErr(t, map[string]string{
				"lib/util.lua":   "return {}",
				"types/l2go.lua": "---@meta",
				"evil.lua":       src,
			}, Options{})
			if !strings.Contains(le.Error(), "require") {
				t.Errorf("error = %v, want a require error", le)
			}
		})
	}
}

func TestRequireCircular(t *testing.T) {
	le := loadErr(t, map[string]string{
		"lib/a.lua": `return require("lib.b")`,
		"lib/b.lua": `return require("lib.a")`,
		"main.lua":  `require("lib.a")`,
	}, Options{})
	if !strings.Contains(le.Error(), "circular") {
		t.Errorf("error = %v, want circular require", le)
	}
}

func TestSymlinkOutsideFolder(t *testing.T) {
	parent := t.TempDir()
	outside := filepath.Join(parent, "outside.lua")
	if err := os.WriteFile(outside, []byte(`error("outside file ran")`), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "scripts")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.lua")); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}
	_, err := New(context.Background(), Options{Dir: dir})
	if err == nil {
		t.Fatal("a symlink to a file outside the scripts folder loaded")
	}
	if strings.Contains(err.Error(), "outside file ran") {
		t.Errorf("the file outside the folder ran: %v", err)
	}
}

// --- Logging ---

func TestLogModule(t *testing.T) {
	logs := &logBuffer{}
	newEngine(t, map[string]string{"logs.lua": `log.debug("d")
log.info("hello", "n", 5, "f", 1.5, "s", "x", "ok", true, "list", {1, 2}, "map", {a = 1})
log.warn("w")
log.error("e")
print("p", 1, nil)
`}, Options{Logger: logs.logger()})

	rec, ok := logs.find(t, "hello")
	if !ok {
		t.Fatal("log.info record missing")
	}
	if rec["level"] != "INFO" || rec["n"] != 5.0 || rec["f"] != 1.5 || rec["s"] != "x" || rec["ok"] != true {
		t.Errorf("record = %v", rec)
	}
	if fmt.Sprint(rec["list"]) != "[1 2]" || fmt.Sprint(rec["map"]) != "map[a:1]" {
		t.Errorf("tables = %v, %v", rec["list"], rec["map"])
	}
	if s, _ := rec["script"].(string); !strings.HasSuffix(s, "logs.lua:2") {
		t.Errorf("script = %q, want logs.lua:2", s)
	}
	for msg, level := range map[string]string{"d": "DEBUG", "w": "WARN", "e": "ERROR", "p\t1\tnil": "INFO"} {
		if rec, ok := logs.find(t, msg); !ok || rec["level"] != level {
			t.Errorf("record %q = %v, want level %s", msg, rec, level)
		}
	}
}

// --- Reload ---

func TestReloadReplacesHandlers(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"events/names.lua": rejectName("Old", ReasonIncorrectName),
		"events/gone.lua":  rejectName("Gone", ReasonNameExists),
	})
	e := newEngine(t, nil, Options{Dir: dir})
	wantRejected(t, call(t, e, "Old"), ReasonIncorrectName)
	wantRejected(t, call(t, e, "Gone"), ReasonNameExists)

	writeScript(t, dir, "events/names.lua", rejectName("New", ReasonIncorrectName))
	if err := os.Remove(filepath.Join(dir, "events", "gone.lua")); err != nil {
		t.Fatal(err)
	}
	if err := e.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	wantAccepted(t, call(t, e, "Old"))
	wantAccepted(t, call(t, e, "Gone"))
	wantRejected(t, call(t, e, "New"), ReasonIncorrectName)
}

func TestReloadBrokenKeepsOld(t *testing.T) {
	dir := writeScripts(t, map[string]string{
		"events/names.lua": rejectName("Old", ReasonIncorrectName),
	})
	e := newEngine(t, nil, Options{Dir: dir})

	writeScript(t, dir, "events/names.lua", rejectName("New", ReasonIncorrectName))
	writeScript(t, dir, "events/broken.lua", "events.on_character_create(")
	err := e.Reload(context.Background())
	if _, ok := errors.AsType[*LoadError](err); !ok {
		t.Fatalf("Reload = %v, want a *LoadError", err)
	}
	if !strings.Contains(err.Error(), "broken.lua") {
		t.Errorf("error does not name the broken file: %v", err)
	}
	// The old handlers are still in place, the new ones are not.
	wantRejected(t, call(t, e, "Old"), ReasonIncorrectName)
	wantAccepted(t, call(t, e, "New"))

	// Fixing the file and reloading again works.
	if err := os.Remove(filepath.Join(dir, "events", "broken.lua")); err != nil {
		t.Fatal(err)
	}
	if err := e.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	wantRejected(t, call(t, e, "New"), ReasonIncorrectName)
}

func TestReloadResetsGlobals(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.lua": `
counter = (counter or 0) + 1
events.on_character_create(function(ev)
  if counter ~= 1 then return events.reject("creation_failed") end
end)`})
	e := newEngine(t, nil, Options{Dir: dir})
	if err := e.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantAccepted(t, call(t, e, "x"))
}

// --- Concurrency ---

// Run with -race.
func TestConcurrentCalls(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.lua": `
events.on_character_create(function(ev)
  if ev.name:sub(1, 3) == "bad" then
    return events.reject("incorrect_name")
  end
end)`})
	e := newEngine(t, nil, Options{Dir: dir})

	const goroutines, calls = 16, 50
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range calls {
				name := fmt.Sprintf("good-%d-%d", g, i)
				if i%2 == 0 {
					name = fmt.Sprintf("bad-%d-%d", g, i)
				}
				res, err := e.Call(context.Background(), createEvent(name))
				if err != nil {
					t.Errorf("Call(%s): %v", name, err)
					return
				}
				if want := i%2 == 0; res.Rejected != want || res.Err != nil {
					t.Errorf("Call(%s) = %+v, want rejected=%v", name, res, want)
				}
			}
		})
	}
	wg.Go(func() {
		for range 5 {
			if err := e.Reload(context.Background()); err != nil {
				t.Errorf("Reload: %v", err)
			}
			e.Dispatch(createEvent("bad-dispatch"))
		}
	})
	wg.Wait()
}

// --- Errors ---

func TestToError(t *testing.T) {
	tests := []struct {
		msg        string
		file       string
		line       int
		wantString string
	}{
		{"scripts/a.lua:12: boom", "x", 1, "scripts/a.lua:12: boom"},
		{"C:\\l2go\\scripts\\a.lua:3: boom", "x", 1, "C:\\l2go\\scripts\\a.lua:3: boom"},
		{"no location", "scripts/b.lua", 7, "scripts/b.lua:7: no location"},
		{"no location", "scripts/b.lua", 0, "scripts/b.lua: no location"},
		{"a.lua:1: multi\nline", "", 0, "a.lua:1: multi\nline"},
	}
	for _, tt := range tests {
		if got := toError(errors.New(tt.msg), tt.file, tt.line).Error(); got != tt.wantString {
			t.Errorf("toError(%q) = %q, want %q", tt.msg, got, tt.wantString)
		}
	}
}

// --- Benchmarks ---

func BenchmarkCall(b *testing.B) {
	e := newEngine(b, map[string]string{"names.lua": `
local forbidden = { "annou", "ammou", "amnou", "anmou", "anou", "amou" }
events.on_character_create(function(ev)
  local name = ev.name:lower()
  for _, word in ipairs(forbidden) do
    if name:find(word, 1, true) then
      return events.reject("incorrect_name")
    end
  end
end)`}, Options{})
	ev := createEvent("Frostwind")
	ctx := context.Background()
	for b.Loop() {
		if _, err := e.Call(ctx, ev); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCallNoHandlers(b *testing.B) {
	e := newEngine(b, nil, Options{})
	ev := createEvent("Frostwind")
	ctx := context.Background()
	for b.Loop() {
		if _, err := e.Call(ctx, ev); err != nil {
			b.Fatal(err)
		}
	}
}
