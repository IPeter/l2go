package scripting

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

var (
	// ErrStopped is returned when the engine is not running any more.
	ErrStopped = errors.New("scripting: engine stopped")
	// ErrTimeout is wrapped by the Error of a script that ran too long.
	ErrTimeout = errors.New("scripting: timed out")
)

// Error is a script failure: a syntax error, a Lua error or a timeout.
type Error struct {
	File string // script path, empty if unknown
	Line int    // 0 if unknown
	Msg  string
	Err  error // ErrTimeout for timeouts, nil otherwise
}

func (e *Error) Error() string {
	switch {
	case e.File != "" && e.Line > 0:
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Msg)
	case e.File != "":
		return e.File + ": " + e.Msg
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Err }

// luaLocation matches the "file:line: " prefix Lua puts on error messages.
var luaLocation = regexp.MustCompile(`^(.+?):(\d+): (?s:(.*))$`)

// toError turns an error from gopher-lua into an *Error. file and line are
// used when the message carries no location.
func toError(err error, file string, line int) *Error {
	msg := err.Error()
	if ae, ok := errors.AsType[*lua.ApiError](err); ok && ae.Object != nil {
		msg = ae.Object.String() // without the stack trace
	}
	if m := luaLocation.FindStringSubmatch(msg); m != nil {
		n, _ := strconv.Atoi(m[2])
		return &Error{File: m[1], Line: n, Msg: m[3]}
	}
	return &Error{File: file, Line: line, Msg: msg}
}
