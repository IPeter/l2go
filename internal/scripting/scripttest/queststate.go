package scripttest

import (
	"strconv"

	"github.com/IPeter/l2go/internal/scripting"
)

// QuestState is the fake world's quest state: Mobius QuestState's rules
// for state, variables and the cond flags, with counters for the packets
// the real model sends. It stands in for internal/quest's model.
type QuestState struct {
	quest  string
	custom bool
	state  int
	cond   int
	vars   map[string]string
	owner  *Player
	world  *World
	exited bool // deleted by exit_quest(true)
}

var _ scripting.QuestState = (*QuestState)(nil)

// Quest returns the quest's name.
func (s *QuestState) Quest() string { return s.quest }

// State returns the state (scripting.StateCreated, ...).
func (s *QuestState) State() int { return s.state }

// Vars returns a copy of the variables.
func (s *QuestState) Vars() map[string]string {
	out := make(map[string]string, len(s.vars))
	for k, v := range s.vars {
		out[k] = v
	}
	return out
}

// SetState is Mobius setState.
func (s *QuestState) SetState(state int) {
	if s.state == state {
		return
	}
	s.state = state
	s.world.questList(s.owner)
}

// Get returns a variable.
func (s *QuestState) Get(name string) (string, bool) {
	v, ok := s.vars[name]
	return v, ok
}

// Cond returns the cond as last set.
func (s *QuestState) Cond() int { return s.cond }

// Set is Mobius set(name, value) with the cond rules.
func (s *QuestState) Set(name, value string) {
	if s.vars == nil {
		s.vars = make(map[string]string)
	}
	old, had := s.vars[name]
	s.vars[name] = value
	if name != "cond" {
		return
	}
	prev := 0
	if had {
		prev, _ = strconv.Atoi(old)
	}
	next, _ := strconv.Atoi(value)
	s.cond = next
	s.setCond(next, prev)
}

// setCond is Mobius QuestState.setCond(int, int): the
// __compltdStateFlags arithmetic in 32-bit ints, then QuestList and the
// quest mark.
func (s *QuestState) setCond(cond, old int) {
	if cond == old {
		return
	}
	var flags int32
	if cond < 3 || cond > 31 {
		s.Unset("__compltdStateFlags")
	} else {
		v, _ := strconv.ParseInt(s.vars["__compltdStateFlags"], 10, 32)
		flags = int32(v)
	}
	switch {
	case flags == 0:
		if cond > old+1 {
			flags = int32(-0x7fffffff) // 0x80000001
			flags |= int32(1)<<uint(old) - 1
			flags |= int32(1) << uint(cond-1)
			s.Set("__compltdStateFlags", strconv.Itoa(int(flags)))
		}
	case cond < old:
		flags &= int32(1)<<uint(cond) - 1
		if flags == int32(1)<<uint(cond)-1 {
			s.Unset("__compltdStateFlags")
		} else {
			flags |= int32(-0x7fffffff)
			s.Set("__compltdStateFlags", strconv.Itoa(int(flags)))
		}
	default:
		flags |= int32(1) << uint(cond-1)
		s.Set("__compltdStateFlags", strconv.Itoa(int(flags)))
	}
	s.world.questList(s.owner)
	if !s.custom && cond > 0 {
		s.world.questMark(s.owner, s.quest)
	}
}

// Unset removes a variable.
func (s *QuestState) Unset(name string) {
	if _, ok := s.vars[name]; !ok {
		return
	}
	delete(s.vars, name)
	if name == "cond" {
		s.cond = 0
	}
}

// Exit is Mobius exitQuest after the quest items are gone.
func (s *QuestState) Exit(repeatable bool) {
	if repeatable {
		delete(s.owner.quests, s.quest)
		s.exited = true
		s.world.questList(s.owner)
	} else {
		s.SetState(scripting.StateCompleted)
	}
	s.vars = nil
}
