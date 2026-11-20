package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/testclient"
)

// Retry backoff after a failed session.
const (
	minBackoff = 2 * time.Second
	maxBackoff = 30 * time.Second
)

// logoutTimeout bounds the logout at the end of a run.
const logoutTimeout = 5 * time.Second

// defaultSpeed is the movement speed a bot assumes when UserInfo has none.
const defaultSpeed = 120.0

// phrases are what bots say in general chat.
var phrases = []string{
	"hello", "anyone selling a sword?", "lfg", "nice weather today", "where is the gatekeeper?",
	"wts adena", "brb", "gl hf", "which way to the village?", "lol",
}

// bot is one load-test player.
type bot struct {
	id      int
	account string
	char    string
	cfg     config
	stats   *Stats
	rng     *rand.Rand
	logger  *slog.Logger
	state   state

	// The estimated movement: from from to to, leaving at departed with
	// speed units per second. The client reports its estimated position
	// with ValidatePosition, as the real client does.
	start    testclient.Position
	from, to testclient.Position
	departed time.Time
	speed    float64
	reported testclient.Position
}

func newBot(id int, cfg config, stats *Stats, logger *slog.Logger) *bot {
	account, char := botNames(cfg.prefix, id, cfg.n)
	return &bot{
		id:      id,
		account: account,
		char:    char,
		cfg:     cfg,
		stats:   stats,
		rng:     rand.New(rand.NewPCG(uint64(cfg.seed), uint64(id))), //nolint:gosec // behaviour, not security
		logger:  logger.With("bot", account),
		state:   stateIdle,
	}
}

// botNames returns the account and character names of bot id: the prefix
// and the bot's number, zero-padded to the width of the largest number
// (at least 4 digits): bot0007 and Bot0007.
func botNames(prefix string, id, n int) (account, char string) {
	num := fmt.Sprintf("%0*d", nameDigits(n), id)
	return prefix + num, strings.ToUpper(prefix[:min(1, len(prefix))]) + prefix[min(1, len(prefix)):] + num
}

// nameDigits is the number width for n bots.
func nameDigits(n int) int {
	return max(4, len(strconv.Itoa(max(n-1, 0))))
}

// setState moves the bot to s in the statistics.
func (b *bot) setState(s state) {
	if b.state == s {
		return
	}
	b.stats.Move(b.state, s)
	b.state = s
}

// run plays sessions until ctx ends, waiting with a growing backoff after
// each failed one.
func (b *bot) run(ctx context.Context) {
	defer b.setState(stateIdle)
	backoff := minBackoff
	for ctx.Err() == nil {
		b.setState(stateStarting)
		err := b.session(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			b.stats.Error(errorKind(err))
			b.logger.Debug("session failed", "err", err, "retry in", backoff)
		}
		b.setState(stateWaiting)
		if sleep(ctx, jitter(b.rng, backoff)) != nil {
			return
		}
		backoff = min(2*backoff, maxBackoff)
	}
}

// session logs in, enters the world and plays until ctx ends (then logs
// out) or the connection breaks.
func (b *bot) session(ctx context.Context) error {
	t0 := time.Now()
	lctx, cancel := context.WithTimeout(ctx, b.cfg.timeout)
	keys, addr, err := login(lctx, b.cfg, b.account)
	cancel()
	if err != nil {
		return staged("login", err)
	}
	b.stats.Observe("login", time.Since(t0))

	gctx, cancel := context.WithTimeout(ctx, b.cfg.timeout)
	gc, err := testclient.DialGame(gctx, addr, testclient.InterludeProtocol, testclient.WithQueueLimit(b.cfg.queue))
	cancel()
	if gc != nil {
		defer func() { _ = gc.Close() }()
	}
	if err != nil {
		return staged("connect", err)
	}
	b.stats.Track(gc)
	defer b.stats.Untrack(gc)

	if err := b.enter(ctx, gc, keys); err != nil {
		return err
	}
	b.setState(stateInWorld)
	for {
		if sleep(ctx, jitter(b.rng, b.cfg.step)) != nil {
			return b.logout(ctx, gc)
		}
		err := b.act(ctx, gc)
		switch {
		case ctx.Err() != nil:
			return b.logout(ctx, gc)
		case err == nil:
		case errors.Is(err, testclient.ErrServerClosed):
			return err
		default:
			b.stats.Error(errorKind(err))
			b.logger.Debug("action failed", "err", err)
		}
	}
}

