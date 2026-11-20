package scripting

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// fileKind tells how the loader treats a .lua file.
type fileKind int

const (
	kindScript fileKind = iota // runs when the scripts load
	kindLib                    // lib/: shared helpers, run only through require
	kindTypes                  // types/: editor definitions, only compiled
)

// kindOf returns the kind of the file at a slash-separated path relative
// to the scripts folder.
func kindOf(path string) fileKind {
	switch {
	case strings.HasPrefix(path, "lib/"):
		return kindLib
	case strings.HasPrefix(path, "types/"):
		return kindTypes
	}
	return kindScript
}

// vm is one Lua state with the scripts loaded into it. Only one goroutine
// may use it at a time.
type vm struct {
	L      *lua.LState
	dir    string
	logger *slog.Logger
	output Output // nil: npc:say only logs

	protos map[string]*lua.FunctionProto // compiled files by slash path
	broken map[string]bool               // files that did not compile
	loaded map[string]lua.LValue         // results of files that ran

	handlers  map[string][]handler           // events.on_<name> handlers by event name
	scripts   map[string]*npcScript          // script.new and quest.new objects by name
	order     []*npcScript                   // the same, in load order
	questIDs  map[int]*npcScript             // quests by ID
	firstTalk map[int]firstTalkOwner         // by NPC template ID
	byID      [numHooks]map[int][]*npcScript // scripts registered for an ID, per handler kind, in load order
	loading   bool                           // scripts are loading: handlers may register
	running   []string                       // files running while loading, innermost last
	event     *eventRun                      // the event being handled, or nil
	depth     int                            // runs in progress (nested runs count too)
	refs      []*lua.LUserData               // Player, Npc and QuestState values handed out in this run
	refCache  map[refKey]*lua.LUserData      // the same values by object, so one object is one value per run
	timers    map[timerKey]*questTimer       // running quest timers
	timerSeq  uint64                         // start order of quest timers
	quest     questEnv                       // what the quest API works with
	engine    *Engine                        // the engine running this vm, nil while prepared

	files int // .lua files compiled
}

// display returns the path of a script as shown in errors and logs.
func (v *vm) display(path string) string {
	return filepath.Join(v.dir, filepath.FromSlash(path))
}

// currentFile returns the slash path of the innermost file whose top level
// is running, while the scripts load. A lib/ helper that calls script.new
// for its caller is not a file running at the top level, so the caller's
// file counts.
func (v *vm) currentFile() string {
	if n := len(v.running); n > 0 {
		return v.running[n-1]
	}
	return ""
}

// close closes the Lua state. Closing twice does nothing.
func (v *vm) close() {
	if v.L != nil {
		v.L.Close()
		v.L = nil
	}
}

// LoadError lists every script that failed to load.
type LoadError struct {
	Dir  string
	Errs []error // one *Error per problem
}

func (e *LoadError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "scripting: scripts in %s failed to load:", e.Dir)
	for _, err := range e.Errs {
		b.WriteString("\n")
		b.WriteString(err.Error())
	}
	return b.String()
}

func (e *LoadError) Unwrap() []error { return e.Errs }

// source is one .lua file read from the scripts folder.
type source struct {
	path string // slash-separated, relative to the scripts folder
	data []byte
}

// load builds a vm from opts.Dir. It compiles every file, then runs every
// script outside lib/ and types/ in path order. It reports every failing
// file, not only the first, as a *LoadError.
func load(ctx context.Context, opts Options) (*vm, error) {
	files, errs, err := readScripts(opts.Dir, opts.Include)
	if err != nil {
		return nil, err
	}
	v := &vm{
		dir:       opts.Dir,
		logger:    opts.Logger,
		output:    opts.Output,
		protos:    make(map[string]*lua.FunctionProto),
		broken:    make(map[string]bool),
		loaded:    make(map[string]lua.LValue),
		handlers:  make(map[string][]handler),
		scripts:   make(map[string]*npcScript),
		questIDs:  make(map[int]*npcScript),
		firstTalk: make(map[int]firstTalkOwner),
		refCache:  make(map[refKey]*lua.LUserData),
		timers:    make(map[timerKey]*questTimer),
		quest: questEnv{
			world:  opts.World,
			rand:   opts.Rand,
			consts: opts.Constants,
			rates:  opts.Rates,
		},
	}
	for k := range v.byID {
		v.byID[k] = make(map[int][]*npcScript)
	}
	v.quest.actions, _ = opts.World.(QuestActions)
	v.L = v.newState()

	for _, f := range files {
		proto, err := compile(f.data, v.display(f.path))
		if err != nil {
			errs = append(errs, err)
			v.broken[f.path] = true
			continue
		}
		v.protos[f.path] = proto
		v.files++
	}

	v.loading = true
	for _, f := range files {
		if kindOf(f.path) != kindScript || v.broken[f.path] {
			continue
		}
		if _, done := v.loaded[f.path]; done {
			continue // already run by a require
		}
		if err := v.runScript(ctx, f.path, opts.LoadTimeout); err != nil {
			errs = append(errs, err)
		}
	}
	v.loading = false

	if len(errs) > 0 {
		v.close()
		return nil, &LoadError{Dir: opts.Dir, Errs: errs}
	}
	return v, nil
}

