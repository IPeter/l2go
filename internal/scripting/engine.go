// Package scripting runs the Lua scripts of the game server.
//
// The engine is synchronous. Invoke and the NPC methods (FirstTalk, Talk,
// Event) run the handlers on the caller's goroutine and return what they
// decided. The game server calls them from the world loop, which owns the
// Lua state like the rest of the world state. The engine does not lock:
// the caller serializes the calls, so two goroutines never use it at once.
// A Go function a handler calls (an Output or ref method) may call the
// engine again on the same goroutine; that nested run shares the outer
// run's time limit.
//
// Scripts run in a sandbox: the base, string, table and math libraries
// only, a require that reads from the scripts folder, and a time limit per
// call (one event, or one NPC handler).
//
// Reloading has two steps. Prepare loads every script into a fresh Lua
// state, off the loop, and fails if any script fails, so a broken script
// never replaces working ones. Swap, on the loop, puts the prepared
// scripts in place between two calls. Watch prepares on every file change
// and hands the result to a function that posts the Swap to the loop.
package scripting

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Defaults for Options.
const (
	DefaultDir         = "scripts"
	DefaultTimeout     = 50 * time.Millisecond
	DefaultLoadTimeout = 2 * time.Second
)

// Options configure an Engine.
type Options struct {
	// Dir is the scripts folder. Empty means DefaultDir.
	Dir string
	// Timeout bounds the handlers of one event, or one NPC handler. Zero
	// means DefaultTimeout.
	Timeout time.Duration
	// LoadTimeout bounds running one script file while loading. Zero means
	// DefaultLoadTimeout.
	LoadTimeout time.Duration
	// Include, if set, picks the scripts that run, by slash-separated path
	// relative to Dir (for example "events/character_create.lua"). Files in
	// lib/ stay available to require. Tests use it to load one script.
	Include func(path string) bool
	// Logger gets engine messages and the output of the log module. Nil
	// discards them.
	Logger *slog.Logger
	// Output receives what scripts do to the world, such as npc:say. Nil
	// makes those calls log at debug level only (as in `l2go scripts
	// check`).
	Output Output
}

func (o Options) withDefaults() Options {
	if o.Dir == "" {
		o.Dir = DefaultDir
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.LoadTimeout <= 0 {
		o.LoadTimeout = DefaultLoadTimeout
	}
	if o.Logger == nil {
		o.Logger = slog.New(slog.DiscardHandler)
	}
	return o
}

// Engine runs the scripts of one scripts folder.
//
// Invoke, FirstTalk, Talk, Event, HasFirstTalk, TalkScripts, HasScript and
// Swap belong to one goroutine, the world loop: the caller serializes them
// and the engine does not lock. Prepare and Watch may run on any
// goroutine, at the same time as the loop's calls. Close is for after the
// loop stopped.
type Engine struct {
	opts Options

	// Owned by the loop.
	vm      *vm       // the running scripts
	pending *Prepared // a Swap called during a run, applied when it ends
	closed  bool

	// compatMu serializes the deprecated Call, Reload, SwapLocked and Run
	// (compat.go) with each other and with Close.
	compatMu sync.Mutex
}

// New loads every script in opts.Dir. If any script fails, New returns a
// *LoadError listing all of them. The engine is ready to use; it needs no
// goroutine of its own.
func New(ctx context.Context, opts Options) (*Engine, error) {
	opts = opts.withDefaults()
	v, err := load(ctx, opts)
	if err != nil {
		return nil, err
	}
	opts.Logger.Info("scripts loaded", "dir", opts.Dir, "files", v.files)
	return &Engine{opts: opts, vm: v}, nil
}

// Invoke runs the handlers of ev on the caller's goroutine and returns
// what they decided. Handler failures (Lua errors, timeouts) are logged
// and reported in Result.Err; a failed handler does not change the result.
// If ctx is done or the engine is closed, no handler runs and Result.Err
// says why. Cancelling ctx also stops a running handler.
func (e *Engine) Invoke(ctx context.Context, ev Event) Result {
	if err := e.stopped(ctx); err != nil {
		res := baseResult(ev)
		res.Err = err
		return res
	}
	defer e.settle()
	return e.vm.fire(ctx, ev, e.opts.Timeout)
}

// stopped returns why no call can run now, or nil.
func (e *Engine) stopped(ctx context.Context) error {
	if e.closed {
		return ErrStopped
	}
	return ctx.Err()
}

// settle applies a Swap or Close requested while a run was in progress,
// once the outermost run has ended.
func (e *Engine) settle() {
	if e.vm.depth > 0 {
		return
	}
	if e.closed {
		e.vm.close()
		return
	}
	if p := e.pending; p != nil {
		e.pending = nil
		e.Swap(p)
	}
}

// Prepared holds scripts loaded by Prepare, waiting for Swap.
type Prepared struct {
	vm *vm
}

// Files returns the number of .lua files the prepared scripts compiled.
func (p *Prepared) Files() int {
	if p == nil || p.vm == nil {
		return 0
	}
	return p.vm.files
}

// Close discards prepared scripts that will not be swapped in, for example
// because the loop stopped before the Swap ran. Close after Swap does
// nothing. Close and Swap must not run at the same time.
func (p *Prepared) Close() {
	if p != nil && p.vm != nil {
		p.vm.close()
		p.vm = nil
	}
}

// Prepare loads every script again into a fresh Lua state, as New does,
// without touching the running scripts. It may run on any goroutine, at
// the same time as the loop's calls. If any script fails, it returns a
// *LoadError listing all of them and the running scripts stay. Hand the
// result to Swap on the loop.
func (e *Engine) Prepare(ctx context.Context) (*Prepared, error) {
	v, err := load(ctx, e.opts)
	if err != nil {
		return nil, err
	}
	return &Prepared{vm: v}, nil
}

// Swap replaces the running scripts with p and closes the old Lua state.
// It belongs to the loop, like Invoke. Called while a handler runs (from a
// Go function the handler called), it waits until the outermost run ends.
// A nil p, or one already swapped in or closed, does nothing; on a closed
// engine, Swap closes p.
func (e *Engine) Swap(p *Prepared) {
	if p == nil || p.vm == nil {
		return
	}
	if e.closed {
		p.Close()
		return
	}
	if e.vm.depth > 0 {
		if e.pending != nil && e.pending != p {
			e.pending.Close()
		}
		e.pending = p
		return
	}
	old := e.vm
	e.vm, p.vm = p.vm, nil
	old.close()
	e.opts.Logger.Info("scripts reloaded", "dir", e.opts.Dir, "files", e.vm.files)
}

// Close closes the Lua state. Call it once the loop no longer calls the
// engine; it is safe on any goroutine then, and closing twice does
// nothing. Afterwards Invoke reports ErrStopped and the NPC methods have
// no reply.
func (e *Engine) Close() {
	e.compatMu.Lock()
	defer e.compatMu.Unlock()
	e.closed = true
	if e.pending != nil {
		e.pending.Close()
		e.pending = nil
	}
	if e.vm.depth == 0 {
		e.vm.close()
	}
}
