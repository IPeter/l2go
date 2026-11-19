package scripting

import (
	"context"
	"log/slog"

	lua "github.com/yuin/gopher-lua"
)

// logFuncs is the log module: log.debug, log.info, log.warn and
// log.error(msg, key, value, ...). Each record also gets the script file
// and line as "script".
func (v *vm) logFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"debug": v.logAt(slog.LevelDebug),
		"info":  v.logAt(slog.LevelInfo),
		"warn":  v.logAt(slog.LevelWarn),
		"error": v.logAt(slog.LevelError),
	}
}

func (v *vm) logAt(level slog.Level) lua.LGFunction {
	return func(L *lua.LState) int {
		msg := L.ToStringMeta(L.Get(1)).String()
		top := L.GetTop()
		attrs := make([]slog.Attr, 0, 1+top/2)
		attrs = append(attrs, slog.String("script", where(L)))
		for i := 2; i <= top; i += 2 {
			key := L.ToStringMeta(L.Get(i)).String()
			attrs = append(attrs, slog.Any(key, fromLua(L.Get(i+1))))
		}
		v.logger.LogAttrs(context.Background(), level, msg, attrs...)
		return 0
	}
}