// readScripts reads every .lua file under dir through an os.Root, so a
// symlink cannot reach outside the folder. Hidden files and folders are
// skipped, and so are scripts include rejects (lib/ and types/ are always
// read). Unreadable files are returned as errors; only a folder that
// cannot be opened fails the whole read.
func readScripts(dir string, include func(string) bool) ([]source, []error, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("scripting: open scripts folder: %w", err)
	}
	defer root.Close()

	fsys := root.FS()
	var files []source
	var errs []error
	walkErr := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		display := filepath.Join(dir, filepath.FromSlash(path))
		if err != nil {
			errs = append(errs, &Error{File: display, Msg: err.Error()})
			return nil
		}
		if path != "." && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".lua") {
			return nil
		}
		if include != nil && kindOf(path) == kindScript && !include(path) {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			errs = append(errs, &Error{File: display, Msg: err.Error()})
			return nil
		}
		files = append(files, source{path: path, data: data})
		return nil
	})
	if walkErr != nil {
		return nil, nil, fmt.Errorf("scripting: read scripts folder: %w", walkErr)
	}
	return files, errs, nil
}

// compile parses and compiles one file. Errors carry the file and line.
func compile(src []byte, name string) (*lua.FunctionProto, error) {
	chunk, err := parse.Parse(bytes.NewReader(src), name)
	if err != nil {
		pe, ok := errors.AsType[*parse.Error](err)
		if !ok {
			return nil, &Error{File: name, Msg: err.Error()}
		}
		msg, line := pe.Message, pe.Pos.Line
		if line == parse.EOF {
			msg += " at the end of the file"
			line = bytes.Count(src, []byte("\n")) + 1
		} else if pe.Token != "" {
			msg = fmt.Sprintf("%s near '%s'", msg, pe.Token)
		}
		return nil, &Error{File: name, Line: line, Msg: msg}
	}
	proto, err := lua.Compile(chunk, name)
	if err != nil {
		if ce, ok := errors.AsType[*lua.CompileError](err); ok {
			return nil, &Error{File: name, Line: ce.Line, Msg: ce.Message}
		}
		return nil, &Error{File: name, Msg: err.Error()}
	}
	return proto, nil
}

// runScript runs one script file within timeout.
func (v *vm) runScript(ctx context.Context, path string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	v.L.SetContext(ctx)
	defer v.L.RemoveContext()

	run := v.L.NewFunction(func(L *lua.LState) int {
		v.loadModule(L, path, path)
		return 0
	})
	err := v.L.CallByParam(lua.P{Fn: run, Protect: true})
	if err == nil {
		return nil
	}
	e := toError(err, v.display(path), 0)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		e.Msg, e.Err = fmt.Sprintf("did not finish loading within %v", timeout), ErrTimeout
	case ctx.Err() != nil:
		e.Msg, e.Err = "loading canceled", ctx.Err()
	}
	return e
}

// Check loads every script in opts.Dir into a sandbox, as New does, then
// discards it. It returns the number of .lua files checked. On failure the
// error is a *LoadError listing every problem with file and line,
// including NPCs whose first talk two scripts claim.
func Check(ctx context.Context, opts Options) (int, error) {
	v, err := load(ctx, opts.withDefaults())
	if err != nil {
		return 0, err
	}
	v.close()
	return v.files, nil
}