// enter authenticates on the game server, creates the bot's character if
// the account has none, selects it and enters the world.
func (b *bot) enter(ctx context.Context, gc *testclient.GameConn, keys session.Keys) error {
	step := func(stage string, fn func(ctx context.Context) error) error {
		ctx, cancel := context.WithTimeout(ctx, b.cfg.timeout)
		defer cancel()
		return staged(stage, fn(ctx))
	}
	var list testclient.CharacterList
	if err := step("auth", func(ctx context.Context) error {
		var err error
		list, err = gc.AuthLogin(ctx, b.account, keys)
		return err
	}); err != nil {
		return err
	}
	if _, ok := list.Find(b.char); !ok {
		if err := step("create", func(ctx context.Context) error {
			if _, err := gc.NewCharacter(ctx); err != nil {
				return err
			}
			res, err := gc.CreateCharacter(ctx, humanFighter(b.char))
			if err != nil {
				return err
			}
			if !res.Created {
				return fmt.Errorf("%w: character %s: %v", errRefused, b.char, res.Reason)
			}
			list = res.Characters
			return nil
		}); err != nil {
			return err
		}
	}
	slot := slices.IndexFunc(list.Characters, func(c testclient.CharInfo) bool { return c.Name == b.char })
	if slot < 0 {
		return staged("select", fmt.Errorf("character %s is not in the list", b.char))
	}
	if err := step("select", func(ctx context.Context) error {
		_, err := gc.SelectCharacter(ctx, int32(slot)) //nolint:gosec // at most 7 characters
		return err
	}); err != nil {
		return err
	}
	t0 := time.Now()
	if err := step("enter", func(ctx context.Context) error {
		_, err := gc.EnterWorld(ctx)
		return err
	}); err != nil {
		return err
	}
	b.stats.Observe("enter", time.Since(t0))

	me := gc.World().Me()
	b.start = me.Pos
	b.from, b.to, b.reported = b.start, b.start, b.start
	b.departed = time.Now()
	b.speed = moveSpeed(me.UserInfo)
	return nil
}

// moveSpeed is the run (or walk) speed UserInfo shows, in units per
// second.
func moveSpeed(u testclient.UserInfo) float64 {
	mult := u.MoveMultiplier
	if mult <= 0 {
		mult = 1
	}
	speed := float64(u.WalkSpd) * mult
	if u.Running {
		speed = float64(u.RunSpd) * mult
	}
	if speed <= 0 {
		return defaultSpeed
	}
	return speed
}

// act reports the bot's position and does one random action: talk to the
// nearest NPC, say something, or walk.
func (b *bot) act(ctx context.Context, gc *testclient.GameConn) error {
	// The packets queued while the bot slept are old; an answer must come
	// after the request.
	if _, err := gc.Drain(); err != nil {
		b.stats.Error("stream:malformed")
	}
	ctx, cancel := context.WithTimeout(ctx, b.cfg.timeout)
	defer cancel()
	if err := b.reportPosition(ctx, gc); err != nil {
		return staged("validate", err)
	}
	r := b.rng.Float64()
	switch {
	case r < b.cfg.talk:
		return b.talk(ctx, gc)
	case r < b.cfg.talk+b.cfg.say:
		return b.say(ctx, gc)
	default:
		return b.walk(ctx, gc)
	}
}

// position estimates where the bot is at now.
func (b *bot) position(now time.Time) testclient.Position {
	return estimate(b.from, b.to, now.Sub(b.departed), b.speed)
}

