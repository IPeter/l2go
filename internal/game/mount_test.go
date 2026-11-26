package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The mount test items: the Wind Strider's collar and its food.
const (
	mountTestCollar = 4422 // Dragonflute of Wind
	mountTestFood   = 5168 // Food for Strider
)

// mountAction sends action 38 (Mount/Dismount) and returns what p got.
func mountAction(t *testing.T, e *testEnv, sess *fakeSession) []testclient.Packet {
	t.Helper()
	sess.take(t)
	summonAction(e, sess, clientpackets.ActionUseMountDismount, false)
	return sess.take(t)
}

// mountState reads p's mount on the loop.
func mountState(e *testEnv, p *Player) playerMount {
	var m playerMount
	e.onLoop(func() { m = p.mount })
	return m
}

// mountRideSpeed is the strider's run speed on ride at a level.
func mountRideSpeed(t *testing.T, e *testEnv, npcID, level int) *data.PetLevel {
	t.Helper()
	pd, ok := e.srv.summons.data.Pet(npcID)
	if !ok {
		t.Fatalf("no pet data for %d", npcID)
	}
	l, ok := pd.Level(level)
	if !ok {
		t.Fatalf("no level %d for %d", level, npcID)
	}
	return l
}

// A strider pet is ridden with action 38: not while sitting or in battle;
// then the weapon comes off, everyone sees Ride and the rider's info with
// the strider's speed and no weapon glow, the pet goes back into its
// collar and the feed gauge shows its meal. A rider may not take up a
// weapon, destroy the collar, sit or use physical skills; the mount eats
// every 10 s and food when hungry, and a hungry mount slows the rider and
// may not be left. Action 38 again gets off; the strider keeps its meal.
func TestMountStriderRideAndDismount(t *testing.T) {
	e, store := newSummonEnv(t)
	sess, p := summonEnter(t, e, "rider", classHumanFighter, 40,
		storage.Item{ItemID: mountTestCollar, Count: 1},
		storage.Item{ItemID: mountTestFood, Count: 2},
		storage.Item{ItemID: itemSquireSword, Count: 1, Slot: storage.PaperdollRHand})
	sm := petSummonFrom(t, e, sess, p, mountTestCollar)
	var npcID, level, fed int
	e.onLoop(func() { npcID, level, fed = sm.npc.Template.ID, sm.pet.level, sm.pet.fed })
	if npcID != mountNpcStrider {
		t.Fatalf("the collar brought out %d", npcID)
	}
	collar := petObjectID(t, e, p, mountTestCollar)
	sword := petObjectID(t, e, p, itemSquireSword)

	// Sitting, then in battle: refused with ActionFailed and the message.
	e.onLoop(func() { p.Sitting = true })
	ps := mountAction(t, e, sess)
	if len(only[testclient.ActionFailed](ps)) != 1 || !slices.Equal(summonMessages(ps), []int32{int32(mountMsgOnlyStanding)}) {
		t.Errorf("sitting: %v %v", testclient.Names(ps), summonMessages(ps))
	}
	e.onLoop(func() { p.Sitting = false; e.srv.StartAutoAttack(p) })
	ps = mountAction(t, e, sess)
	if !slices.Equal(summonMessages(ps), []int32{int32(mountMsgStriderInBattle)}) {
		t.Errorf("in battle: %v", summonMessages(ps))
	}
	if m := mountState(e, p); m.typ != serverpackets.MountNone {
		t.Fatalf("mounted after a refusal: %+v", m)
	}
	e.advance(17 * time.Second) // the stance ends
	e.onLoop(func() { fed = sm.pet.fed })

	// On the strider.
	ps = mountAction(t, e, sess)
	e.advance(time.Second) // the skill list, the coalesced info
	ps = append(ps, sess.take(t)...)
	rides := only[testclient.Ride](ps)
	if len(rides) != 1 || !rides[0].Mounted || rides[0].MountType != testclient.MountStrider || rides[0].MountNpcID() != mountNpcStrider || rides[0].ObjectID != p.ObjectID() {
		t.Fatalf("Ride %+v", rides)
	}
	if len(only[testclient.PetDelete](ps)) != 1 || summonOfPlayer(e, p) != nil {
		t.Error("the strider did not go back into its collar")
	}
	if !slices.Contains(summonMessages(ps), int32(sysMsgDisarmed)) {
		t.Errorf("no disarm message: %v", summonMessages(ps))
	}
	ride := mountRideSpeed(t, e, mountNpcStrider, level)
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 {
		t.Fatal("no UserInfo")
	}
	ui := uis[len(uis)-1]
	if ui.MountType != 1 || ui.MountNpcID != 1000000+mountNpcStrider || ui.RunSpd != int32(ride.RunOnRide) || ui.EnchantEffect != 0 || ui.FlyRunSpd != 0 {
		t.Errorf("UserInfo mount %d/%d run %d (want %v) glow %d fly %d", ui.MountType, ui.MountNpcID, ui.RunSpd, ride.RunOnRide, ui.EnchantEffect, ui.FlyRunSpd)
	}
	gauges := only[testclient.SetupGauge](ps)
	if len(gauges) == 0 || gauges[0].Color != int32(serverpackets.GaugeGreen) || gauges[0].MaxTime != int32(ride.MaxMeal*10000/ride.MealInNormal) {
		t.Errorf("feed gauge %+v", gauges)
	}
	lists := only[testclient.SkillList](ps)
	if len(lists) == 0 {
		t.Error("no SkillList")
	}
	m := mountState(e, p)
	if m.typ != serverpackets.MountStrider || m.npcID != mountNpcStrider || m.level != level || m.control != collar || m.feed != fed || !m.canFeed {
		t.Fatalf("mount %+v, fed %d", m, fed)
	}
	var worn bool
	var speed float64
	e.onLoop(func() { worn = p.inv.Paperdoll(storage.PaperdollRHand) != nil; speed = p.stats.RunSpeed() })
	if worn {
		t.Error("the sword stayed in the hand")
	}

	// What a rider may not do.
	sess.take(t)
	petUseItem(e, sess, sword)
	if ps := sess.take(t); !slices.Contains(summonMessages(ps), int32(sysMsgEquipConditionNotMet)) {
		t.Errorf("equipping the sword: %v", summonMessages(ps))
	}
	e.srv.Handle(sess, clientpackets.OpRequestDestroyItem, ints(collar, 1))
	e.sync()
	if ps := sess.take(t); !slices.Equal(summonMessages(ps), []int32{int32(mountMsgPetSummonedNoDestroy)}) {
		t.Errorf("destroying the collar: %v", summonMessages(ps))
	}
	summonAction(e, sess, clientpackets.ActionUseSitStand, false)
	e.onLoop(func() {
		worn = p.inv.Paperdoll(storage.PaperdollRHand) != nil
		if p.Sitting {
			t.Error("the rider sat down")
		}
		if p.inv.ByObjectID(collar) == nil {
			t.Error("the collar is gone")
		}
		if !p.skill.RidingStrider() {
			t.Error("Strider Siege Assault's condition does not see the strider")
		}
		if phys, ok := e.srv.Skill(3, 1); ok && mountSkillAllowed(p, phys) {
			t.Error("a physical skill is allowed on a strider")
		}
	})
	if worn {
		t.Error("the rider took up the sword")
	}

	// The feed: a tick eats the meal; hungry, the rider is slowed, may not
	// get off, and the mount eats food.
	e.advance(mountFeedTick)
	if m := mountState(e, p); m.feed != fed-ride.MealInNormal {
		t.Errorf("feed after a tick %d, want %d", m.feed, fed-ride.MealInNormal)
	}
	var hungrySpeed float64
	e.onLoop(func() {
		e.srv.mountSetFeed(p, 10)
		hungrySpeed = p.stats.RunSpeed()
	})
	if hungrySpeed != speed/2 {
		t.Errorf("hungry speed %v, want %v", hungrySpeed, speed/2)
	}
	ps = mountAction(t, e, sess)
	if !slices.Equal(summonMessages(ps), []int32{int32(mountMsgHungryStrider)}) {
		t.Errorf("getting off hungry: %v", summonMessages(ps))
	}
	e.advance(mountFeedTick)
	ps = sess.take(t)
	if !slices.Contains(summonMessages(ps), int32(sumMsgPetWasHungryAte)) || len(only[testclient.MagicSkillUse](ps)) == 0 {
		t.Errorf("the mount did not eat: %v", summonMessages(ps))
	}
	var food int64
	e.onLoop(func() { food = p.inv.Count(mountTestFood, -1, true) })
	if m := mountState(e, p); food != 1 || m.feed <= 10-ride.MealInNormal {
		t.Errorf("after eating: food %d, feed %d", food, m.feed)
	}

	// Off the strider: Ride, no mount in the info, and the meal goes to the
	// strider's row.
	e.onLoop(func() { e.srv.mountSetFeed(p, 1000) })
	want := mountState(e, p).feed
	ps = mountAction(t, e, sess)
	e.advance(time.Second)
	ps = append(ps, sess.take(t)...)
	rides = only[testclient.Ride](ps)
	if len(rides) != 1 || rides[0].Mounted || rides[0].MountType != 0 || rides[0].MountNpcID() != 0 {
		t.Errorf("Ride off %+v", rides)
	}
	uis = only[testclient.UserInfo](ps)
	if len(uis) == 0 || uis[len(uis)-1].MountType != 0 || uis[len(uis)-1].MountNpcID != 0 {
		t.Errorf("UserInfo off %+v", uis)
	}
	if m := mountState(e, p); m.typ != serverpackets.MountNone || m.control != 0 || m.feedTimer != 0 {
		t.Errorf("mount off %+v", m)
	}
	if err := e.srv.WaitSummonWrites(e.ctx); err != nil {
		t.Fatal(err)
	}
	if row, err := store.LoadPet(e.ctx, collar); err != nil || int(row.Fed) != want {
		t.Errorf("stored strider fed %d, %v; want %d", row.Fed, err, want)
	}
	e.onLoop(func() { speed = p.stats.RunSpeed() })
	if speed == hungrySpeed*2 {
		t.Error("the rider keeps the strider's speed on foot")
	}
}

