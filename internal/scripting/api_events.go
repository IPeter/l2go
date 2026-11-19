package scripting

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// rejectionType names the metatable of the values events.reject returns.
const rejectionType = "l2go.rejection"

// rejection is the Go value inside a rejection userdata.
type rejection struct{ reason string }

// eventRun is the state of the event being handled.
type eventRun struct {
	spec     *eventSpec
	rejected bool
	reason   string
}

// eventsFuncs is the events module: events.on_<event>(fn) for every event
// in eventSpecs, and events.reject(reason).
func (v *vm) eventsFuncs() map[string]lua.LGFunction {
	funcs := map[string]lua.LGFunction{"reject": v.luaReject}
	for _, spec := range eventSpecs {
		funcs["on_"+spec.name] = v.luaOn(spec.name)
	}
	return funcs
}

// luaOn returns events.on_<event>(fn), which adds fn to the handlers of
// the event. Handlers run in the order they were registered.
func (v *vm) luaOn(event string) lua.LGFunction {
	return func(L *lua.LState) int {
		fn := L.CheckFunction(1)
		if !v.loading {
			L.RaiseError("events.on_%s: handlers can only be registered while the scripts load", event)
			return 0
		}
		h := handler{fn: fn, file: "?", line: 0}
		if fn.Proto != nil {
			h.file, h.line = fn.Proto.SourceName, fn.Proto.LineDefined
		}
		v.handlers[event] = append(v.handlers[event], h)
		return 0
	}
}

// luaReject is events.reject(reason). It rejects the event being handled
// and returns a rejection value, so handlers can write
// `return events.reject("incorrect_name")`. The first rejection wins.
func (v *vm) luaReject(L *lua.LState) int {
	reason := L.CheckString(1)
	run := v.event
	// RaiseError and ArgError do not return; the returns are for readers.
	if run == nil {
		L.RaiseError("events.reject: called outside an event handler")
		return 0
	}
	if len(run.spec.reasons) == 0 {
		L.RaiseError("events.reject: %s events cannot be rejected", run.spec.name)
		return 0
	}
	if !slices.Contains(run.spec.reasons, reason) {
		L.ArgError(1, fmt.Sprintf("unknown reason %q for %s, use one of: %s",
			reason, run.spec.name, strings.Join(run.spec.reasons, ", ")))
		return 0
	}
	if !run.rejected {
		run.rejected, run.reason = true, reason
	}
	ud := L.NewUserData()
	ud.Value = rejection{reason: reason}
	L.SetMetatable(ud, L.GetTypeMetatable(rejectionType))
	L.Push(ud)
	return 1
}

// initRejectionType registers the metatable of rejection values.
func initRejectionType(L *lua.LState) {
	mt := L.NewTypeMetatable(rejectionType)
	L.SetField(mt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		r, _ := L.CheckUserData(1).Value.(rejection)
		L.Push(lua.LString("rejection: " + r.reason))
		return 1
	}))
	L.SetField(mt, "__metatable", lua.LFalse)
}

func isRejection(lv lua.LValue) bool {
	ud, ok := lv.(*lua.LUserData)
	if !ok {
		return false
	}
	_, ok = ud.Value.(rejection)
	return ok
}

// fire runs the handlers of ev in registration order until one rejects.
// All handlers share one timeout. A failing handler is logged and skipped.
func (v *vm) fire(ctx context.Context, ev Event, timeout time.Duration) Result {
	spec, ok := specOf(ev.Name)
	if !ok {
		return Result{Err: fmt.Errorf("scripting: unknown event %q", ev.Name)}
	}
	handlers := v.handlers[ev.Name]
	if len(handlers) == 0 {
		return Result{}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	L := v.L
	L.SetContext(ctx)
	defer L.RemoveContext()
	run := &eventRun{spec: spec}
	v.event = run
	defer func() { v.event = nil }()

	table := toLua(L, ev.Data)
	var errs []error
	for _, h := range handlers {
		err := L.CallByParam(lua.P{Fn: h.fn, NRet: 1, Protect: true}, table)
		if err != nil {
			e := toError(err, h.file, h.line)
			switch {
			case errors.Is(ctx.Err(), context.DeadlineExceeded):
				e.Msg, e.Err = fmt.Sprintf("%s handler timed out after %v", ev.Name, timeout), ErrTimeout
			case ctx.Err() != nil:
				e.Msg, e.Err = ev.Name+" handler canceled", ctx.Err()
			}
			v.logger.Warn("script handler failed", "event", ev.Name, "handler", h.where(), "err", e)
			errs = append(errs, e)
			if ctx.Err() != nil {
				break // the time is up for the remaining handlers too
			}
			continue
		}
		ret := L.Get(-1)
		L.Pop(1)
		if ret != lua.LNil && !isRejection(ret) {
			e := &Error{File: h.file, Line: h.line, Msg: fmt.Sprintf(
				"%s handler returned a %s; return nothing or events.reject(reason)", ev.Name, ret.Type())}
			v.logger.Warn("script handler failed", "event", ev.Name, "handler", h.where(), "err", e)
			errs = append(errs, e)
		}
		if run.rejected {
			break
		}
	}
	return Result{Rejected: run.rejected, Reason: run.reason, Err: errors.Join(errs...)}
}
