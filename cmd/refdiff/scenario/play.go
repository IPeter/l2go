package scenario

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// Item templates the scenarios use (Mobius item IDs).
const (
	itemAdena         int32 = 57
	itemDagger        int32 = 10
	itemBow           int32 = 14
	itemWoodenArrow   int32 = 17
	itemHealingPotion int32 = 1060 // Lesser Healing Potion
	// itemHealingPotionM is the Healing Potion: skill 2032, HP over 15 s,
	// a 10 s reuse.
	itemHealingPotionM int32 = 1061
	itemSquirePants    int32 = 1147
	itemCrystalD       int32 = 1458
	itemCrystalC       int32 = 1459
	// itemSoulshotNG is Soulshot: No Grade.
	itemSoulshotNG  int32 = 1835
	itemSquireSword int32 = 2369
)

// NPC templates the scenarios use (Mobius NPC IDs).
const (
	// npcKaterina is the grocer of Talking Island village (buylist 3000400).
	npcKaterina int32 = 30004
	// npcWilford is the warehouse keeper of Talking Island village.
	npcWilford int32 = 30005
	// npcGalladucci is the weapon merchant of Giran (multisell 300974001).
	npcGalladucci int32 = 30097
	// npcBeardedKeltir is the level 1 monster of the fields around the
	// Human Fighter's start point.
	npcBeardedKeltir int32 = 20481
	// npcBaulro is the Human Mystic trainer of Talking Island village.
	npcBaulro int32 = 30033
	// npcNewbieGuide is the Newbie Guide of Talking Island village, who
	// gives the support magic of the newbie buffer.
	npcNewbieGuide int32 = 30598
)

// Fixed spots the GM setups move characters to, so that both servers start
// the scenario proper at the same point: a few steps from the NPC's spawn
// (Mobius spawns/TalkingIsland/TalkingIslandNPCs.xml, spawns/Giran/GiranNPCs.xml)
// or, for two players, on the open ground east of the Human Fighter's start.
var (
	spotKaterina   = testclient.Position{X: -84124, Y: 240403, Z: -3720}
	spotWilford    = testclient.Position{X: -81432, Y: 243424, Z: -3712}
	spotGalladucci = testclient.Position{X: 79679, Y: 145522, Z: -3494}
	spotBaulro     = testclient.Position{X: -84700, Y: 244950, Z: -3720}
	spotGuide      = testclient.Position{X: -84050, Y: 243270, Z: -3728}
	spotDropAlice  = testclient.Position{X: -71000, Y: 258100, Z: -3104}
	spotDropBob    = testclient.Position{X: -70900, Y: 258100, Z: -3104}
	// keltirField is the middle of the Bearded Keltir territory south-west
	// of the Human Fighter's start (gludio32_1725_01 in
	// spawns/TalkingIsland/TalkingIslandMonsters.xml).
	keltirField = testclient.Position{X: -74384, Y: 254368, Z: -3200}
)

// Pauses between requests.
const (
	// settle lets an answer's trailing packets arrive before the next
	// request, so that both servers show them in the same place: Mobius
	// gathers inventory changes into an InventoryUpdate 100 ms later.
	settle = 500 * time.Millisecond
	// floodGap is how long Mobius's flood protectors drop the next drop,
	// transaction (buy, sell, warehouse) or multisell request:
	// FloodProtector*Interval = 10 ticks of 100 ms, plus a margin.
	floodGap = 1200 * time.Millisecond
	// bypassGap separates a dialog link from the click or link before it:
	// FloodProtectorServerBypassInterval is 3 ticks.
	bypassGap = 500 * time.Millisecond
	// stanceWait outlasts the combat stance (15 s after the last blow,
	// checked every second), during which Mobius refuses a logout.
	stanceWait = 17 * time.Second
)

// gmHint is added to the error of a GM step that got no answer.
func gmHint(p *Player, err error) error {
	return fmt.Errorf("%w (the scenario needs GM rights for the character %s on %s)", err, p.Char, p.s.target.Label)
}

// giveItem is a GM step: //create_item gives p count of itemID, and the step
// waits until the inventory has them.
func giveItem(p *Player, itemID int32, count int64) Step {
	return Step{fmt.Sprintf("%s gets %d of item %d (GM)", p.Role, count, itemID), func(ctx context.Context) error {
		w := p.Game().World()
		want := w.ItemCount(itemID) + count
		if err := p.Game().GMCommand(ctx, fmt.Sprintf("create_item %d %d", itemID, count)); err != nil {
			return err
		}
		if err := poll(ctx, func() bool { return w.ItemCount(itemID) >= want }); err != nil {
			return gmHint(p, fmt.Errorf("item %d: %d in the inventory, want %d: %w", itemID, w.ItemCount(itemID), want, err))
		}
		return nil
	}}
}

