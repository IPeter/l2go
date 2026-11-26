package scenario

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Milestone 10 scenarios (Plan 13, "Test strategy" 7, the exact
// reference diff): the door packets on approach and after //open, a static
// object click, a raid boss's group at spawn and its kill up to the
// minions' DeleteObject, the ferry's VehicleInfo and VehicleDeparture, and
// heights in Cruma Tower. Every one needs GM rights for its setup and its
// GM commands. The raid scenarios spawn their own Kutus with //spawn, so
// the servers' spawner bosses stay alive and the scenarios can run again.
// cruma-heights compares heights only when both servers load the same
// geodata.

// IDs the Milestone 10 scenarios use (Mobius data).
const (
	// doorGludinHall is a Gludin clan hall door (gludin_agit_003_003).
	doorGludinHall int32 = 17220001
	// staticTownMapTalking is the first town map of Talking Island village.
	staticTownMapTalking int32 = 17250001
	// npcKutus is the raid boss Greyclaw Kutus, with three Guards of Kutus
	// and a Pawn of Kutus (the template's Privates).
	npcKutus      int32 = 25001
	npcKutusGuard int32 = 25002
	npcKutusPawn  int32 = 25003
	// msgRaidSuccessful is "Congratulations. Your raid was successful."
	msgRaidSuccessful int32 = 1209
)

// Fixed spots of the Milestone 10 scenarios.
var (
	// spotGludinDoor is next to doorGludinHall (-84495, 155208, -3150).
	spotGludinDoor = testclient.Position{X: -84400, Y: 155208, Z: -3150}
	// spotTownMap is within reach of the town map (-82873, 244808, -3717).
	spotTownMap = testclient.Position{X: -82823, Y: 244808, Z: -3717}
	// spotRaid is open ground east of the Human Fighter's start, far from
	// the spawner's Kutus, where the raid scenarios spawn theirs.
	spotRaid = testclient.Position{X: -70800, Y: 258100, Z: -3104}
	// spotPier is on the Talking Island pier, 660 units from where the
	// ferry docks (-96622, 261660, -3610).
	spotPier = testclient.Position{X: -96622, Y: 261000, Z: -3610}
	// spotCruma is on the first floor of Cruma Tower, next to the Porta
	// spawned at (12530, 117248, -12084) (spawns/Others/20_21.xml).
	spotCruma = testclient.Position{X: 12580, Y: 117248, Z: -12084}
)

// ferryWait bounds the wait for a ferry departure: the Talking Island to
// Gludin cycle takes about 54 minutes.
const ferryWait = time.Hour

func init() { scenarios = append(scenarios, worldScenarios...) }

// worldScenarios are the Milestone 10 scenarios, listed after the others.
var worldScenarios = []Scenario{
	{
		Name:        "door-open-close",
		Description: "GM setup: next to a Gludin clan hall door; enter (StaticObjectInfo, DoorStatusUpdate), //open it, //close it",
		Roles:       []string{"alice"},
		Run:         doorOpenClose,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "static-object-click",
		Description: "GM setup: next to a Talking Island town map; target it, click it again and see the map (ShowTownMap)",
		Roles:       []string{"alice"},
		Run:         staticObjectClick,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "raid-spawn",
		Description: "GM: invisible on open ground, //spawn Greyclaw Kutus; his NpcInfo with three guards and a pawn; //delete him (the minions go too)",
		Roles:       []string{"alice"},
		Run:         raidSpawn,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "raid-kill",
		Description: "GM: invisible on open ground, //spawn Greyclaw Kutus, //kill him: Die, the raid success message, the minions' DeleteObject",
		Roles:       []string{"alice"},
		Run:         raidKill,
		Ignore:      npcNoise,
		GM:          true,
	},
	{
		Name:        "ferry-departure",
		Description: "GM setup: on the Talking Island pier; see the ferry (VehicleInfo) and wait, up to an hour, for a departure (VehicleStarted, VehicleDeparture)",
		Roles:       []string{"alice"},
		Run:         ferryDeparture,
		// The calls and the ferry's place depend on when the run starts in
		// the ferry's cycle.
		Ignore: slices.Concat(npcNoise, []string{
			"VehicleInfo.X", "VehicleInfo.Y", "VehicleInfo.Z", "VehicleInfo.Heading", "CreatureSay", "PlaySound",
		}),
		GM: true,
	},
	{
		Name:        "cruma-heights",
		Description: "GM setup: 1,000 adena on the first floor of Cruma Tower (geodata on both servers); the monsters' heights, then a drop of 10 adena at alice's feet",
		Roles:       []string{"alice"},
		Run:         crumaHeights,
		// The heights are what this compares: NpcInfo.Z stays.
		Ignore: []string{"NpcInfo.X", "NpcInfo.Y", "NpcInfo.Heading", "SocialAction", "MoveToLocation"},
		GM:     true,
	},
}

// gmStep is a step that sends the GM command cmd.
func gmStep(p *Player, cmd string) Step {
	return Step{fmt.Sprintf("%s: //%s (GM)", p.Role, cmd), func(ctx context.Context) error {
		return p.Game().GMCommand(ctx, cmd)
	}}
}

// doorStep is a step in which p sends the door GM command cmd (open or
// close) for doorGludinHall and sees the door's new state.
func doorStep(p *Player, cmd string, closed bool) Step {
	return Step{fmt.Sprintf("%s: //%s %d (GM)", p.Role, cmd, doorGludinHall), func(ctx context.Context) error {
		if err := p.Game().GMCommand(ctx, fmt.Sprintf("%s %d", cmd, doorGludinHall)); err != nil {
			return err
		}
		_, err := testclient.Await(ctx, p.Game(), func(d testclient.DoorStatusUpdate) bool {
			return d.DoorID == doorGludinHall && d.Closed == closed
		})
		if err != nil {
			return gmHint(p, err)
		}
		return nil
	}}
}

func doorOpenClose(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, gmMoveTo(a, spotGludinDoor)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var door testclient.Door
	err := s.Steps(ctx,
		Step{"alice sees the door", func(ctx context.Context) error {
			return poll(ctx, func() bool {
				var ok bool
				door, ok = a.Game().World().FindDoor(doorGludinHall)
				return ok
			})
		}},
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	s.Note("door %d: closed %v, HP %d of %d", door.DoorID, door.Closed, door.CurHP, door.MaxHP)
	var steps []Step
	if !door.Closed {
		// A run before left it open.
		steps = append(steps, doorStep(a, "close", true), Idle(settle))
	}
	steps = append(steps,
		doorStep(a, "open", false),
		Idle(settle),
		doorStep(a, "close", true),
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
	return s.Steps(ctx, steps...)
}

func staticObjectClick(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, gmMoveTo(a, spotTownMap)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var obj testclient.StaticObject
	return s.Steps(ctx,
		Step{"alice sees the town map", func(ctx context.Context) error {
			return poll(ctx, func() bool {
				i := slices.IndexFunc(a.Game().World().StaticObjects(), func(o testclient.StaticObject) bool {
					return o.StaticID == staticTownMapTalking
				})
				if i < 0 {
					return false
				}
				obj = a.Game().World().StaticObjects()[i]
				return true
			})
		}},
		Idle(time.Second),
		Step{"alice targets the town map", func(ctx context.Context) error {
			_, err := a.Game().Target(ctx, obj.ObjectID)
			return err
		}},
		Step{"alice clicks it again and sees the map", func(ctx context.Context) error {
			if err := a.Game().Action(ctx, obj.ObjectID, false); err != nil {
				return err
			}
			m, err := testclient.Await[testclient.ShowTownMap](ctx, a.Game(), nil)
			if err == nil {
				s.Note("town map %s at %d, %d", m.Texture, m.X, m.Y)
			}
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

// kutusGroup returns the Kutus p sees that is not in skip, with his guards
// and pawns (those not in skip).
func kutusGroup(p *Player, skip map[int32]bool) (boss testclient.Object, minions []int32, guards, pawns int) {
	for _, o := range p.Game().World().Npcs() {
		if skip[o.ObjectID] || o.Dead {
			continue
		}
		switch o.NpcID {
		case npcKutus:
			boss = o
		case npcKutusGuard:
			guards++
			minions = append(minions, o.ObjectID)
		case npcKutusPawn:
			pawns++
			minions = append(minions, o.ObjectID)
		}
	}
	return boss, minions, guards, pawns
}

// spawnKutus makes p invisible (so that the group does not attack), spawns
// Kutus with //spawn and waits for him with his three guards and his pawn.
// It returns the boss and the minions' object IDs.
func spawnKutus(ctx context.Context, s *Session, p *Player) (testclient.Object, []int32, error) {
	var (
		boss    testclient.Object
		minions []int32
	)
	skip := map[int32]bool{}
	err := s.Steps(ctx,
		gmStep(p, "invis"),
		Idle(time.Second),
		Step{p.Role + " notes what it sees before the spawn", func(context.Context) error {
			for _, o := range p.Game().World().Npcs() {
				skip[o.ObjectID] = true
			}
			return nil
		}},
		gmStep(p, fmt.Sprintf("spawn %d", npcKutus)),
		Step{p.Role + " sees Kutus with three guards and a pawn", func(ctx context.Context) error {
			var guards, pawns int
			err := poll(ctx, func() bool {
				boss, minions, guards, pawns = kutusGroup(p, skip)
				return boss.ObjectID != 0 && guards == 3 && pawns == 1
			})
			s.Note("Kutus %v, %d guards, %d pawns", boss.ObjectID != 0, guards, pawns)
			if err != nil {
				return gmHint(p, err)
			}
			return nil
		}},
		// The NpcInfo of the group and their first moves settle.
		Idle(2*time.Second),
	)
	return boss, minions, err
}

// awaitGone waits until p has seen the DeleteObject of every object of ids.
func awaitGone(ctx context.Context, p *Player, ids []int32) error {
	left := map[int32]bool{}
	for _, id := range ids {
		left[id] = true
	}
	_, err := p.Game().WaitFor(ctx, func(pk testclient.Packet) bool {
		if d, ok := pk.(testclient.DeleteObject); ok {
			delete(left, d.ObjectID)
		}
		return len(left) == 0
	})
	if err != nil {
		return fmt.Errorf("%d objects still in sight: %w", len(left), err)
	}
	return nil
}

func raidSpawn(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, gmMoveTo(a, spotRaid)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	boss, minions, err := spawnKutus(ctx, s, a)
	if err != nil {
		return err
	}
	return s.Steps(ctx,
		Step{"alice targets Kutus", func(ctx context.Context) error {
			_, err := a.Game().Target(ctx, boss.ObjectID)
			return err
		}},
		gmStep(a, "delete"),
		Step{"Kutus and his minions vanish", func(ctx context.Context) error {
			return awaitGone(ctx, a, append([]int32{boss.ObjectID}, minions...))
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func raidKill(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, gmMoveTo(a, spotRaid)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	boss, minions, err := spawnKutus(ctx, s, a)
	if err != nil {
		return err
	}
	return s.Steps(ctx,
		Step{"alice targets Kutus", func(ctx context.Context) error {
			_, err := a.Game().Target(ctx, boss.ObjectID)
			return err
		}},
		gmStep(a, "kill"),
		Step{"Kutus dies, the raid succeeds, the minions vanish", func(ctx context.Context) error {
			died, success := false, false
			left := map[int32]bool{}
			for _, id := range minions {
				left[id] = true
			}
			_, err := a.Game().WaitFor(ctx, func(p testclient.Packet) bool {
				switch v := p.(type) {
				case testclient.Die:
					died = died || v.ObjectID == boss.ObjectID
				case testclient.SystemMessage:
					success = success || v.ID == msgRaidSuccessful
				case testclient.DeleteObject:
					delete(left, v.ObjectID)
				}
				return died && success && len(left) == 0
			})
			if err != nil {
				return fmt.Errorf("after //kill: Kutus died %v, raid success %v, %d minions left: %w", died, success, len(left), err)
			}
			return nil
		}},
		// The exp and the level-ups of the kill arrive.
		Idle(2*time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func ferryDeparture(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, gmMoveTo(a, spotPier)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	return s.Steps(ctx,
		Step{"alice sees the ferry", func(ctx context.Context) error {
			var b testclient.Boat
			err := poll(ctx, func() bool {
				var ok bool
				b, ok = a.Game().World().NearestBoat()
				return ok
			})
			if err == nil {
				s.Note("ferry %.0f units away, moving %v", b.Pos.DistanceTo(a.Game().World().Me().Pos), b.Moving)
			}
			return err
		}},
		Long(ferryWait, Step{"alice waits for a departure", func(ctx context.Context) error {
			d, err := testclient.Await[testclient.VehicleDeparture](ctx, a.Game(), nil)
			if err == nil {
				s.Note("departure at %d, %d speeds %d/%d", d.X, d.Y, d.MoveSpeed, d.RotationSpeed)
			}
			return err
		}}),
		Idle(2*time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func crumaHeights(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, giveItem(a, itemAdena, 1000), gmMoveTo(a, spotCruma)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var adena testclient.Item
	return s.Steps(ctx,
		// The monsters came with the enter-world sequence; invisible, alice
		// is left alone.
		gmStep(a, "invis"),
		Idle(2*time.Second),
		Step{"alice notes the monsters' heights", func(context.Context) error {
			var zs []int32
			for _, o := range a.Game().World().Npcs() {
				zs = append(zs, o.Pos.Z)
			}
			if len(zs) == 0 {
				return errors.New("no monster in sight")
			}
			s.Note("%d NPCs, heights %d to %d", len(zs), slices.Min(zs), slices.Max(zs))
			return nil
		}},
		findItems(a, map[int32]*testclient.Item{itemAdena: &adena}),
		Step{"alice drops 10 adena at her feet", func(ctx context.Context) error {
			me := a.Game().World().Me().Pos
			d, err := a.Game().DropItem(ctx, adena.ObjectID, 10, me.X, me.Y, me.Z)
			if err == nil {
				s.Note("the adena lie at height %d (alice at %d)", d.Z, me.Z)
			}
			return err
		}},
		Idle(settle),
		Step{"alice logs out", a.Logout},
	)
}
