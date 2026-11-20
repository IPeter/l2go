// Package scenario holds the refdiff scenarios: fixed client sessions that
// run the same way against L2Go and against the L2J Mobius reference
// server, recording every packet on the way.
//
// A scenario drives one or more players (roles such as "alice" and "bob").
// Each run gives every role a fresh random account and character name, so
// a scenario can run again and again against the same servers. Run returns
// a Capture: the packets of every login and game connection in order, and
// what the normalizer needs to know (names, session keys, server ID).
// Package recording turns it into a comparable recording.
//
// A scenario that needs items or a place (a shop, a warehouse) sets its
// characters up first with GM commands in a session of their own
// (Session.Prepare), whose connections stay out of the capture; such a
// scenario is marked GM and needs GM rights for the generated characters
// on both servers. Each scenario also names its default ignore patterns,
// the noise any server shows in it.
package scenario

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/testclient"
)

// DefaultStepTimeout bounds each step of a scenario when Options does not
// set one.
const DefaultStepTimeout = 20 * time.Second

// Target is one server stack a scenario runs against.
type Target struct {
	// Label names the side in reports, such as "l2go" or "reference".
	Label string
	// LoginAddr is the login server address.
	LoginAddr string
	// ServerID is the game server to play on.
	ServerID byte
	// GameAddr overrides the game server address the server list gives.
	GameAddr string
	// Password is the password of the generated accounts. Both servers
	// create unknown accounts on their first login.
	Password string
}

// Source describes where the target is, for reports.
func (t Target) Source() string {
	s := fmt.Sprintf("login %s, server %d", t.LoginAddr, t.ServerID)
	if t.GameAddr != "" {
		s += ", game " + t.GameAddr
	}
	return s
}

// Scenario is a named client session.
type Scenario struct {
	Name        string
	Description string
	// Roles are the players of the scenario, in the order they first act.
	Roles []string
	// Run drives the players through the session. It returns the first
	// error; the packets recorded until then are kept.
	Run func(ctx context.Context, s *Session) error
	// Ignore lists the comparison's default ignore patterns (see
	// recording.Options.Ignore): what is noise in this scenario on any
	// server, such as where wandering NPCs stand. refdiff adds the -ignore
	// patterns to them.
	Ignore []string
	// GM marks a scenario whose setup uses GM commands (//create_item,
	// //move_to, //kill): its characters need GM rights on both servers.
	// Run fails at the first GM step otherwise.
	GM bool
	// Sample marks a scenario made for refdiff -sample: it swings at
	// monsters until Options.Swings attacks are recorded.
	Sample bool
}

// All returns every scenario, in a fixed order.
func All() []Scenario { return slices.Clone(scenarios) }

// Find returns the scenario called name.
func Find(name string) (Scenario, bool) {
	for _, sc := range scenarios {
		if sc.Name == name {
			return sc, true
		}
	}
	return Scenario{}, false
}

// Identity is the account and character a role played with in one run.
type Identity struct {
	Role    string
	Account string
	Char    string
}

// Packet is one packet of a stream, in the order the client sent or read
// it.
type Packet struct {
	// Time is when the client sent or read the packet; zero for login
	// packets.
	Time     time.Time
	Dir      testclient.Direction
	Name     string
	Opcode   byte
	ExOpcode uint16
	// Raw is the decrypted game packet, opcode included. It is nil for
	// login packets, which are recorded decoded only.
	Raw []byte
	// Value is the decoded server packet: one of the testclient game packet
	// types (Unknown when the client does not parse it) or a login packet
	// such as testclient.Init or testclient.ServerListResult. It is nil for
	// client packets and for server packets that could not be decoded.
	Value any
	// Err is why a malformed server packet could not be decoded.
	Err error
}

// Stream is the packets of one connection.
type Stream struct {
	// Name identifies the connection across servers and runs: "login
	// alice", "game alice", and "game alice 2" for the second game
	// connection of the role.
	Name    string
	Packets []Packet
}

// Capture is everything one scenario run recorded.
type Capture struct {
	Scenario string
	Target   Target
	// Players are the roles with the names they used, in the order they
	// joined the scenario.
	Players []Identity
	// Keys are the session keys of every login, SessionIDs the session IDs
	// of every login server Init.
	Keys       []session.Keys
	SessionIDs []int32
	// Streams are the connections in the order they were opened.
	Streams []Stream
	// Notes are remarks of the scenario, such as which NPC it talked to.
	Notes []string
	// Err is the error that ended the scenario early, or nil.
	Err error
}