// gmMoveTo is a GM step: //move_to teleports p to pos, and the step finishes
// the teleport the way the client does.
func gmMoveTo(p *Player, pos testclient.Position) Step {
	return Step{fmt.Sprintf("%s teleports to %d %d %d (GM)", p.Role, pos.X, pos.Y, pos.Z), func(ctx context.Context) error {
		if err := p.Game().GMCommand(ctx, fmt.Sprintf("move_to %d %d %d", pos.X, pos.Y, pos.Z)); err != nil {
			return err
		}
		if _, err := p.Game().FinishTeleport(ctx); err != nil {
			return gmHint(p, err)
		}
		return nil
	}}
}

// poll calls done every pollInterval until it is true or ctx ends.
func poll(ctx context.Context, done func() bool) error {
	t := time.NewTicker(pollInterval)
	defer t.Stop()
	for !done() {
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// inventoryItem returns the first item of itemID in p's inventory.
func inventoryItem(p *Player, itemID int32) (testclient.Item, error) {
	it, ok := p.Game().World().FindItem(itemID)
	if !ok {
		return testclient.Item{}, fmt.Errorf("%s has no item %d", p.Role, itemID)
	}
	return it, nil
}

// findNpc waits until p sees an NPC of template npcID and returns the
// nearest one.
func findNpc(ctx context.Context, p *Player, npcID int32) (testclient.Object, error) {
	var npc testclient.Object
	err := poll(ctx, func() bool {
		var ok bool
		npc, ok = p.Game().World().FindNpc(npcID)
		return ok
	})
	if err != nil {
		return npc, fmt.Errorf("%s does not see NPC %d: %w", p.Role, npcID, err)
	}
	return npc, nil
}

// talkTo opens the dialog of the nearest NPC of template npcID (the
// character walks up to it) and returns the page.
func talkTo(ctx context.Context, s *Session, p *Player, npcID int32) (testclient.NpcHtmlMessage, error) {
	npc, err := findNpc(ctx, p, npcID)
	if err != nil {
		return testclient.NpcHtmlMessage{}, err
	}
	s.Note("%s talks to NPC %d %q, %.0f units away", p.Role, npc.NpcID, npc.Name, npc.Pos.DistanceTo(p.Game().World().Me().Pos))
	return p.Game().Talk(ctx, npc.ObjectID)
}

// openLink clicks the link of page whose target contains part, such as
// "_Buy" or "DepositP", after bypassGap.
func openLink(ctx context.Context, p *Player, page testclient.NpcHtmlMessage, part string) error {
	l, ok := FindLink(page.HTML, part)
	if !ok {
		return fmt.Errorf("the page of NPC object %d has no link with %q", page.ObjectID, part)
	}
	if err := sleep(ctx, bypassGap); err != nil {
		return err
	}
	return l.Send(ctx, p.Game())
}

// FindLink returns the first link of a dialog page whose target contains
// part, ignoring case.
func FindLink(html, part string) (Link, bool) {
	part = strings.ToLower(part)
	for _, m := range linkRe.FindAllStringSubmatch(html, -1) {
		if strings.Contains(strings.ToLower(m[2]), part) {
			return Link{Bypass: !strings.EqualFold(m[1], "link"), Target: m[2]}, true
		}
	}
	return Link{}, false
}

// huntMonster returns a monster of template npcID for p to attack
// (testclient.HuntMonster): monsters stand at random points of their
// territory, so the template is what keeps both servers on the same kind of
// fight. When none is in sight, p walks towards field until one shows up.
func huntMonster(ctx context.Context, p *Player, npcID int32, field testclient.Position) (testclient.Object, error) {
	return p.Game().HuntMonster(ctx, npcID, field)
}

// fightUntilDead waits until the monster or player targetID that p attacks
// dies, and returns how many swings p made meanwhile
// (testclient.FightUntilDead). A death of p ends it with
// testclient.ErrDied.
func fightUntilDead(ctx context.Context, p *Player, targetID int32) (int, error) {
	return p.Game().FightUntilDead(ctx, targetID)
}
