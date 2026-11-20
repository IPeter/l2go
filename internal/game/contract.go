// This file is the contract between the session layer (internal/gameserver)
// and the game: the Session and Service interfaces and the Options of New.
// The session layer talks to this package only through them.

package game

import (
	"context"
	"log/slog"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// State is a connection's protocol state, as Mobius ConnectionState.
// Packets that arrive in the wrong state are dropped.
type State int

// Connection states, in the order a client goes through them. A restart
// goes from StateInGame back to StateAuthenticated.
const (
	StateConnected State = iota
	StateAuthenticated
	StateEntering
	StateInGame
)

// Session is one client connection. It is implemented by internal/gameserver.
// All methods are safe to call from any goroutine. The game keys its players
// by session, so the implementation must be comparable: a pointer type.
type Session interface {
	// Send queues an encoded packet body (opcode included). It never
	// blocks; the connection's writer encrypts and sends bodies in order.
	world.Sink

	// Account is the lowercase account name the client authenticated with.
	Account() string
	// PlayKey1 is the session key that CharSelectionInfo shows as the
	// account id.
	PlayKey1() int32

	State() State
	SetState(State)

	// ReturnToCharacterSelect sends RestartResponse(true) and the
	// character list, and sets StateAuthenticated.
	ReturnToCharacterSelect(ctx context.Context)

	// CloseAfter sends final (if not nil), then closes the connection
	// after delay (Mobius closes 1 s after LeaveWorld).
	CloseAfter(final []byte, delay time.Duration)

	RemoteAddr() string
}

// CharacterSlots is an optional interface of a Session: the object ID of
// the character in each slot of the last character list the session sent
// (Mobius GameClient.getCharSelection). When a session has it,
// SelectCharacter takes the slot from it, so the slot means what the client
// saw; otherwise it lists the account's characters again (in the
// CharSelectionInfo order).
type CharacterSlots interface {
	SlotObjectID(slot int32) (objectID int32, ok bool)
}

// Service is the game side that the session layer calls. *Server
// implements it. All methods are safe to call from any goroutine; they hand
// work to the world loop themselves.
type Service interface {
	// SelectCharacter loads the character in the given slot of
	// s.Account() (slots in CharSelectionInfo order), kicks any other
	// session of the same account out of the world, sends CharSelected and
	// sets StateEntering.
	SelectCharacter(ctx context.Context, s Session, slot int) error

	// EnterWorld puts the selected character into the world and sends the
	// enter-world packets. It sets StateInGame.
	EnterWorld(s Session)

	// Handle and HandleEx dispatch an in-world client packet (by opcode,
	// or 0xD0 sub-opcode) to its handler on the loop.
	Handle(s Session, opcode byte, body []byte)
	HandleEx(s Session, subOpcode uint16, body []byte)

	// Restart saves the player, removes it from the world and calls
	// s.ReturnToCharacterSelect. Logout does the same, then closes the
	// connection with LeaveWorld. Disconnected handles a lost connection.
	Restart(s Session)
	Logout(s Session)
	Disconnected(s Session)

	// OnLoop runs fn on the world loop and waits for it, for pre-world
	// work that needs the loop (the character creation script hook).
	OnLoop(ctx context.Context, fn func()) error

	// Scripts is the Lua engine. Call it only on the loop.
	Scripts() *scripting.Engine
}

// Options configures New.
type Options struct {
	Config   config.Config
	ServerID int
	Data     *data.Data
	// World is the world data: NPCs, spawns, teleporters, zones and map
	// regions (data.LoadWorld). The NPC, zone, teleporter and map region
	// features use it; nil means none of that data is loaded.
	World *data.World

	// Characters stores the characters. It must also be a
	// storage.CharacterSaver: players are saved with the changes to their
	// items.
	Characters storage.CharacterStore
	IDs        storage.IDAllocator
	// IDPool gives new items their object IDs on the loop (decision I3).
	// Fill it before Run. Nil makes New create one on IDs and fill it.
	IDPool *storage.IDPool
	// Passives are the passive skill data (data.LoadPassives); the weight
	// penalty uses skill 4270 from it. Nil means a weight penalty changes
	// no stats.
	Passives *data.Passives
	// Categories are the class categories (data.LoadCategories) that item
	// conditions test; nil means a player is in none.
	Categories *data.Categories
	// Economy is the shop data (data.LoadEconomy): buy lists, multisell
	// lists and merchant price configs. Nil means no NPC sells anything.
	Economy *data.Economy
	Scripts *scripting.Engine
	// ScriptOutput is the adapter the scripting engine got as
	// scripting.Options.Output (make it with NewScriptOutput before the
	// engine). New connects it to the server, so that npc:say reaches the
	// players. Nil makes New create one (Server.ScriptOutput), for an engine
	// made afterwards.
	ScriptOutput *ScriptOutput

	// Geo answers the geodata questions; nil means geo.None (no geodata).
	Geo geo.Engine

	// Rand is the game's random source on the loop (decision F7): combat
	// rolls draw from it in Mobius's order. Nil means a PCG source with a
	// random seed, which New logs. Tests pass rnd.NewScripted or rnd.New
	// with a fixed seed.
	Rand rnd.Source

	// Clock drives the world loop; nil means the real clock.
	Clock  world.Clock
	Logger *slog.Logger
	// CheckLoopOwner makes the loop-only methods of the world check that
	// they run on the loop (world.LoopOptions.CheckOwner). For tests and
	// debugging; each check costs about a microsecond.
	CheckLoopOwner bool
}