// estimate returns the point on the way from from to to after walking for
// elapsed at speed units per second; it stops at to.
func estimate(from, to testclient.Position, elapsed time.Duration, speed float64) testclient.Position {
	dist := from.DistanceTo(to)
	if dist == 0 || speed <= 0 {
		return to
	}
	f := speed * elapsed.Seconds() / dist
	if f >= 1 {
		return to
	}
	if f <= 0 {
		return from
	}
	lerp := func(a, b int32) int32 { return a + int32(math.Round(float64(b-a)*f)) }
	return testclient.Position{X: lerp(from.X, to.X), Y: lerp(from.Y, to.Y), Z: lerp(from.Z, to.Z)}
}

// heading is the Mobius heading (0-65535) of the direction from from to
// to (Util.calculateHeadingFrom).
func heading(from, to testclient.Position) int32 {
	deg := math.Atan2(float64(to.Y-from.Y), float64(to.X-from.X)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return int32(deg * 182.044444444)
}

// reportPosition sends ValidatePosition with the estimated position when
// the bot moved since the last report.
func (b *bot) reportPosition(ctx context.Context, gc *testclient.GameConn) error {
	pos := b.position(time.Now())
	if pos == b.reported {
		return nil
	}
	if err := gc.ValidatePosition(ctx, pos, heading(b.from, b.to)); err != nil {
		return err
	}
	b.reported = pos
	return nil
}

// randomPoint returns a uniformly random point of the disc of radius
// around center, at center's height.
func randomPoint(rng *rand.Rand, center testclient.Position, radius float64) testclient.Position {
	r := radius * math.Sqrt(rng.Float64())
	a := 2 * math.Pi * rng.Float64()
	return testclient.Position{
		X: center.X + int32(math.Round(r*math.Cos(a))),
		Y: center.Y + int32(math.Round(r*math.Sin(a))),
		Z: center.Z,
	}
}

// walk clicks a random point around the start and waits for the server's
// MoveToLocation for the bot's character.
func (b *bot) walk(ctx context.Context, gc *testclient.GameConn) error {
	now := time.Now()
	origin := b.position(now)
	dest := randomPoint(b.rng, b.start, b.cfg.radius)
	me := gc.World().Me().ObjectID
	if err := gc.SendMove(ctx, testclient.MoveRequest{Target: dest, Origin: origin}); err != nil {
		return staged("move", err)
	}
	m, err := testclient.Await(ctx, gc, func(m testclient.MoveToLocation) bool { return m.ObjectID == me })
	if err != nil {
		return staged("move", err)
	}
	b.stats.Observe("move", time.Since(now))
	b.from, b.to, b.departed = m.Origin(), m.Destination(), time.Now()
	return nil
}

// say sends a phrase to general chat and waits for its echo.
func (b *bot) say(ctx context.Context, gc *testclient.GameConn) error {
	text := phrases[b.rng.IntN(len(phrases))]
	me := gc.World().Me().ObjectID
	t0 := time.Now()
	if err := gc.Say(ctx, testclient.ChatGeneral, text); err != nil {
		return staged("say", err)
	}
	_, err := testclient.Await(ctx, gc, func(m testclient.CreatureSay) bool {
		return m.ObjectID == me && m.Type == testclient.ChatGeneral && m.Text == text
	})
	if err != nil {
		return staged("say", err)
	}
	b.stats.Observe("say", time.Since(t0))
	return nil
}

// talk targets the nearest NPC in range and opens its dialog. The server
// walks the character to it first, so "talk" includes the walk; "target"
// is the plain round trip. An NPC that is the target already (the bot
// talked to it before and stayed near) is not clicked to target it: the
// server answers a click on the current target by talking, never with
// MyTargetSelected.
func (b *bot) talk(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	me := w.Me()
	npc, ok := nearestNpc(me.Pos, w.Npcs(), b.cfg.talkRange)
	if !ok {
		return b.walk(ctx, gc)
	}
	if me.TargetID != npc.ObjectID {
		t0 := time.Now()
		if _, err := gc.Target(ctx, npc.ObjectID); err != nil {
			return staged("target", err)
		}
		b.stats.Observe("target", time.Since(t0))
	}
	t1 := time.Now()
	if _, err := gc.Talk(ctx, npc.ObjectID); err != nil {
		return staged("talk", err)
	}
	b.stats.Observe("talk", time.Since(t1))
	// The character walked up to the NPC.
	me = gc.World().Me()
	pos := me.Pos
	if me.Moving {
		pos = me.Dest
	}
	b.from, b.to, b.departed = pos, pos, time.Now()
	return nil
}

// nearestNpc returns the NPC of objs (shown by NpcInfo, not attackable)
// nearest to me within maxDist. Ties go to the lower object ID.
func nearestNpc(me testclient.Position, objs []testclient.Object, maxDist float64) (testclient.Object, bool) {
	var (
		best     testclient.Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		if o.Kind != testclient.KindNpc || o.Attackable {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if d > maxDist {
			continue
		}
		if !found || d < bestDist || (d == bestDist && cmp.Less(o.ObjectID, best.ObjectID)) {
			best, bestDist, found = o, d, true
		}
	}
	return best, found
}

// logout logs out with a short timeout of its own, since ctx is usually
// done by then.
func (b *bot) logout(ctx context.Context, gc *testclient.GameConn) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), logoutTimeout)
	defer cancel()
	if err := gc.Logout(ctx); err != nil {
		b.stats.Error(errorKind(staged("logout", err)))
		b.logger.Debug("logout failed", "err", err)
	}
	return nil
}

