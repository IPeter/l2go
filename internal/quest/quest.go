// Package quest is the quest state model of decision Q3: per character and
// quest a state, a step (cond) and string variables, with the rules of the
// Mobius QuestState class, and the conversion to and from the stored quest
// documents (decision Q4). It knows nothing about Lua or the world: the game
// tells it about quests through a Lookup and hears about packets, sounds and
// quest items through a Listener.
//
// A Book holds one character's quests. Everything in this package is
// loop-only: Books and States are not safe for concurrent use.
package quest

// StateID is the state of a quest for a character (Mobius State).
type StateID int8

// Quest states. The values are Mobius's (State.CREATED, STARTED, COMPLETED).
const (
	Created   StateID = 0 // the state exists but the quest is not accepted
	Started   StateID = 1 // the quest is in progress
	Completed StateID = 2 // the quest is done
)

// Name returns the state's stored name (Mobius State.getStateName): "Start",
// "Started" or "Completed". Unknown states are "Start".
func (s StateID) Name() string {
	switch s {
	case Started:
		return "Started"
	case Completed:
		return "Completed"
	default:
		return "Start"
	}
}

// String returns the stored name.
func (s StateID) String() string { return s.Name() }

// StateFromName returns the state with this stored name (Mobius
// State.getStateId): Created for anything but "Started" and "Completed".
func StateFromName(name string) StateID {
	switch name {
	case "Started":
		return Started
	case "Completed":
		return Completed
	default:
		return Created
	}
}

// Type is how a quest ends (Mobius QuestType), for State.ExitQuestType.
type Type int8

// Quest types. The values are the Mobius ordinals.
const (
	Repeatable Type = 0 // the state is deleted: the quest can be taken again
	OneTime    Type = 1 // the state stays COMPLETED
	Daily      Type = 2 // COMPLETED until the next reset (ResetHour:ResetMinute)
)

// Well-known variable names (Mobius QuestState and Quest).
const (
	VarState     = "<state>"             // the state row of a stored quest; never in State's variables
	VarCond      = "cond"                // the quest step
	VarFlags     = "__compltdStateFlags" // the steps done when steps were skipped
	VarMemo      = "memoState"
	VarMemoEx    = "memoStateEx" // followed by the slot number
	VarRestart   = "restartTime" // epoch milliseconds when a daily quest can be taken again
	MaxVarName   = 20            // longest variable name the L2J character_quests table holds
	MaxVarValue  = 255           // longest value it holds
	ResetHour    = 6             // daily quests reset at 06:30 server time
	ResetMinute  = 30
	MaxListID    = 19999 // quests with IDs above are not listed (Mobius getAllActiveQuests)
	SoundAccept  = "ItemSound.quest_accept"
	SoundMiddle  = "ItemSound.quest_middle"
	SoundFinish  = "ItemSound.quest_finish"
	flagsSkipped = int32(-0x7fffffff) // 0x80000001: the "steps skipped" bit and step 1
)

// Info is what the model needs to know about a quest script.
type Info struct {
	ID     int32  // the quest ID; 1-19999 for listed quests, -1 for plain scripts
	Name   string // the script name, which keys the states
	Custom bool   // Mobius isCustomQuest: no quest mark, startQuest does nothing
}

// Lookup returns the quest script with this name, as currently loaded. Like
// Mobius QuestState.getQuest, the model looks the quest up on every use, so a
// reloaded script takes effect at once.
type Lookup func(name string) (Info, bool)

// Listener receives what a Book does that the player must see or that
// touches the rest of the game, synchronously and in Mobius's order. A nil
// Listener in NewBook ignores everything.
type Listener interface {
	// QuestList asks for a QuestList packet.
	QuestList()
	// QuestMark asks for an ExShowQuestMark packet of the quest.
	QuestMark(questID int32)
	// PlaySound asks for a PlaySound packet of a quest sound (SoundAccept,
	// SoundMiddle, SoundFinish).
	PlaySound(sound string)
	// RemoveDeathNotify runs first in every ExitQuest: the game stops
	// notifying st of deaths (Mobius removeNotifyQuestOfDeath).
	RemoveDeathNotify(st *State)
	// RemoveQuestItems runs in ExitQuest of a started quest before its state
	// changes: the game takes the quest's registered items (Mobius
	// removeRegisteredQuestItems).
	RemoveQuestItems(st *State)
	// Changed runs when the Book gets changes to save after it had none: the
	// game schedules a save (quests.saveDelay).
	Changed()
}

// NopListener is a Listener that does nothing, to embed in listeners that
// need only some of the calls.
type NopListener struct{}

func (NopListener) QuestList()               {}
func (NopListener) QuestMark(int32)          {}
func (NopListener) PlaySound(string)         {}
func (NopListener) RemoveDeathNotify(*State) {}
func (NopListener) RemoveQuestItems(*State)  {}
func (NopListener) Changed()                 {}

var _ Listener = NopListener{}

// CheckVar reports whether a variable fits the L2J character_quests columns
// (name ≤ MaxVarName, value ≤ MaxVarValue characters). The model stores any
// variable, as Mobius keeps it in memory; callers can warn about the rest.
func CheckVar(name, value string) bool {
	return len([]rune(name)) <= MaxVarName && len([]rune(value)) <= MaxVarValue
}
