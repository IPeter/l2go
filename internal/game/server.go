package game

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Errors of the game server.
var (
	// ErrNoCharacter means the selected slot has no character. Mobius sends
	// nothing back then.
	ErrNoCharacter = errors.New("game: no character in that slot")
	// ErrAlreadySelected means the session selected a character before; a
	// repeated CharacterSelect is ignored, as in Mobius.
	ErrAlreadySelected = errors.New("game: the session has already selected a character")
	// ErrCharacterBanned means the selected character's access level is
	// below 0; the connection is closed with ServerClose.
	ErrCharacterBanned = errors.New("game: the character is banned")
	// ErrRunning means Run was called twice.
	ErrRunning = errors.New("game: server already running")
	// ErrDuplicate is returned when a handler, command or feature is
	// registered twice.
	ErrDuplicate = errors.New("game: already registered")
	// ErrNotInWorld means the player is not in the world (it is teleporting
	// or has left).
	ErrNotInWorld = errors.New("game: player is not in the world")
)

const (
	// saveTimeout bounds one write of a character.
	saveTimeout = 10 * time.Second
	// shutdownTimeout bounds saving everyone when the server stops.
	shutdownTimeout = 30 * time.Second
	// restartListTimeout bounds sending the character list after a restart.
	restartListTimeout = 10 * time.Second
)

// Settings are the game rules of this server, from its config.GameServer
// with the Mobius defaults for what the config leaves out.
type Settings struct {
	// TickInterval is the world loop's tick.
	TickInterval time.Duration
	// AutosaveInterval is how often a player in the world is saved; 0 saves
	// only when the player leaves.
	AutosaveInterval time.Duration
	// DeleteCharAfterDays is how long a character marked for deletion can
	// be restored.
	DeleteCharAfterDays int
	// MaxRunSpeed caps the run speed of players.
	MaxRunSpeed int
	// SpawnProtection is how long a player that entered the world is safe
	// from aggressive monsters; 0 turns it off.
	SpawnProtection time.Duration
	Chat            config.Chat
	NPCs            config.NPCs
	// GeodataDir is the geodata folder, empty without geodata.
	GeodataDir string
	// DataDir is the game data folder; HTMLDir is its html subfolder, where
	// the NPC dialogs are.
	DataDir, HTMLDir string
}

// settingsFrom reads the settings of gs, filling in the defaults for zero
// values that would break the server.
func settingsFrom(gs config.GameServer) Settings {
	w := gs.World
	def := config.DefaultWorld()
	s := Settings{
		TickInterval:        time.Duration(w.TickInterval),
		AutosaveInterval:    max(time.Duration(w.AutosaveInterval), 0),
		DeleteCharAfterDays: w.DeleteCharAfterDays,
		MaxRunSpeed:         w.MaxRunSpeed,
		SpawnProtection:     max(time.Duration(w.SpawnProtection), 0),
		Chat:                gs.Chat,
		NPCs:                gs.NPCs,
		GeodataDir:          gs.Geodata.Dir,
		DataDir:             gs.DataDir,
		HTMLDir:             filepath.Join(gs.DataDir, "html"),
	}
	if s.TickInterval <= 0 {
		s.TickInterval = time.Duration(def.TickInterval)
	}
	if s.MaxRunSpeed <= 0 {
		s.MaxRunSpeed = def.MaxRunSpeed
	}
	if s.Chat == (config.Chat{}) {
		s.Chat = config.DefaultChat()
	}
	if s.NPCs == (config.NPCs{}) {
		s.NPCs = config.DefaultNPCs()
	}
	return s
}

// Server is the game on the world loop. It implements Service for the
// session layer, and the API the game features build on: the registries
// (Register, RegisterGMCommand, ...), the hooks (OnPlayerEnter, ...), the
// world helpers (AddObject, MoveObject, Teleport, ...) and the accessors.
//
// The fields of a Server and its players belong to the loop, except where a
// method says otherwise.
type Server struct {
	log       *slog.Logger
	gs        config.GameServer
	settings  Settings
	data      *data.Data
	worldData *data.World
	chars     storage.CharacterStore
	ids       storage.IDAllocator
	scripts   *scripting.Engine
	output    *ScriptOutput
	geo       geo.Engine
	rules     *stats.Rules
	classes   map[int]*stats.ClassTemplate // stats templates by class ID
	loop      *world.Loop
	world     *world.World
	saver     *saver

	started atomic.Bool
	// selecting serializes SelectCharacter per account.
	selecting keyedMutex
	// bg is the context of background work (saves, the character list after
	// a restart). It outlives Run's context, so the last saves still finish.
	bg atomic.Pointer[context.Context]

	// Loop-only.
	bySession    map[Session]*Player // every selected player
	byAccount    map[string]*Player  // the same, by account
	byName       map[string]*Player  // players in the game, by storage.NameKey
	handlers     [256]Handler
	exHandlers   map[uint16]Handler
	gmCommands   map[string]gmCommand
	userCommands map[int32]UserCommand
	hooks        hooks

	// Per-feature state, loop-only. Each type is defined in its feature's
	// own file (movement.go, npc.go, dialog.go), so features never need to
	// edit this struct.
	movement movementFeature
	npc      npcFeature
	dialog   dialogFeature
}

