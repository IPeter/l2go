package quest

import (
	"maps"
	"strconv"
	"time"
)

// State is one character's state of one quest: Mobius QuestState. Its
// methods follow Mobius line by line, quirks included (see the package tests):
//   - Cond is 0 unless the quest is started, but IsCond compares the step
//     whatever the state, and a completed quest keeps its last step in
//     memory.
//   - Set(VarCond, …) runs the skipped-step flag arithmetic and asks for
//     QuestList and the quest mark; SetCond does nothing unless started.
//   - IsMemoState ignores the state, MemoState and IsMemoStateEx do not.
//   - ExitQuest(true) removes the state from its Book; the State keeps its
//     state and step, and later changes to it are no longer saved.
//
// Every change that Mobius writes to the database marks the quest dirty in
// its Book; the Book's TakeChanges turns that into the stored documents.
type State struct {
	book  *Book
	name  string
	state StateID
	cond  int32
	vars  map[string]string // nil after ExitQuest, like Mobius _vars
	// stored reports whether the "<state>" row exists: the state left
	// CREATED once (Mobius createQuestInDb) or was loaded with the row.
	stored   bool
	detached bool // removed from its Book by ExitQuest(true) or New
}

// Name returns the quest's script name.
func (st *State) Name() string { return st.name }

// Book returns the book the state belongs to, or belonged to.
func (st *State) Book() *Book { return st.book }

// Quest returns the quest script as currently loaded (Mobius getQuest).
func (st *State) Quest() (Info, bool) { return st.book.quest(st.name) }

// Detached reports whether the state was removed from its Book, by
// ExitQuest(true) or by a newer state of the same quest (Book.New).
func (st *State) Detached() bool { return st.detached }

// State returns the state (Mobius getState).
func (st *State) State() StateID { return st.state }

// IsCreated reports whether the state is Created.
func (st *State) IsCreated() bool { return st.state == Created }

// IsStarted reports whether the state is Started.
func (st *State) IsStarted() bool { return st.state == Started }

// IsCompleted reports whether the state is Completed.
func (st *State) IsCompleted() bool { return st.state == Completed }

// SetState changes the state (Mobius setState). A real change is saved and
// asks for QuestList.
func (st *State) SetState(s StateID) {
	if st.state == s {
		return
	}
	newQuest := st.IsCreated()
	st.state = s
	if newQuest {
		st.stored = true // createQuestInDb; otherwise updateQuestInDb
	}
	st.changed()
	st.book.listener.QuestList()
}

// Get returns a variable (Mobius get); false if it is not set.
func (st *State) Get(name string) (string, bool) {
	v, ok := st.vars[name]
	return v, ok
}

// IsSet reports whether a variable is set (Mobius isSet).
func (st *State) IsSet(name string) bool {
	_, ok := st.vars[name]
	return ok
}

// GetInt returns a variable as a Java int (Mobius getInt): 0 if it is not
// set, empty or not an int.
func (st *State) GetInt(name string) int32 {
	n, _ := parseInt(st.vars[name])
	return n
}

// Vars returns a copy of the variables, nil if there are none. "<state>" is
// not one of them.
func (st *State) Vars() map[string]string {
	if len(st.vars) == 0 {
		return nil
	}
	return maps.Clone(st.vars)
}

// Set sets a variable (Mobius set) and saves it, whatever the state. Lua's
// nil is "", as Java's null. Setting VarCond also sets the step: the value
// as an int, 0 if it is not one; if the step changed, the skipped-step flags
// are updated, QuestList is asked for and, for a step above 0 of a quest that
// is not custom, the quest mark.
func (st *State) Set(name, value string) {
	if st.vars == nil {
		st.vars = make(map[string]string)
	}
	old := st.vars[name] // a missing variable parses as 0, like Java's null
	st.vars[name] = value
	st.changed()
	if name == VarCond {
		prev, _ := parseInt(old)
		cond, _ := parseInt(value)
		st.cond = cond
		st.condChanged(cond, prev)
	}
}

// SetInt sets a variable to a Java int (Mobius set(String, int)).
func (st *State) SetInt(name string, value int32) {
	st.Set(name, formatInt(value))
}

