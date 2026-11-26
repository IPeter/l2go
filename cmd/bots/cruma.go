package main

import (
	"context"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Cruma Tower hunt (-mode cruma), Plan 13's load test: bots fight on
// the first floor of Cruma Tower, a multi-floor place where pathfinding,
// sight checks and the monster AI work hardest with geodata on.

// modeCruma hunts like modeHunt, in Cruma Tower.
const modeCruma = "cruma"

// crumaLevel is the level a cruma bot gets when -level is not given: the
// first floor's monsters are level 40 to 45.
const crumaLevel int32 = 40

// maxBotLevel is the highest level -level accepts (Mobius MaxPlayerLevel
// for Interlude, 80).
const maxBotLevel = 80

// crumaLeash is how far from its spot a cruma bot may be before it
// teleports back: past the hunt range, so a chase does not end in a
// teleport, while a restart in the village does.
const crumaLeash = 4000

// crumaSpots are where the cruma bots hunt, on the first floor of Cruma
// Tower next to the Porta and Excuro spawns (Mobius spawns/Others/
// 20_21.xml). Bot i takes spot i modulo their number.
var crumaSpots = []testclient.Position{
	{X: 12600, Y: 118300, Z: -12082},
	{X: 14500, Y: 118000, Z: -12085},
	{X: 20700, Y: 118000, Z: -12085},
	{X: 22400, Y: 119500, Z: -12082},
}

// crumaSpot returns the spot of bot id.
func crumaSpot(id int) testclient.Position { return crumaSpots[id%len(crumaSpots)] }

// resolveLevel sets c.level from the -level flag: -1 (not given) is
// crumaLevel for a cruma bot and 0 for the others; anything else must be a
// level, or 0 to keep the character's own.
func (c *config) resolveLevel(flag int) error {
	switch {
	case flag == -1 && c.mode == modeCruma:
		c.level = crumaLevel
	case flag == -1:
		c.level = 0
	case flag < 0 || flag > maxBotLevel:
		return fmt.Errorf("-level %d: use 1 to %d, or 0 to keep the character's level", flag, maxBotLevel)
	default:
		c.level = int32(flag) //nolint:gosec // checked above
	}
	return nil
}

// crumaStep is what a cruma bot does next.
type crumaStep uint8

const (
	// crumaRestart: the bot is dead.
	crumaRestart crumaStep = iota
	// crumaLevelUp: the bot is below -level.
	crumaLevelUp
	// crumaTeleport: the bot is away from its spot (it entered elsewhere,
	// or restarted in the village).
	crumaTeleport
	// crumaFight: the bot hunts around its spot.
	crumaFight
)

// nextCrumaStep decides a cruma bot's next step from the character's state,
// the level it should have (0: any) and its spot.
func nextCrumaStep(me testclient.Me, level int32, spot testclient.Position) crumaStep {
	switch {
	case me.Dead:
		return crumaRestart
	case level > 0 && me.HasUserInfo && me.UserInfo.Level < level:
		return crumaLevelUp
	case me.Pos.DistanceTo(spot) > crumaLeash:
		return crumaTeleport
	}
	return crumaFight
}

// crumaHunt is one action of a cruma bot: restart when dead, level up with
// //set_level, teleport to its spot with //teleport ("teleport": the GM
// command to the UserInfo after the teleport), else hunt there.
func (b *bot) crumaHunt(ctx context.Context, gc *testclient.GameConn) error {
	spot := crumaSpot(b.id)
	switch nextCrumaStep(gc.World().Me(), b.cfg.level, spot) {
	case crumaRestart:
		return b.restart(ctx, gc)
	case crumaLevelUp:
		t0 := time.Now()
		if _, err := gc.LevelTo(ctx, b.cfg.level); err != nil {
			return staged("level", err)
		}
		b.stats.Observe("level", time.Since(t0))
		return nil
	case crumaTeleport:
		return b.toCruma(ctx, gc, spot)
	}
	b.start = spot // the home of the hunt, also after a reconnect
	return b.hunt(ctx, gc)
}

// toCruma teleports the bot to its spot and makes the spot the home of its
// hunt.
func (b *bot) toCruma(ctx context.Context, gc *testclient.GameConn, spot testclient.Position) error {
	t0 := time.Now()
	if err := gc.GMCommand(ctx, fmt.Sprintf("teleport %d %d %d", spot.X, spot.Y, spot.Z)); err != nil {
		return staged("teleport", err)
	}
	if _, err := gc.FinishTeleport(ctx); err != nil {
		return staged("teleport", err)
	}
	b.stats.Observe("teleport", time.Since(t0))
	b.start, b.from, b.to, b.reported = spot, spot, spot, spot
	b.target, b.attacking, b.avoid = 0, 0, 0
	return nil
}
