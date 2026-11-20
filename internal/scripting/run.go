package scripting

import (
	"context"
	"errors"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// handler is a Lua function registered by a script.
type handler struct {
	fn   *lua.LFunction
	file string // where the function is defined
	line int
}

func newHandler(fn *lua.LFunction) handler {
	h := handler{fn: fn, file: "?"}
	if fn.Proto != nil {
		h.file, h.line = fn.Proto.SourceName, fn.Proto.LineDefined
	}
	return h
}

func (h handler) where() string { return fmt.Sprintf("%s:%d", h.file, h.line) }

// begin starts a run: one call into the engine that runs handlers. The
// outermost run puts its time limit on the Lua state. A nested run, started
// by a Go function a handler called (an Output or ref method that fires
// another event), shares the outer run's deadline: replacing the context
// of a running Lua state would break the outer handler. The returned
// function ends the run; when the outermost run ends, the refs handed to
// its handlers stop working.
func (v *vm) begin(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	v.depth++
	if v.depth > 1 {
		if outer := v.L.Context(); outer != nil {
			return outer, v.end
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	v.L.SetContext(ctx)
	return ctx, func() {
		v.L.RemoveContext()
		cancel()
		v.end()
	}
}

func (v *vm) end() {
	v.depth--
	if v.depth == 0 {
		v.expireRefs()
	}
}

// call runs fn in protected mode and returns its first two results. A Go
// panic in a function fn calls becomes an error too.
func (v *vm) call(fn *lua.LFunction, args ...lua.LValue) (lua.LValue, lua.LValue, error) {
	L := v.L
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 2, Protect: true}, args...); err != nil {
		return lua.LNil, lua.LNil, err
	}
	r1, r2 := L.Get(-2), L.Get(-1)
	L.Pop(2)
	return r1, r2, nil
}

// callWith is call with the arguments built inside the protected call, so
// a Go panic while building them (a ref's ObjectID) is an error too.
func (v *vm) callWith(fn *lua.LFunction, args func() []lua.LValue) (lua.LValue, lua.LValue, error) {
	wrapper := v.L.NewFunction(func(L *lua.LState) int {
		a := args()
		L.Push(fn)
		for _, x := range a {
			L.Push(x)
		}
		L.Call(len(a), 2)
		return 2
	})
	return v.call(wrapper)
}

// handlerError turns the error of a failed handler into an *Error, naming
// timeouts and cancellation. what names the handler, for example
// "chat handler".
func handlerError(ctx context.Context, err error, h handler, what string, timeout time.Duration) *Error {
	e := toError(err, h.file, h.line)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		e.Msg, e.Err = fmt.Sprintf("%s timed out after %v", what, timeout), ErrTimeout
	case ctx.Err() != nil:
		e.Msg, e.Err = what+" canceled", ctx.Err()
	}
	return e
}