// Unset removes a variable (Mobius unset). Removing VarCond sets the step to
// 0, without packets.
func (st *State) Unset(name string) {
	if _, ok := st.vars[name]; !ok {
		return
	}
	delete(st.vars, name)
	if name == VarCond {
		st.cond = 0
	}
	st.changed()
}

// setInternal is Mobius setInternal: a variable read from the database.
func (st *State) setInternal(name, value string) {
	if st.vars == nil {
		st.vars = make(map[string]string)
	}
	if name == VarCond {
		if cond, ok := parseInt(value); ok {
			st.cond = cond
		}
	}
	st.vars[name] = value
}

// condChanged is Mobius's private setCond(cond, old): the skipped-step flags
// (VarFlags) and the packets of a step change.
//
// The flags are 0 (absent) while no step was skipped; QuestList then sends
// the step. Once a step is skipped, they hold the top bit, bit 0 (step 1 is
// never skipped) and one bit per step reached, and QuestList sends them
// instead (only when positive, which with the top bit set they never are).
func (st *State) condChanged(cond, old int32) {
	if cond == old {
		return
	}
	var flags int32
	if cond < 3 || cond > 31 {
		st.Unset(VarFlags)
	} else {
		flags = st.GetInt(VarFlags)
	}
	switch {
	case flags == 0:
		// No step skipped so far. Skipping one now starts the flags with
		// every step up to old and the new one.
		if cond > old+1 {
			flags = flagsSkipped | (bit(old) - 1) | bit(cond-1)
			st.SetInt(VarFlags, flags)
		}
	case cond < old:
		// Back to an earlier step: forget the steps after it.
		flags &= bit(cond) - 1
		if flags == bit(cond)-1 {
			st.Unset(VarFlags) // no step before it was skipped
		} else {
			flags |= flagsSkipped
			st.SetInt(VarFlags, flags)
		}
	default:
		// Forward: mark the new step.
		flags |= bit(cond - 1)
		st.SetInt(VarFlags, flags)
	}
	l := st.book.listener
	l.QuestList()
	// Mobius getQuest() is null for an unloaded script: the exception it
	// throws here is caught and logged, so no mark is sent.
	if q, ok := st.Quest(); ok && !q.Custom && cond > 0 {
		l.QuestMark(q.ID)
	}
}

// Cond returns the step (Mobius getCond): 0 unless the quest is started.
func (st *State) Cond() int32 {
	if st.IsStarted() {
		return st.cond
	}
	return 0
}

// Step returns the step as last set, whatever the state (Mobius _cond,
// which IsCond compares).
func (st *State) Step() int32 { return st.cond }

// IsCond reports whether the step is cond (Mobius isCond), whatever the
// state.
func (st *State) IsCond(cond int32) bool { return st.cond == cond }

// SetCond sets the step of a started quest (Mobius setCond), as
// Set(VarCond, …), and plays SoundMiddle if playMiddle is true. It does
// nothing unless the quest is started.
func (st *State) SetCond(cond int32, playMiddle bool) {
	if !st.IsStarted() {
		return
	}
	st.Set(VarCond, formatInt(cond))
	if playMiddle {
		st.book.listener.PlaySound(SoundMiddle)
	}
}

// MemoState returns VarMemo as an int: 0 unless the quest is started.
func (st *State) MemoState() int32 {
	if st.IsStarted() {
		return st.GetInt(VarMemo)
	}
	return 0
}

// SetMemoState sets VarMemo.
func (st *State) SetMemoState(v int32) { st.SetInt(VarMemo, v) }

// IsMemoState reports whether VarMemo is v, whatever the state.
func (st *State) IsMemoState(v int32) bool { return st.GetInt(VarMemo) == v }

// memoExVar is the variable of a memo state ex slot.
func memoExVar(slot int32) string { return VarMemoEx + formatInt(slot) }

// MemoStateEx returns the memo state ex of a slot: 0 unless the quest is
// started.
func (st *State) MemoStateEx(slot int32) int32 {
	if st.IsStarted() {
		return st.GetInt(memoExVar(slot))
	}
	return 0
}

// SetMemoStateEx sets the memo state ex of a slot.
func (st *State) SetMemoStateEx(slot, v int32) { st.SetInt(memoExVar(slot), v) }