// Options tune Run.
type Options struct {
	// StepTimeout bounds each step; 0 means DefaultStepTimeout.
	StepTimeout time.Duration
	// Rand makes the account and character names; nil means a random seed.
	Rand *rand.Rand
	// Logger receives a debug line per step; nil means no logging.
	Logger *slog.Logger
	// Swings is how many attacks a Sample scenario records; 0 means
	// DefaultSwings.
	Swings int
}

// DefaultSwings is the number of attacks a Sample scenario records when
// Options does not set one (Plan 09: at least 300 per pairing).
const DefaultSwings = 300

// Run runs sc against t and returns what it recorded. It always returns a
// capture; Capture.Err tells whether the scenario finished. Every
// connection is closed when Run returns.
func Run(ctx context.Context, sc Scenario, t Target, opts Options) *Capture {
	s := newSession(t, opts)
	for _, role := range sc.Roles {
		s.Player(role)
	}
	var err error
	if sc.Run == nil {
		err = errors.New("scenario has no Run function")
	} else {
		err = sc.Run(ctx, s)
	}
	s.close()
	return s.capture(sc.Name, err)
}

// Session is the state of one scenario run: its players, connections and
// recordings. Scenarios use it from one goroutine.
type Session struct {
	target      Target
	stepTimeout time.Duration
	rng         *rand.Rand
	logger      *slog.Logger
	swings      int

	// setup is set while Prepare runs: the connections opened meanwhile
	// are left out of the capture.
	setup   bool
	players []*Player
	streams []*stream
	// opened counts the connections per kind and role, for stream names.
	opened     map[string]int
	keys       []session.Keys
	sessionIDs []int32
	notes      []string
	conns      []*testclient.GameConn
}

func newSession(t Target, opts Options) *Session {
	s := &Session{target: t, stepTimeout: opts.StepTimeout, rng: opts.Rand, logger: opts.Logger, swings: opts.Swings}
	if s.stepTimeout <= 0 {
		s.stepTimeout = DefaultStepTimeout
	}
	if s.swings <= 0 {
		s.swings = DefaultSwings
	}
	if s.rng == nil {
		s.rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) //nolint:gosec // names, not secrets
	}
	if s.logger == nil {
		s.logger = slog.New(slog.DiscardHandler)
	}
	return s
}

// Target returns the server stack the session plays on.
func (s *Session) Target() Target { return s.target }

// Player returns the player of role, creating it with fresh names the
// first time.
func (s *Session) Player(role string) *Player {
	for _, p := range s.players {
		if p.Role == role {
			return p
		}
	}
	p := &Player{Identity: newIdentity(s.rng, role), s: s}
	s.players = append(s.players, p)
	return p
}

// Note adds a remark to the capture, such as which NPC the scenario chose.
func (s *Session) Note(format string, args ...any) {
	s.notes = append(s.notes, fmt.Sprintf(format, args...))
}

// Step is one named action of a scenario.
type Step struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Do runs one step with the step timeout. The error of a failed step
// names it.
func (s *Session) Do(ctx context.Context, step Step) error {
	s.logger.Debug("refdiff step", "server", s.target.Label, "step", step.Name)
	ctx, cancel := context.WithTimeout(ctx, s.stepTimeout)
	defer cancel()
	if err := step.Fn(ctx); err != nil {
		return fmt.Errorf("step %q: %w", step.Name, err)
	}
	return nil
}

