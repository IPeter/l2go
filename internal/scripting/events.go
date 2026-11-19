package scripting

import "slices"

// Event is something that happened in the game, sent to the scripts.
type Event struct {
	// Name selects the handlers, for example EventCharacterCreate.
	Name string
	// Data becomes the event table passed to every handler. Build it with
	// the typed events (CharacterCreate, ChatMessage); they wrap player
	// refs so handlers get Player values.
	Data map[string]any
}

// Result is what the handlers of one event decided.
type Result struct {
	// Rejected is set when a handler called events.reject, with the reason
	// name it gave (character_create).
	Rejected bool
	Reason   string
	// Dropped is set when a chat handler dropped the message by returning
	// false or an empty text.
	Dropped bool
	// Text is the chat text to send: the original text, or the last
	// replacement a handler returned. It is set for chat events only, also
	// when no handler ran. The game layer still applies its own length
	// limit.
	Text string
	// Err joins the errors of failed handlers (Lua errors, timeouts). They
	// are already logged. A failed handler does not change the result. Err
	// is also ErrStopped or the context's error when the event did not run.
	Err error
}

// EventCharacterCreate runs before a new character is accepted. Handlers
// may reject it with one of Reasons(EventCharacterCreate).
const EventCharacterCreate = "character_create"

// EventChat runs before a chat message is sent. Handlers may drop it or
// replace its text; see ChatMessage.
const EventChat = "chat"

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
	// filter events have a "text" field: a handler may return a new text,
	// which later handlers see, or false to drop the event.
	filter bool
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
	{name: EventChat, filter: true},
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

// baseResult is the result of ev before any handler ran: for chat, the
// text unchanged.
func baseResult(ev Event) Result {
	var r Result
	if s, ok := specOf(ev.Name); ok && s.filter {
		r.Text, _ = ev.Data["text"].(string)
	}
	return r
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

// Event returns the event to pass to Engine.Invoke.
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

// ChatMessage is a chat message about to be sent, as handlers of
// events.on_chat see it. The game layer fires it after its own checks
// (length, type, whisper target) and sends Result.Text unless
// Result.Dropped.
type ChatMessage struct {
	// Player wrote the message.
	Player PlayerRef
	// Type is the chat type the client sent (Mobius ChatType client IDs:
	// 0 general, 1 shout, 2 whisper, 3 party, 4 clan, 8 trade, 9 alliance,
	// 17 hero voice, ...). Scripts see its name, for example "shout".
	Type int
	// Text is the message as written.
	Text string
	// Target is the name a whisper goes to; empty for other types.
	Target string
}

// Event returns the event to pass to Engine.Invoke.
func (c ChatMessage) Event() Event {
	data := map[string]any{
		"player": playerArg{c.Player},
		"type":   chatTypeName(c.Type),
		"text":   c.Text,
	}
	if c.Target != "" {
		data["target"] = c.Target
	}
	return Event{Name: EventChat, Data: data}
}

// chatTypes are the names scripts see for the chat types, indexed by the
// client ID: Mobius's ChatType names in lower case.
var chatTypes = []string{
	"general",             // 0
	"shout",               // 1
	"whisper",             // 2
	"party",               // 3
	"clan",                // 4
	"gm",                  // 5
	"petition_player",     // 6
	"petition_gm",         // 7
	"trade",               // 8
	"alliance",            // 9
	"announcement",        // 10
	"boat",                // 11
	"friend",              // 12
	"msnchat",             // 13
	"partymatch_room",     // 14
	"partyroom_commander", // 15
	"partyroom_all",       // 16
	"hero_voice",          // 17
	"critical_announce",   // 18
	"screen_announce",     // 19
	"battlefield",         // 20
	"mpcc_room",           // 21
}

// chatTypeUnknown is the name of a chat type ID outside chatTypes.
const chatTypeUnknown = "unknown"

func chatTypeName(id int) string {
	if id >= 0 && id < len(chatTypes) {
		return chatTypes[id]
	}
	return chatTypeUnknown
}