var _ Service = (*Server)(nil)

// New returns the game server for opts.ServerID. Run starts its world loop.
// The features of the package (movement, chat, NPCs, ...) register their
// handlers and hooks while New runs.
func New(opts Options) (*Server, error) {
	switch {
	case opts.Data == nil:
		return nil, errors.New("game: no game data")
	case opts.Characters == nil:
		return nil, errors.New("game: no character store")
	}
	gs, ok := opts.Config.GameServerByID(opts.ServerID)
	if !ok {
		return nil, fmt.Errorf("game: no game server with id %d in the config", opts.ServerID)
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	settings := settingsFrom(gs)
	rules, err := newRules(opts.Data, settings.MaxRunSpeed)
	if err != nil {
		return nil, err
	}
	classes, err := statsClassTemplates(opts.Data)
	if err != nil {
		return nil, err
	}

	loop := world.NewLoop(world.LoopOptions{
		Clock:        opts.Clock,
		TickInterval: settings.TickInterval,
		Logger:       logger,
		CheckOwner:   opts.CheckLoopOwner,
	})
	geoEngine := opts.Geo
	if geoEngine == nil {
		geoEngine = geo.None{}
	}
	output := opts.ScriptOutput
	if output == nil {
		output = NewScriptOutput()
	}
	s := &Server{
		log:          logger,
		gs:           gs,
		settings:     settings,
		data:         opts.Data,
		worldData:    opts.World,
		chars:        opts.Characters,
		ids:          opts.IDs,
		scripts:      opts.Scripts,
		output:       output,
		geo:          geoEngine,
		rules:        rules,
		classes:      classes,
		loop:         loop,
		world:        world.New(loop),
		saver:        newSaver(opts.Characters, logger),
		bySession:    make(map[Session]*Player),
		byAccount:    make(map[string]*Player),
		byName:       make(map[string]*Player),
		exHandlers:   make(map[uint16]Handler),
		gmCommands:   make(map[string]gmCommand),
		userCommands: make(map[int32]UserCommand),
	}
	bg := context.Background()
	s.bg.Store(&bg)
	s.world.SetUpdateFlush(s.flushUpdate)
	if err := s.registerCore(); err != nil {
		return nil, err
	}
	for _, f := range sortedFeatures() {
		if err := f.setup(s); err != nil {
			return nil, fmt.Errorf("game: feature %s: %w", f.name, err)
		}
	}
	output.bind(s)
	return s, nil
}

// Run runs the world loop and the background saver until ctx is done. Then
// it saves and removes every player (the sessions get ServerClose), waits
// for the saves to finish, and returns nil. Run may be called once.
func (s *Server) Run(ctx context.Context) error {
	if !s.started.CompareAndSwap(false, true) {
		return ErrRunning
	}
	bg := context.WithoutCancel(ctx)
	s.bg.Store(&bg)

	saverDone := make(chan struct{})
	go func() {
		defer close(saverDone)
		s.saver.run(bg)
	}()

	loopCtx, stopLoop := context.WithCancel(bg)
	defer stopLoop()
	loopDone := make(chan error, 1)
	go func() { loopDone <- s.loop.Run(loopCtx) }()
	s.log.Info("game world running", "tick", s.settings.TickInterval, "autosave", s.settings.AutosaveInterval)

	var runErr error
	select {
	case <-ctx.Done():
		sctx, cancel := context.WithTimeout(bg, shutdownTimeout)
		if err := s.loop.Call(sctx, s.shutdown); err != nil {
			s.log.Error("game world: cannot save the players at shutdown", "err", err)
		}
		cancel()
		stopLoop()
		runErr = <-loopDone // nil: the loop stops quietly when its context ends
	case runErr = <-loopDone:
	}

	s.saver.close()
	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()
	select {
	case <-saverDone:
	case <-timer.C:
		s.log.Error("game world: saves still running at shutdown; giving up on them")
	}
	s.log.Info("game world stopped")
	return runErr
}

// background returns the context for background work.
func (s *Server) background() context.Context { return *s.bg.Load() }

// OnLoop runs fn on the world loop and waits for it (Service). On the loop it
// runs fn at once.
func (s *Server) OnLoop(ctx context.Context, fn func()) error { return s.loop.Call(ctx, fn) }

// Scripts returns the Lua engine, or nil without one (Service). Call it only
// on the loop.
func (s *Server) Scripts() *scripting.Engine { return s.scripts }

// SwapScripts puts reloaded scripts in place on the loop. It is the swap
// function for scripting.Engine.Watch and safe on any goroutine:
//
//	engine.Watch(ctx, scripting.DefaultDebounce, srv.SwapScripts)
func (s *Server) SwapScripts(p *scripting.Prepared) {
	if s.scripts == nil {
		p.Close()
		return
	}
	if !s.loop.Post(func() { s.scripts.Swap(p) }) {
		p.Close()
	}
}

// ScriptOutput returns the adapter that carries what scripts do to the
// world (scripting.Options.Output). Safe on any goroutine.
func (s *Server) ScriptOutput() *ScriptOutput { return s.output }

// Loop returns the world loop. Safe on any goroutine.
func (s *Server) Loop() *world.Loop { return s.loop }

// World returns the world. Its methods are loop-only, except Loop, GameTime
// and IDs. Safe on any goroutine.
func (s *Server) World() *world.World { return s.world }

// Clock returns the loop's clock. Safe on any goroutine.
func (s *Server) Clock() world.Clock { return s.loop.Clock() }

// Now returns the loop's time (world.Loop.Now). Loop-only.
func (s *Server) Now() time.Time { return s.loop.Now() }

// Data returns the player and item data. Safe on any goroutine.
func (s *Server) Data() *data.Data { return s.data }

// WorldData returns the world data (NPCs, spawns, teleporters, zones, map
// regions), or nil when none was loaded. Safe on any goroutine.
func (s *Server) WorldData() *data.World { return s.worldData }

// Geo returns the geodata engine. Safe on any goroutine.
func (s *Server) Geo() geo.Engine { return s.geo }

// Rules returns the stat rules every stat set of the server shares. Safe on
// any goroutine.
func (s *Server) Rules() *stats.Rules { return s.rules }

// Settings returns the game rules of the server. Safe on any goroutine.
func (s *Server) Settings() Settings { return s.settings }

// Config returns the configuration of this game server. Safe on any
// goroutine.
func (s *Server) Config() config.GameServer { return s.gs }

// Logger returns the server's logger. Safe on any goroutine.
func (s *Server) Logger() *slog.Logger { return s.log }

// IDs returns the allocator of persistent object IDs (characters, items),
// or nil. Safe on any goroutine.
func (s *Server) IDs() storage.IDAllocator { return s.ids }

// Characters returns the character store. Use it only off the loop, or
// through Save: the loop must never wait for the database. Safe on any
// goroutine.
func (s *Server) Characters() storage.CharacterStore { return s.chars }

// Player returns the player in the game with the given object ID.
// Loop-only.
func (s *Server) Player(id world.ObjectID) (*Player, bool) {
	v, ok := s.world.Player(id)
	if !ok {
		return nil, false
	}
	p, ok := v.(*Player)
	return p, ok
}

// PlayerByName returns the player in the game with the given name, ignoring
// case (storage.NameKey). Loop-only.
func (s *Server) PlayerByName(name string) (*Player, bool) {
	p, ok := s.byName[storage.NameKey(name)]
	return p, ok
}

// Players yields the players in the game, in or out of the world (a
// teleporting player is out of it). Don't add or remove players while
// iterating. Loop-only.
func (s *Server) Players() iter.Seq[*Player] {
	return func(yield func(*Player) bool) {
		for v := range s.world.Players() {
			if p, ok := v.(*Player); ok && !yield(p) {
				return
			}
		}
	}
}

// PlayerCount returns the number of players in the game. Loop-only.
func (s *Server) PlayerCount() int { return s.world.PlayerCount() }

// playerOf returns the player of a session, or nil.
func (s *Server) playerOf(sess Session) *Player { return s.bySession[sess] }

// safely runs fn and logs a panic instead of letting it end the command that
// called it, so that one failing hook does not break, say, entering the
// world.
func (s *Server) safely(what string, fn func()) {
	defer func() {
		if v := recover(); v != nil {
			s.log.Error("game: "+what+" panicked", "panic", v)
		}
	}()
	fn()
}

// feature is a part of the game that plugs into the server; see
// registerFeature.
type feature struct {
	name  string
	setup func(s *Server) error
}

var (
	featuresMu sync.Mutex
	features   []feature
)

// registerFeature adds a feature that New sets up, after the core and in
// name order. Each feature file registers itself from an init function, so
// that features need no central list:
//
//	func init() { registerFeature("movement", setupMovement) }
//
//	func setupMovement(s *Server) error {
//		return s.Register(clientpackets.OpMoveToLocation, s.handleMoveToLocation)
//	}
//
// setup runs before the loop starts; it may register handlers, commands and
// hooks, and use the loop-only world methods.
func registerFeature(name string, setup func(s *Server) error) {
	featuresMu.Lock()
	defer featuresMu.Unlock()
	features = append(features, feature{name: name, setup: setup})
}

// sortedFeatures returns the registered features in name order.
func sortedFeatures() []feature {
	featuresMu.Lock()
	defer featuresMu.Unlock()
	fs := slices.Clone(features)
	slices.SortStableFunc(fs, func(a, b feature) int { return cmp.Compare(a.name, b.name) })
	return fs
}

// keyedMutex is a set of mutexes by key, created on demand and dropped when
// nobody holds or waits for them. The zero value is ready to use.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int // holders and waiters
}

// lock locks the mutex of key and returns the function that unlocks it.
func (k *keyedMutex) lock(key string) (unlock func()) {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = make(map[string]*keyedLock)
	}
	l := k.locks[key]
	if l == nil {
		l = &keyedLock{}
		k.locks[key] = l
	}
	l.refs++
	k.mu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		k.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
