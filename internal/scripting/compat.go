package scripting

import "context"

// This file keeps the API of the asynchronous engine, where Run owned the
// Lua state on its own goroutine, so callers can move to the world loop one
// at a time. The wrappers serialize with a mutex. They must not be mixed
// with the loop-only methods (Invoke, Swap, the NPC methods) on one engine.

// Run waits until ctx is done, then closes the engine. The engine no
// longer needs a goroutine of its own.
//
// Deprecated: Call Invoke and the NPC methods on the world loop, and Close
// once the loop has stopped.
func (e *Engine) Run(ctx context.Context) error {
	<-ctx.Done()
	e.Close()
	return nil
}

// Call runs the handlers of ev like Invoke, from any goroutine: calls are
// serialized with a mutex. The error is ctx.Err() if ctx is done before
// the handlers run, and ErrStopped once the engine is closed; handler
// failures are in Result.Err.
//
// Deprecated: Use Invoke on the world loop.
func (e *Engine) Call(ctx context.Context, ev Event) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	e.compatMu.Lock()
	defer e.compatMu.Unlock()
	if e.closed {
		return Result{}, ErrStopped
	}
	return e.Invoke(ctx, ev), nil
}

// SwapLocked is Swap serialized with Call, from any goroutine. It lets
// Watch run next to the deprecated Call:
//
//	engine.Watch(ctx, scripting.DefaultDebounce, engine.SwapLocked)
//
// Deprecated: Post Swap to the world loop instead.
func (e *Engine) SwapLocked(p *Prepared) {
	e.compatMu.Lock()
	defer e.compatMu.Unlock()
	e.Swap(p)
}

// Reload prepares every script again and swaps them in, serialized with
// Call. If any script fails, the old scripts stay and Reload returns a
// *LoadError.
//
// Deprecated: Use Prepare off the loop and Swap on the loop.
func (e *Engine) Reload(ctx context.Context) error {
	p, err := e.Prepare(ctx)
	if err != nil {
		return err
	}
	e.compatMu.Lock()
	defer e.compatMu.Unlock()
	if e.closed {
		p.Close()
		return ErrStopped
	}
	e.Swap(p)
	return nil
}