// //ride_wyvern puts the GM's target (or the GM) on a wyvern of its level:
// it flies (fly speeds in UserInfo), knows Wyvern Breath and only it, and
// does not eat; //ride_strider on a rider is refused; //unride gets off.
// The scripts' mount (the wyvern managers) feeds the wyvern; a no-landing
// zone forces a non-GM rider down after its delay and refuses action 38
// meanwhile.
func TestMountWyvernGMAndNoLanding(t *testing.T) {
	e, _ := newSummonEnv(t)
	sess, p := summonEnter(t, e, "lord", classHumanFighter, 60)
	e.onLoop(func() { p.char.AccessLevel = storage.AccessLevelAdmin })

	ps := adminRun(t, e, sess, p, "ride_wyvern")
	e.advance(time.Second)
	ps = append(ps, sess.take(t)...)
	m := mountState(e, p)
	if m.typ != serverpackets.MountWyvern || m.npcID != mountNpcWyvern || m.level != 60 || !m.flying || m.canFeed || m.feedTimer != 0 {
		t.Fatalf("GM wyvern %+v", m)
	}
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 {
		t.Fatal("no UserInfo")
	}
	ui := uis[len(uis)-1]
	ride := mountRideSpeed(t, e, mountNpcWyvern, 60)
	if ui.MountType != 2 || ui.RunSpd != int32(ride.RunOnRide) || ui.FlyRunSpd != ui.RunSpd || ui.FlyWalkSpd != ui.WalkSpd {
		t.Errorf("wyvern UserInfo mount %d run %d fly %d/%d", ui.MountType, ui.RunSpd, ui.FlyRunSpd, ui.FlyWalkSpd)
	}
	lists := only[testclient.SkillList](ps)
	if len(lists) == 0 {
		t.Fatal("no SkillList")
	}
	breath := false
	for _, sk := range lists[len(lists)-1].Skills {
		switch {
		case sk.ID == skillWyvernBreath:
			breath = !sk.Disabled
		case !sk.Passive && !sk.Disabled:
			t.Errorf("skill %d usable on a wyvern", sk.ID)
		}
	}
	if !breath {
		t.Error("no usable Wyvern Breath")
	}
	if msgs := sysSays(adminRun(t, e, sess, p, "ride_strider")); !slices.Equal(msgs, []string{"Target already have a summon."}) {
		t.Errorf("//ride_strider on a rider: %q", msgs)
	}
	ps = adminRun(t, e, sess, p, "unride")
	e.advance(time.Second)
	ps = append(ps, sess.take(t)...)
	if m := mountState(e, p); m.typ != serverpackets.MountNone || m.flying {
		t.Fatalf("after //unride %+v", m)
	}
	var known bool
	e.onLoop(func() { known = p.KnownSkill(skillWyvernBreath) != nil })
	if known || len(only[testclient.Ride](ps)) != 1 {
		t.Errorf("after //unride: breath %v, %v", known, testclient.Names(ps))
	}

	// The wyvern managers' mount: a fed wyvern; a no-landing zone.
	e.onLoop(func() {
		p.char.AccessLevel = storage.AccessLevelPlayer
		if !p.Mount(mountNpcWyvern, 0, true) {
			t.Error("player:mount refused")
		}
	})
	if m := mountState(e, p); !m.canFeed || m.feed != ride.MaxMeal || m.feedTimer == 0 || m.typ != serverpackets.MountWyvern {
		t.Fatalf("fed wyvern %+v", m)
	}
	sess.take(t)
	e.onLoop(func() {
		p.Zones.Enter(ZoneNoLanding)
		e.srv.mountZoneEntered(p, &data.Zone{Type: "NoLandingZone"})
	})
	if ps := sess.take(t); !slices.Equal(summonMessages(ps), []int32{int32(mountMsgWyvernNoEntry)}) {
		t.Errorf("entering a no-landing zone: %v", summonMessages(ps))
	}
	ps = mountAction(t, e, sess)
	if !slices.Equal(summonMessages(ps), []int32{int32(mountMsgNotAllowedToDismount)}) {
		t.Errorf("getting off over a no-landing zone: %v", summonMessages(ps))
	}
	e.advance(mountDismountDelay * time.Second)
	if m := mountState(e, p); m.typ != serverpackets.MountNone {
		t.Errorf("still on the wyvern after the delay: %+v", m)
	}
}