// login runs the client's login sequence and returns the session keys and
// the game server address (the -game flag, or the server list's).
func login(ctx context.Context, cfg config, account string) (session.Keys, string, error) {
	c, err := testclient.DialLogin(ctx, cfg.login)
	if err != nil {
		return session.Keys{}, "", err
	}
	defer func() { _ = c.Close() }()
	if err := c.AuthGameGuard(ctx); err != nil {
		return session.Keys{}, "", err
	}
	res, err := c.Login(ctx, account, cfg.password, false)
	if err != nil {
		return session.Keys{}, "", err
	}
	var list testclient.ServerListResult
	switch {
	case res.Ok != nil:
		if list, err = c.ServerList(ctx); err != nil {
			return session.Keys{}, "", err
		}
	case res.ServerList != nil:
		list = *res.ServerList
	case res.Fail != nil:
		return session.Keys{}, "", &testclient.LoginFailError{Reason: res.Fail.Reason}
	default:
		return session.Keys{}, "", fmt.Errorf("%w: %v", errRefused, res)
	}
	addr := cfg.game
	if addr == "" {
		if addr, err = serverAddr(list, cfg.serverID); err != nil {
			return session.Keys{}, "", err
		}
	}
	play, err := c.ServerLogin(ctx, cfg.serverID)
	if err != nil {
		return session.Keys{}, "", err
	}
	switch {
	case play.Ok != nil:
		return c.Keys(), addr, nil
	case play.LoginFail != nil:
		return session.Keys{}, "", &testclient.LoginFailError{Reason: play.LoginFail.Reason}
	default:
		return session.Keys{}, "", fmt.Errorf("%w: %v", errRefused, play)
	}
}

// serverAddr returns the address of game server id from the server list.
func serverAddr(list testclient.ServerListResult, id byte) (string, error) {
	for _, s := range list.Servers {
		if s.ID == id {
			if !s.Up {
				return "", fmt.Errorf("%w: game server %d is down", errRefused, id)
			}
			return net.JoinHostPort(s.IP.String(), strconv.Itoa(int(s.Port))), nil
		}
	}
	return "", fmt.Errorf("game server %d is not in the server list", id)
}

// humanFighter is the CharacterCreate of a male Human Fighter with the
// template's stats, as the client sends it.
func humanFighter(name string) testclient.CreateRequest {
	return testclient.CreateRequest{
		Name: name, Race: 0, Sex: 0, ClassID: 0,
		INT: 21, STR: 40, CON: 43, MEN: 25, DEX: 30, WIT: 11,
		HairStyle: 1, HairColor: 2, Face: 1,
	}
}

// jitter returns d scaled by a random factor between 0.5 and 1.5.
func jitter(rng *rand.Rand, d time.Duration) time.Duration {
	return time.Duration(float64(d) * (0.5 + rng.Float64()))
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
