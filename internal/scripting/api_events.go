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
		v.handlers[event] = append(v.handlers[event], newHandler(fn))
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
		hint := ""
		if run.spec.filter {
			hint = "; return false to drop it"
		}
		L.RaiseError("events.reject: %s events cannot be rejected%s", run.spec.name, hint)
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

// fire runs the handlers of ev in registration order until one rejects or
// drops it. All handlers share one time limit. A failing handler is logged
// and skipped.
func (v *vm) fire(ctx context.Context, ev Event, timeout time.Duration) Result {
	spec, ok := specOf(ev.Name)
	if !ok {
		return Result{Err: fmt.Errorf("scripting: unknown event %q", ev.Name)}
	}
	res := baseResult(ev)
	handlers := v.handlers[ev.Name]
	if len(handlers) == 0 {
		return res
	}

	ctx, end := v.begin(ctx, timeout)
	defer end()
	run := &eventRun{spec: spec}
	prev := v.event
	v.event = run
	defer func() { v.event = prev }()

	table, _ := v.toLua(v.L, ev.Data).(*lua.LTable)
	var errs []error
	fail := func(h handler, e *Error) {
		v.logger.Warn("script handler failed", "event", ev.Name, "handler", h.where(), "err", e)
		errs = append(errs, e)
	}
	for _, h := range handlers {
		if spec.filter {
			// Every handler sees the text as the handlers before it left it.
			table.RawSetString("text", lua.LString(res.Text))
		}
		ret, _, err := v.call(h.fn, table)
		if err != nil {
			fail(h, handlerError(ctx, err, h, ev.Name+" handler", timeout))
			if ctx.Err() != nil {
				break // the time is up for the remaining handlers too
			}
			continue
		}
		if spec.filter {
			if e := filterResult(&res, ret); e != "" {
				fail(h, &Error{File: h.file, Line: h.line, Msg: fmt.Sprintf("%s handler %s", ev.Name, e)})
			}
		} else if ret != lua.LNil && !isRejection(ret) {
			fail(h, &Error{File: h.file, Line: h.line, Msg: fmt.Sprintf(
				"%s handler returned a %s; return nothing or events.reject(reason)", ev.Name, ret.Type())})
		}
		if run.rejected || res.Dropped {
			break
		}
	}
	res.Rejected, res.Reason = run.rejected, run.reason
	res.Err = errors.Join(errs...)
	return res
}

// filterResult applies what a filter handler returned: nothing keeps the
// text, a string replaces it, false or "" drops the event. It returns a
// description of a wrong return value, or "".
func filterResult(res *Result, ret lua.LValue) string {
	switch x := ret.(type) {
	case *lua.LNilType:
	case lua.LBool:
		if x {
			return "returned true; return nothing, a new text or false"
		}
		res.Dropped = true
	case lua.LString:
		if x == "" {
			res.Dropped = true
		} else {
			res.Text = string(x)
		}
	default:
		return fmt.Sprintf("returned a %s; return nothing, a new text or false", ret.Type())
	}
	return ""
}