// Steps runs steps in order and stops at the first error.
func (s *Session) Steps(ctx context.Context, steps ...Step) error {
	for _, st := range steps {
		if err := s.Do(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

// Enter takes p from the login server into the world: login, game server
// connection, a new Human Fighter when the account has no character of
// p's name, character selection and EnterWorld.
func (s *Session) Enter(ctx context.Context, p *Player) error {
	return s.Steps(ctx,
		Step{p.Role + " logs in", p.Login},
		Step{p.Role + " connects to the game server", p.Connect},
		Step{p.Role + " creates a Human Fighter", p.Create},
		Step{p.Role + " selects the character", p.Select},
		Step{p.Role + " enters the world", p.EnterWorld},
	)
}

// Prepare runs a setup for p that is not compared: p enters the world, the
// steps run (GM commands that give items or move the character) and p logs
// out. Its connections are recorded as "setup login alice" and "setup game
// alice" and left out of the capture, so that the scenario proper starts
// with a fresh login, the same on both servers. The session keys still
// reach the normalizer.
func (s *Session) Prepare(ctx context.Context, p *Player, steps ...Step) error {
	s.setup = true
	defer func() { s.setup = false }()
	if err := s.Enter(ctx, p); err != nil {
		return fmt.Errorf("setup: %w", err)
	}
	all := append(slices.Clone(steps), Step{p.Role + " logs out after the setup", p.Logout})
	if err := s.Steps(ctx, all...); err != nil {
		return fmt.Errorf("setup: %w", err)
	}
	return nil
}

// Swings returns how many attacks a Sample scenario records.
func (s *Session) Swings() int { return s.swings }

// Idle is a step that waits for d, while the background readers keep
// recording.
func Idle(d time.Duration) Step {
	return Step{fmt.Sprintf("idle %v", d), func(ctx context.Context) error { return sleep(ctx, d) }}
}

// Long gives st the time limit d instead of the step timeout, for steps
// that take long by nature, such as a fight or a walk across a field.
// Cancelling the scenario (its parent context) still stops the step.
func Long(d time.Duration, st Step) Step {
	fn := st.Fn
	st.Fn = func(ctx context.Context) error {
		lctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), d)
		defer cancel()
		stop := context.AfterFunc(ctx, func() {
			// The step timeout is what Long replaces; anything else (a
			// cancelled run) ends the step.
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				cancel()
			}
		})
		defer stop()
		return fn(lctx)
	}
	return st
}

// sleep waits for d or until ctx ends.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// stream is a connection being recorded: login packets are added one by
// one, game packets come from the recorder.
type stream struct {
	name    string
	packets []Packet
	rec     *testclient.Recorder
	// setup marks a connection of Prepare, left out of the capture.
	setup bool
}

// newStream starts the stream of a new connection of kind ("login" or
// "game") for role. During Prepare the kind gets the prefix "setup ".
func (s *Session) newStream(kind, role string) *stream {
	if s.setup {
		kind = "setup " + kind
	}
	base := kind + " " + role
	if s.opened == nil {
		s.opened = make(map[string]int)
	}
	s.opened[base]++
	name := base
	if n := s.opened[base]; n > 1 {
		name = fmt.Sprintf("%s %d", base, n)
	}
	st := &stream{name: name, setup: s.setup}
	s.streams = append(s.streams, st)
	return st
}

// server records a decoded login server packet.
func (st *stream) server(name string, opcode byte, v any) {
	st.packets = append(st.packets, Packet{Dir: testclient.FromServer, Name: name, Opcode: opcode, Value: v})
}

// client records a login client packet by name.
func (st *stream) client(name string, opcode byte) {
	st.packets = append(st.packets, Packet{Dir: testclient.FromClient, Name: name, Opcode: opcode})
}

// close closes every game connection and waits for their readers.
func (s *Session) close() {
	for _, c := range s.conns {
		_ = c.Close()
	}
}

// capture collects the recordings.
func (s *Session) capture(name string, err error) *Capture {
	c := &Capture{
		Scenario:   name,
		Target:     s.target,
		Keys:       s.keys,
		SessionIDs: s.sessionIDs,
		Notes:      s.notes,
		Err:        err,
	}
	for _, p := range s.players {
		c.Players = append(c.Players, p.Identity)
	}
	for _, st := range s.streams {
		if st.setup {
			continue
		}
		out := Stream{Name: st.name, Packets: st.packets}
		if st.rec != nil {
			for _, r := range st.rec.Records() {
				out.Packets = append(out.Packets, fromRecord(r))
			}
		}
		c.Streams = append(c.Streams, out)
	}
	return c
}

// fromRecord converts a testclient record.
func fromRecord(r testclient.Record) Packet {
	p := Packet{Time: r.Time, Dir: r.Dir, Name: r.Name, Opcode: r.Opcode, ExOpcode: r.ExOpcode, Raw: r.Raw, Err: r.Err}
	if r.Packet != nil {
		p.Value = r.Packet
	}
	return p
}
