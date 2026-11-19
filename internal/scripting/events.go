package scripting

import "slices"

// Event is something that happened in the game, sent to the scripts.
type Event struct {
	// Name selects the handlers, for example EventCharacterCreate.
	Name string
	// Data becomes the event table passed to every handler.
	Data map[string]any
}

// Result is what the handlers of one event decided.
type Result struct {
	// Rejected is set when a handler called events.reject, with the reason
	// name it gave.
	Rejected bool
	Reason   string
	// Err joins the errors of failed handlers (Lua errors, timeouts). They
	// are already logged. A failed handler does not change the result.
	Err error
}

// EventCharacterCreate runs before a new character is accepted. Handlers
// may reject it with one of Reasons(EventCharacterCreate).
const EventCharacterCreate = "character_create"

// Reason names for rejecting character creation. Scripts pass them to
// events.reject; the game server maps them to protocol codes.
const (
	ReasonCreationFailed     = "creation_failed"
	ReasonTooManyCharacters  = "too_many_characters"
	ReasonNameExists         = "name_exists"
	ReasonNameTooLong        = "name_too_long"
	ReasonIncorrectName      = "incorrect_name"
	ReasonCreationNotAllowed = "creation_not_allowed"
)

// eventSpec describes an event scripts can handle.
type eventSpec struct {
	// name is the event name. Handlers register with events.on_<name>.
	name string
	// reasons are the names events.reject accepts. Empty means the event
	// cannot be rejected.
	reasons []string
}

// eventSpecs lists every event. Adding one here adds events.on_<name>.
var eventSpecs = []eventSpec{
	{
		name: EventCharacterCreate,
		reasons: []string{
			ReasonCreationFailed,
			ReasonTooManyCharacters,
			ReasonNameExists,
			ReasonNameTooLong,
			ReasonIncorrectName,
			ReasonCreationNotAllowed,
		},
	},
}

func specOf(name string) (*eventSpec, bool) {
	i := slices.IndexFunc(eventSpecs, func(s eventSpec) bool { return s.name == name })
	if i < 0 {
		return nil, false
	}
	return &eventSpecs[i], true
}

// Reasons returns the reason names events.reject accepts for an event.
func Reasons(event string) []string {
	if s, ok := specOf(event); ok {
		return slices.Clone(s.reasons)
	}
	return nil
}

// CharacterCreate is a character about to be created, as handlers of
// events.on_character_create see it.
type CharacterCreate struct {
	Account   string
	Name      string
	Race      int
	Sex       int
	ClassID   int
	HairStyle int
	HairColor int
	Face      int
}

// Event returns the event to pass to Engine.Call.
func (c CharacterCreate) Event() Event {
	return Event{Name: EventCharacterCreate, Data: map[string]any{
		"account":    c.Account,
		"name":       c.Name,
		"race":       c.Race,
		"sex":        c.Sex,
		"class_id":   c.ClassID,
		"hair_style": c.HairStyle,
		"hair_color": c.HairColor,
		"face":       c.Face,
	}}
}
