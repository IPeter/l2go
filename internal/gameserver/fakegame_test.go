package gameserver

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
)

// gameCall is one call the session layer made to the game.
type gameCall struct {
	method  string // "SelectCharacter", "EnterWorld", "Handle", ...
	session game.Session
	slot    int
	opcode  byte
	sub     uint16
	body    []byte
}

// errLoopStopped is what fakeGame.OnLoop returns after stopLoop.
var errLoopStopped = errors.New("fake loop stopped")

// fakeGame is the game.Service of the session tests. Its loop is a mutex:
// OnLoop runs functions one at a time, as the world loop does. It records
// every call and plays a small game:
//   - SelectCharacter sends CharSelected for the account's character in the
//     slot and sets StateEntering (or fails with selectErr).
//   - EnterWorld sets StateInGame and sends ActionFailed.
//   - Handle and HandleEx answer with ActionFailed.
//   - Restart returns the session to the character screen from another
//     goroutine, as the game does after saving.
//   - Logout closes the session with LeaveWorld after logoutDelay.
type fakeGame struct {
	chars   storage.CharacterStore
	scripts *scripting.Engine

	loop sync.Mutex // held while an OnLoop function runs
	// holdEnter makes EnterWorld only record the call, as when the loop
	// has not run it yet. Set it before the session uses the game.
	holdEnter bool

	mu             sync.Mutex
	logoutDelay    time.Duration
	calls          []gameCall
	selectErr      error
	stopped        bool
	onLoop         int  // OnLoop calls that ran their function
	scriptsOffLoop bool // Scripts was called outside OnLoop
	bg             sync.WaitGroup
}

var _ game.Service = (*fakeGame)(nil)

func newFakeGame(chars storage.CharacterStore, scripts *scripting.Engine) *fakeGame {
	return &fakeGame{chars: chars, scripts: scripts, logoutDelay: 50 * time.Millisecond}
}

func (g *fakeGame) record(c gameCall) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls = append(g.calls, c)
}

// callsTo returns the recorded calls of method, in order.
func (g *fakeGame) callsTo(method string) []gameCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []gameCall
	for _, c := range g.calls {
		if c.method == method {
			out = append(out, c)
		}
	}
	return out
}

// methods returns the names of every recorded call, in order.
func (g *fakeGame) methods() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, c := range g.calls {
		out = append(out, c.method)
	}
	return out
}

// waitCalls waits until method was called n times and returns the calls.
func (g *fakeGame) waitCalls(t *testing.T, method string, n int) []gameCall {
	t.Helper()
	waitFor(t, method+" calls", func() bool { return len(g.callsTo(method)) >= n })
	return g.callsTo(method)
}

func (g *fakeGame) setLogoutDelay(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.logoutDelay = d
}

func (g *fakeGame) setSelectErr(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.selectErr = err
}

func (g *fakeGame) stopLoop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stopped = true
}

func (g *fakeGame) onLoopCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.onLoop
}

// wait waits for the goroutines Restart started.
func (g *fakeGame) wait() { g.bg.Wait() }

func (g *fakeGame) SelectCharacter(ctx context.Context, s game.Session, slot int) error {
	g.record(gameCall{method: "SelectCharacter", session: s, slot: slot})
	g.mu.Lock()
	err := g.selectErr
	g.mu.Unlock()
	if err != nil {
		return err
	}
	chars, err := g.chars.ListByAccount(ctx, s.Account())
	if err != nil {
		return err
	}
	if slot < 0 || slot >= len(chars) {
		return errors.New("no character in the slot")
	}
	ch := chars[slot]
	s.Send(serverpackets.NewCharSelectedPacket(serverpackets.CharSelected{
		Name: ch.Name, ObjectID: ch.ObjectID, SessionID: s.PlayKey1(), Level: ch.Level,
	}))
	s.SetState(game.StateEntering)
	return nil
}

func (g *fakeGame) EnterWorld(s game.Session) {
	g.record(gameCall{method: "EnterWorld", session: s})
	if g.holdEnter {
		return
	}
	s.SetState(game.StateInGame)
	s.Send(serverpackets.NewActionFailedPacket())
}

func (g *fakeGame) Handle(s game.Session, opcode byte, body []byte) {
	g.record(gameCall{method: "Handle", session: s, opcode: opcode, body: slices.Clone(body)})
	s.Send(serverpackets.NewActionFailedPacket())
}

func (g *fakeGame) HandleEx(s game.Session, sub uint16, body []byte) {
	g.record(gameCall{method: "HandleEx", session: s, sub: sub, body: slices.Clone(body)})
	s.Send(serverpackets.NewActionFailedPacket())
}

func (g *fakeGame) Restart(s game.Session) {
	g.record(gameCall{method: "Restart", session: s})
	g.bg.Go(func() { s.ReturnToCharacterSelect(context.Background()) })
}

func (g *fakeGame) Logout(s game.Session) {
	g.record(gameCall{method: "Logout", session: s})
	g.mu.Lock()
	delay := g.logoutDelay
	g.mu.Unlock()
	s.CloseAfter(serverpackets.NewLeaveWorldPacket(), delay)
}

func (g *fakeGame) Disconnected(s game.Session) {
	g.record(gameCall{method: "Disconnected", session: s})
}

func (g *fakeGame) OnLoop(ctx context.Context, fn func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g.mu.Lock()
	stopped := g.stopped
	g.mu.Unlock()
	if stopped {
		return errLoopStopped
	}
	g.loop.Lock()
	defer g.loop.Unlock()
	g.mu.Lock()
	g.onLoop++
	g.mu.Unlock()
	fn()
	return nil
}

// Scripts may be called only inside OnLoop, like the game's; a call outside
// is recorded for checkLoopUse.
func (g *fakeGame) Scripts() *scripting.Engine {
	if g.loop.TryLock() {
		g.loop.Unlock()
		g.mu.Lock()
		g.scriptsOffLoop = true
		g.mu.Unlock()
	}
	return g.scripts
}

// checkLoopUse fails the test if the engine was used off the loop.
func (g *fakeGame) checkLoopUse(t *testing.T) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.scriptsOffLoop {
		t.Error("the scripts engine was used outside OnLoop")
	}
}