// IsMemoStateEx reports whether MemoStateEx(slot) is v: false for v ≠ 0
// unless the quest is started.
func (st *State) IsMemoStateEx(slot, v int32) bool { return st.MemoStateEx(slot) == v }

// StartQuest accepts a quest (Mobius startQuest): from Created only, and not
// for a custom quest, it sets the step to 1 (QuestList, quest mark), the
// state to Started (QuestList) and plays SoundAccept. It does nothing for a
// quest that is not loaded, where Mobius throws.
func (st *State) StartQuest() {
	if !st.IsCreated() {
		return
	}
	if q, ok := st.Quest(); !ok || q.Custom {
		return
	}
	st.Set(VarCond, "1")
	st.SetState(Started)
	st.book.listener.PlaySound(SoundAccept)
}

// ExitQuest finishes the quest (Mobius exitQuest(boolean, boolean)). If the
// quest is started, its registered items go (Listener.RemoveQuestItems) and
// its variables are deleted; then, if repeatable, the state is removed from
// the Book (the document is deleted, QuestList), otherwise it becomes
// Completed (QuestList) and only "<state>" stays stored. playFinish plays
// SoundFinish in any case.
func (st *State) ExitQuest(repeatable, playFinish bool) {
	st.exit(repeatable)
	if playFinish {
		st.book.listener.PlaySound(SoundFinish)
	}
}

// ExitQuestType finishes the quest by type (Mobius exitQuest(QuestType,
// boolean)): Repeatable as ExitQuest(true), OneTime as ExitQuest(false), and
// Daily as ExitQuest(false) followed by SetRestartTime, which runs even if
// the quest was not started.
func (st *State) ExitQuestType(t Type, playFinish bool) {
	switch t {
	case Daily:
		st.exit(false)
		st.SetRestartTime()
	default:
		st.exit(t == Repeatable)
	}
	if playFinish {
		st.book.listener.PlaySound(SoundFinish)
	}
}

// exit is Mobius exitQuest(boolean).
func (st *State) exit(repeatable bool) {
	l := st.book.listener
	l.RemoveDeathNotify(st)
	if !st.IsStarted() {
		return
	}
	l.RemoveQuestItems(st)
	if repeatable {
		st.book.remove(st)
		l.QuestList()
	} else {
		st.vars = nil // deleteQuestInDb keeps only <state>
		st.changed()
		st.SetState(Completed)
	}
	st.vars = nil
}

// SetRestartTime sets VarRestart to the next daily reset (Mobius
// setRestartTime): today at ResetHour:ResetMinute in the time zone of the
// Book's clock, or tomorrow from ResetHour on. As in Mobius, the seconds and
// milliseconds of the current time are kept.
func (st *State) SetRestartTime() {
	now := st.book.now()
	day := now
	if now.Hour() >= ResetHour {
		day = now.AddDate(0, 0, 1)
	}
	ms := now.Nanosecond() / int(time.Millisecond)
	at := time.Date(day.Year(), day.Month(), day.Day(), ResetHour, ResetMinute, now.Second(), ms*int(time.Millisecond), now.Location())
	st.Set(VarRestart, strconv.FormatInt(at.UnixMilli(), 10))
}

// IsNowAvailable reports whether a daily quest can be taken again (Mobius
// isNowAvailable): VarRestart is set and not after the Book's clock. A value
// that is not a Java long is an error, where Mobius throws.
func (st *State) IsNowAvailable() (bool, error) {
	v, ok := st.vars[VarRestart]
	if !ok {
		return false, nil
	}
	at, ok := parseLong(v)
	if !ok {
		return false, &VarError{Quest: st.name, Var: VarRestart, Value: v}
	}
	return at <= st.book.now().UnixMilli(), nil
}

// VarError reports a variable whose value cannot be used, where Mobius
// throws a NumberFormatException.
type VarError struct {
	Quest string // the quest, empty for player and global variables
	Var   string
	Value string
}

func (e *VarError) Error() string {
	msg := "variable " + e.Var + " is not a number: " + strconv.Quote(e.Value)
	if e.Quest != "" {
		msg = "quest " + e.Quest + ": " + msg
	}
	return msg
}

// changed marks the quest dirty in its Book, unless the state was removed
// from it.
func (st *State) changed() {
	if !st.detached {
		st.book.markDirty(st.name)
	}
}
