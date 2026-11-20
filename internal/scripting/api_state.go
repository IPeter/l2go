package scripting

import (
	"fmt"
	"math"
	"strconv"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// Well-known quest variables (Mobius QuestState).
const (
	varCond    = "cond"
	varRestart = "restartTime"
	varMemo    = "memoState"
	varMemoEx  = "memoStateEx"
)

// L2J's column sizes for quest variables (character_quests.var, value).
const (
	maxVarName  = 20
	maxVarValue = 255
)

// The daily reset of daily quests, server time (Mobius QuestState).
const (
	resetHour   = 6
	resetMinute = 30
)

// Mobius QuestType ordinals.
const (
	questTypeRepeatable = 0
	questTypeOneTime    = 1
	questTypeDaily      = 2
)

// stateBox is the Go value inside a QuestState userdata. Like refs, it
// expires when the run that handed it out ends.
type stateBox struct {
	st     QuestState
	player PlayerRef
}

// newQuestState returns a QuestState value for st, owned by p, or nil. Within a
// run the same state is always the same value.
func (v *vm) newQuestState(L *lua.LState, p PlayerRef, st QuestState) lua.LValue {
	if st == nil || p == nil {
		return lua.LNil
	}
	key := refKey{kind: 's', id: p.ObjectID(), quest: st.Quest()}
	if ud, ok := v.refCache[key]; ok {
		if b, _ := ud.Value.(*stateBox); b != nil && b.st == st {
			return ud
		}
	}
	ud := v.newRef(L, &stateBox{st: st, player: p}, stateType)
	v.refCache[key] = ud
	return ud
}

// checkState returns the quest state that method was called on.
func checkState(L *lua.LState, method string) *stateBox {
	ud, _ := L.Get(1).(*lua.LUserData)
	var b *stateBox
	if ud != nil {
		b, _ = ud.Value.(*stateBox)
	}
	if b == nil {
		L.RaiseError("%s must be called on a quest state: st:%s()", method, method)
		return nil
	}
	if b.st == nil {
		L.RaiseError("st:%s(): this quest state was passed to an earlier handler; get it again with q:get_quest_state", method)
		return nil
	}
	return b
}

// getInt is Mobius QuestState.getInt: 0 when unset, empty or not an int.
func getInt(st QuestState, name string) int {
	s, ok := st.Get(name)
	if !ok || s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int(n)
}

// valueString turns a Lua value into a quest variable value as Java's
// set(String, int) and set(String, String) store it.
func valueString(L *lua.LState, n int, fn string) string {
	switch x := L.Get(n).(type) {
	case *lua.LNilType:
		return ""
	case lua.LString:
		return string(x)
	case lua.LNumber:
		f := float64(x)
		if f == math.Trunc(f) && math.Abs(f) <= 1<<53 {
			return strconv.FormatInt(int64(f), 10)
		}
		return strconv.FormatFloat(f, 'g', -1, 64)
	case lua.LBool:
		return strconv.FormatBool(bool(x))
	}
	L.ArgError(n, fn+": a string or a number expected, got "+L.Get(n).Type().String())
	return ""
}

// setVar sets a variable after checking L2J's column sizes.
func setVar(L *lua.LState, st QuestState, name, value, fn string) {
	if name == "" || len(name) > maxVarName {
		L.ArgError(2, fmt.Sprintf("%s: variable names are 1 to %d characters, got %q", fn, maxVarName, name))
		return
	}
	if len(value) > maxVarValue {
		L.ArgError(3, fmt.Sprintf("%s: values are at most %d characters", fn, maxVarValue))
		return
	}
	st.Set(name, value)
}

// stateMethods are the methods of QuestState values (Mobius QuestState).
func (v *vm) stateMethods() map[string]lua.LGFunction {
	getter := func(method string, get func(b *stateBox) lua.LValue) lua.LGFunction {
		return func(L *lua.LState) int {
			L.Push(get(checkState(L, method)))
			return 1
		}
	}
	isState := func(method string, state int) lua.LGFunction {
		return getter(method, func(b *stateBox) lua.LValue { return lua.LBool(b.st.State() == state) })
	}
	return map[string]lua.LGFunction{
		"get_quest_name": getter("get_quest_name", func(b *stateBox) lua.LValue { return lua.LString(b.st.Quest()) }),
		"get_player": func(L *lua.LState) int {
			L.Push(v.newPlayer(L, checkState(L, "get_player").player))
			return 1
		},
		"get_state":    getter("get_state", func(b *stateBox) lua.LValue { return lua.LNumber(b.st.State()) }),
		"is_created":   isState("is_created", StateCreated),
		"is_started":   isState("is_started", StateStarted),
		"is_completed": isState("is_completed", StateCompleted),
		"set_state": func(L *lua.LState) int {
			b := checkState(L, "set_state")
			state := int(checkInt(L, 2, "st:set_state"))
			if state < StateCreated || state > StateCompleted {
				L.ArgError(2, "st:set_state: State.CREATED, STARTED or COMPLETED expected")
				return 0
			}
			b.st.SetState(state)
			return 0
		},
		"start_quest": v.stStartQuest,
		"get_cond": getter("get_cond", func(b *stateBox) lua.LValue {
			if b.st.State() != StateStarted {
				return lua.LNumber(0)
			}
			return lua.LNumber(b.st.Cond())
		}),
		"is_cond": func(L *lua.LState) int {
			b := checkState(L, "is_cond")
			L.Push(lua.LBool(int64(b.st.Cond()) == checkInt(L, 2, "st:is_cond")))
			return 1
		},
		"set_cond": v.stSetCond,
		"set": func(L *lua.LState) int {
			b := checkState(L, "set")
			name := L.CheckString(2)
			setVar(L, b.st, name, valueString(L, 3, "st:set"), "st:set")
			return 0
		},
		"get": func(L *lua.LState) int {
			b := checkState(L, "get")
			if s, ok := b.st.Get(L.CheckString(2)); ok {
				L.Push(lua.LString(s))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"get_int": func(L *lua.LState) int {
			b := checkState(L, "get_int")
			L.Push(lua.LNumber(getInt(b.st, L.CheckString(2))))
			return 1
		},
		"unset": func(L *lua.LState) int {
			checkState(L, "unset").st.Unset(L.CheckString(2))
			return 0
		},
		"is_set": func(L *lua.LState) int {
			b := checkState(L, "is_set")
			_, ok := b.st.Get(L.CheckString(2))
			L.Push(lua.LBool(ok))
			return 1
		},
		"get_memo_state": getter("get_memo_state", func(b *stateBox) lua.LValue {
			if b.st.State() != StateStarted {
				return lua.LNumber(0)
			}
			return lua.LNumber(getInt(b.st, varMemo))
		}),
		"set_memo_state": func(L *lua.LState) int {
			b := checkState(L, "set_memo_state")
			b.st.Set(varMemo, strconv.FormatInt(checkInt(L, 2, "st:set_memo_state"), 10))
			return 0
		},
		"is_memo_state": func(L *lua.LState) int {
			b := checkState(L, "is_memo_state")
			L.Push(lua.LBool(int64(getInt(b.st, varMemo)) == checkInt(L, 2, "st:is_memo_state")))
			return 1
		},
		"get_memo_state_ex": func(L *lua.LState) int {
			b := checkState(L, "get_memo_state_ex")
			L.Push(lua.LNumber(memoEx(b, checkInt(L, 2, "st:get_memo_state_ex"))))
			return 1
		},
		"set_memo_state_ex": func(L *lua.LState) int {
			b := checkState(L, "set_memo_state_ex")
			slot := checkInt(L, 2, "st:set_memo_state_ex")
			val := checkInt(L, 3, "st:set_memo_state_ex")
			b.st.Set(varMemoEx+strconv.FormatInt(slot, 10), strconv.FormatInt(val, 10))
			return 0
		},
		"is_memo_state_ex": func(L *lua.LState) int {
			b := checkState(L, "is_memo_state_ex")
			slot := checkInt(L, 2, "st:is_memo_state_ex")
			L.Push(lua.LBool(int64(memoEx(b, slot)) == checkInt(L, 3, "st:is_memo_state_ex")))
			return 1
		},
		"exit_quest":       v.stExitQuest,
		"is_now_available": v.stIsNowAvailable,
		"set_restart_time": func(L *lua.LState) int {
			b := checkState(L, "set_restart_time")
			v.setRestartTime(L, b)
			return 0
		},
		"show_question_mark":       v.stTutorial("show_question_mark", TutorialQuestionMark, true),
		"play_tutorial_voice":      v.stTutorial("play_tutorial_voice", TutorialVoice, true),
		"show_tutorial_html":       v.stTutorial("show_tutorial_html", TutorialShowHTML, true),
		"close_tutorial_html":      v.stTutorial("close_tutorial_html", TutorialCloseHTML, false),
		"on_tutorial_client_event": v.stTutorial("on_tutorial_client_event", TutorialClientEvent, true),
	}
}

// memoEx is Mobius getMemoStateEx: 0 unless started.
func memoEx(b *stateBox, slot int64) int {
	if b.st.State() != StateStarted {
		return 0
	}
	return getInt(b.st, varMemoEx+strconv.FormatInt(slot, 10))
}

// stStartQuest is st:start_quest(): from CREATED, and only for a quest
// that is not custom, cond 1, STARTED and the accept sound.
func (v *vm) stStartQuest(L *lua.LState) int {
	b := checkState(L, "start_quest")
	w := v.needWorld(L, "st:start_quest")
	s := v.scripts[b.st.Quest()]
	if b.st.State() != StateCreated || (s != nil && s.custom) {
		return 0
	}
	b.st.Set(varCond, "1")
	b.st.SetState(StateStarted)
	w.PlaySound(b.player, soundAccept)
	return 0
}

// stSetCond is st:set_cond(n[, play_middle]): only while started.
func (v *vm) stSetCond(L *lua.LState) int {
	b := checkState(L, "set_cond")
	n := checkInt(L, 2, "st:set_cond")
	play := L.ToBool(3)
	if b.st.State() != StateStarted {
		return 0
	}
	b.st.Set(varCond, strconv.FormatInt(n, 10))
	if play {
		v.needWorld(L, "st:set_cond").PlaySound(b.player, soundMiddle)
	}
	return 0
}

// stExitQuest is st:exit_quest(repeatable[, play_finish]) and
// st:exit_quest(QuestType.X[, play_finish]).
func (v *vm) stExitQuest(L *lua.LState) int {
	b := checkState(L, "exit_quest")
	w := v.needWorld(L, "st:exit_quest")
	play := L.ToBool(3)
	switch x := L.Get(2).(type) {
	case lua.LBool:
		v.exitQuest(b, bool(x))
	case lua.LNumber:
		switch int(x) {
		case questTypeDaily:
			v.exitQuest(b, false)
			v.setRestartTime(L, b)
		case questTypeRepeatable, questTypeOneTime:
			v.exitQuest(b, int(x) == questTypeRepeatable)
		default:
			L.ArgError(2, "st:exit_quest: a QuestType value expected")
		}
	default:
		L.ArgError(2, "st:exit_quest: true, false or a QuestType value expected")
	}
	if play {
		w.PlaySound(b.player, soundFinish)
	}
	return 0
}

// exitQuest is Mobius exitQuest(repeatable): only a started quest ends;
// its registered quest items go first.
func (v *vm) exitQuest(b *stateBox, repeatable bool) {
	if b.st.State() != StateStarted {
		return
	}
	if s := v.scripts[b.st.Quest()]; s != nil {
		for _, id := range s.questItems {
			v.quest.world.TakeItems(b.player, id, -1)
		}
	}
	b.st.Exit(repeatable)
}

// setRestartTime is Mobius setRestartTime: the next 06:30 server time,
// keeping the seconds of now as Java's Calendar does.
func (v *vm) setRestartTime(L *lua.LState, b *stateBox) {
	now := v.needWorld(L, "st:set_restart_time").Now()
	day := now.Day()
	if now.Hour() >= resetHour {
		day++
	}
	ms := now.Nanosecond() / int(time.Millisecond) * int(time.Millisecond)
	next := time.Date(now.Year(), now.Month(), day, resetHour, resetMinute, now.Second(), ms, now.Location())
	b.st.Set(varRestart, strconv.FormatInt(next.UnixMilli(), 10))
}

// stIsNowAvailable is st:is_now_available(): the restart time has passed.
func (v *vm) stIsNowAvailable(L *lua.LState) int {
	b := checkState(L, "is_now_available")
	w := v.needWorld(L, "st:is_now_available")
	s, ok := b.st.Get(varRestart)
	if !ok {
		L.Push(lua.LFalse)
		return 1
	}
	at, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		L.RaiseError("st:is_now_available: restartTime %q is not a number", s)
		return 0
	}
	L.Push(lua.LBool(at <= w.Now().UnixMilli()))
	return 1
}

// stTutorial returns a tutorial method of QuestState (Q00255_Tutorial).
func (v *vm) stTutorial(method string, op TutorialOp, hasArg bool) lua.LGFunction {
	return func(L *lua.LState) int {
		b := checkState(L, method)
		arg := ""
		if hasArg {
			arg = L.ToStringMeta(L.CheckAny(2)).String()
		}
		v.needActions(L, "st:"+method).Tutorial(b.player, op, arg)
		return 0
	}
}

func stateString(ud *lua.LUserData) string {
	if b, ok := ud.Value.(*stateBox); ok && b.st != nil {
		return fmt.Sprintf("quest state %s (%s)", b.st.Quest(), stateName(b.st.State()))
	}
	return "quest state (expired)"
}

// stateName is Mobius State.getStateName.
func stateName(state int) string {
	switch state {
	case StateStarted:
		return "Started"
	case StateCompleted:
		return "Completed"
	}
	return "Start"
}
