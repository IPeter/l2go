package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/htmlutil"
	"github.com/IPeter/l2go/internal/world"
)

// MoveState is a movement in progress (decision W7): a straight walk from
// Origin to Destination at Speed, started at Start. The movement feature
// creates it, computes positions from it on demand, moves the creature with
// Server.MoveObject on each tick, and clears it on arrival or stop.
type MoveState struct {
	Origin      world.Position
	Destination world.Position
	// Start is the loop time the walk started at.
	Start time.Time
	// Speed is the speed of the last update, in game units per second
	// (stats.PlayerView.MoveSpeed): set when the walk starts and again on
	// every tick, as a walk/run toggle, water or //gmspeed change the pace
	// of a walk under way.
	Speed float64
	// Running is set when the creature runs rather than walks.
	Running bool
}

// ZoneFlag is a kind of zone a creature can be inside (Mobius ZoneId).
type ZoneFlag uint8

// The zone flags, in Mobius ZoneId order.
const (
	ZonePvP ZoneFlag = iota
	ZonePeace
	ZoneSiege
	ZoneMotherTree
	ZoneClanHall
	ZoneNoLanding
	ZoneWater
	ZoneJail
	ZoneMonsterTrack
	ZoneCastle
	ZoneSwamp
	ZoneNoSummonFriend
	ZoneNoStore
	ZoneNoPvP
	ZoneTown
	ZoneScript
	ZoneHQ
	ZoneDangerArea
	ZoneAltered
	ZoneNoBookmark
	ZoneNoItemDrop
	ZoneNoRestart
	// NumZoneFlags is the number of zone flags.
	NumZoneFlags
)

// Zones is the set of zones a creature is inside: a counter per flag, since
// zones of one kind may overlap (Mobius Creature._zones), and the IDs of the
// data zones in the order entered. The zero value is empty.
type Zones struct {
	count [NumZoneFlags]uint8
	ids   []int
}

// Inside reports whether the creature is inside a zone of kind f (Mobius
// isInsideZone).
func (z *Zones) Inside(f ZoneFlag) bool { return f < NumZoneFlags && z.count[f] > 0 }

// Enter counts one more zone of kind f around the creature.
func (z *Zones) Enter(f ZoneFlag) {
	if f < NumZoneFlags && z.count[f] < 255 {
		z.count[f]++
	}
}

// Leave counts one zone of kind f less.
func (z *Zones) Leave(f ZoneFlag) {
	if f < NumZoneFlags && z.count[f] > 0 {
		z.count[f]--
	}
}

// In reports whether the creature is inside the data zone with the given ID.
func (z *Zones) In(id int) bool { return slices.Contains(z.ids, id) }

// Add records that the creature is inside the data zone id. It reports false
// when it was already.
func (z *Zones) Add(id int) bool {
	if z.In(id) {
		return false
	}
	z.ids = append(z.ids, id)
	return true
}

// Remove records that the creature left the data zone id. It reports false
// when it was not inside.
func (z *Zones) Remove(id int) bool {
	i := slices.Index(z.ids, id)
	if i < 0 {
		return false
	}
	z.ids = slices.Delete(z.ids, i, i+1)
	return true
}

// IDs returns the data zones the creature is inside, in the order entered.
// The slice must not be modified.
func (z *Zones) IDs() []int { return z.ids }

// HTMLScope is the kind of page a cached action came from (Mobius
// HtmlActionScope). Each scope keeps the actions of its last page.
type HTMLScope uint8

// The HTML scopes.
const (
	HTMLScopeNpc HTMLScope = iota
	HTMLScopeNpcItem
	HTMLScopeNpcQuest
	HTMLScopeTutorial
	HTMLScopeCommunityBoard
	// NumHTMLScopes is the number of scopes.
	NumHTMLScopes
)

// HTMLCache is a player's dialog link cache (Mobius htmlActionCaches): the
// bypass and link actions of the last page of each scope, and the object
// the page came from. A bypass or link the cache does not allow is dropped.
// The zero value is empty.
type HTMLCache struct {
	actions [NumHTMLScopes][]string
	origins [NumHTMLScopes]world.ObjectID
	// lastOrigin is the origin of the last action Validate allowed.
	lastOrigin world.ObjectID
}

// Set replaces the actions of scope with those of html, a page as it is
// sent (placeholders replaced), which came from the object origin (0 when it
// is not bound to an object, for example a page of an item). Mobius does
// this for every page it sends.
func (c *HTMLCache) Set(scope HTMLScope, origin world.ObjectID, html string) {
	if scope >= NumHTMLScopes {
		return
	}
	c.actions[scope] = htmlutil.Actions(html)
	c.origins[scope] = origin
}

// Clear forgets the actions of scope.
func (c *HTMLCache) Clear(scope HTMLScope) {
	if scope >= NumHTMLScopes {
		return
	}
	c.actions[scope] = nil
	c.origins[scope] = 0
}

// Validate reports whether a page offered action (Mobius
// validateHtmlAction): ok is false when no cached page did. origin is the
// object that page came from, or 0 when it is not bound to one, in which case
// no range check is needed.
func (c *HTMLCache) Validate(action string) (origin world.ObjectID, ok bool) {
	for scope := range c.actions {
		for _, cached := range c.actions[scope] {
			if htmlutil.Matches(cached, action) {
				c.lastOrigin = c.origins[scope]
				return c.lastOrigin, true
			}
		}
	}
	return 0, false
}

// LastOrigin returns the origin of the last action Validate allowed (Mobius
// getLastHtmlActionOriginId).
func (c *HTMLCache) LastOrigin() world.ObjectID { return c.lastOrigin }

// Origin returns the object the last page of scope came from.
func (c *HTMLCache) Origin(scope HTMLScope) world.ObjectID {
	if scope >= NumHTMLScopes {
		return 0
	}
	return c.origins[scope]
}