// A rider picking up a cursed weapon gets off first; its holder cannot
// mount (the weapon will not come off).
func TestMountCursedWeapon(t *testing.T) {
	e := newCursedEnv(t, newRecordingStore(), world.NewFakeClock(testStart), memstore.NewIDs())
	sess, p := e.enterChar(testChar{account: "hunter", name: "Hunter"})
	e.onLoop(func() {
		if !e.srv.mountNpc(p, mountNpcStrider, 0, false) {
			t.Error("cannot mount")
		}
	})
	g := e.cwDropFor(p, true)
	sess.take(t)
	e.onLoop(func() { e.srv.doPickUp(p, g) })
	e.advance(time.Second)
	ps := sess.take(t)
	rides := only[testclient.Ride](ps)
	if len(rides) != 1 || rides[0].Mounted {
		t.Errorf("Ride at the pickup %+v", rides)
	}
	var mounted, cursed, again bool
	e.onLoop(func() {
		mounted, cursed = e.srv.mounted(p), p.CursedWeaponEquipped()
		again = e.srv.mountNpc(p, mountNpcStrider, 0, false)
	})
	if mounted || !cursed {
		t.Errorf("after the pickup: mounted %v, cursed %v", mounted, cursed)
	}
	if again || mountState(e, p).typ != serverpackets.MountNone {
		t.Error("a cursed weapon holder mounted")
	}
}
