package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/testclient"
)

// petUseItem sends UseItem for an item.
func petUseItem(e *testEnv, sess *fakeSession, objectID int32) {
	w := packets.NewWriter(8)
	w.WriteInt32(objectID)
	w.WriteInt32(0)
	e.srv.Handle(sess, clientpackets.OpUseItem, w.Bytes())
	e.sync()
}

// petItemMove sends RequestGiveItemToPet or RequestGetItemFromPet.
func petItemMove(e *testEnv, sess *fakeSession, give bool, objectID, count int32) {
	w := packets.NewWriter(12)
	w.WriteInt32(objectID)
	w.WriteInt32(count)
	op := byte(clientpackets.OpRequestGiveItemToPet)
	if !give {
		w.WriteInt32(0)
		op = clientpackets.OpRequestGetItemFromPet
	}
	e.srv.Handle(sess, op, w.Bytes())
	e.sync()
}

// petObjectID returns the object ID of p's first item of the template.
func petObjectID(t *testing.T, e *testEnv, p *Player, itemID int32) int32 {
	t.Helper()
	var id int32
	e.onLoop(func() {
		if it := p.inv.ByItemID(itemID); it != nil {
			id = it.ObjectID()
		}
	})
	if id == 0 {
		t.Fatalf("no item %d", itemID)
	}
	return id
}

// petSummonFromCollar uses the wolf collar and waits for the wolf.
func petSummonFromCollar(t *testing.T, e *testEnv, sess *fakeSession, p *Player) *Summon {
	t.Helper()
	return petSummonFrom(t, e, sess, p, summonWolfCollar)
}

// petSummonFrom uses a collar and waits for its pet.
func petSummonFrom(t *testing.T, e *testEnv, sess *fakeSession, p *Player, collar int32) *Summon {
	t.Helper()
	petUseItem(e, sess, petObjectID(t, e, p, collar))
	for range 40 {
		e.advance(500 * time.Millisecond)
		if sm := summonOfPlayer(e, p); sm != nil {
			return sm
		}
	}
	t.Fatalf("no pet from the collar: %v", testclient.Names(sess.take(t)))
	return nil
}

// Q419's Wolf Collar brings out the wolf at level 15 with full meal; food
// given to it is eaten when it gets hungry; exp levels it up (the collar
// shows the level); action 19 sends it back and the collar brings it back
// with its level and items; items come back from it.
func TestPetCollarFeedLevelAndItems(t *testing.T) {
	e, store := newSummonEnv(t)
	sess, p := summonEnter(t, e, "wolfy", classHumanFighter, 20,
		storage.Item{ItemID: summonWolfCollar, Count: 1},
		storage.Item{ItemID: summonWolfFood, Count: 5})
	sm := petSummonFromCollar(t, e, sess, p)
	var level, fed, maxMeal int
	var kind summon.Kind
	e.onLoop(func() { level, fed, maxMeal, kind = sm.pet.level, sm.pet.fed, sm.pet.maxMeal(), sm.kind })
	if kind != summon.Pet || level != 15 || fed != maxMeal || maxMeal == 0 {
		t.Fatalf("pet %v level %d fed %d/%d", kind, level, fed, maxMeal)
	}
	ps := sess.take(t)
	infos := only[testclient.PetInfo](ps)
	if len(infos) == 0 || !infos[0].IsPet() || infos[0].NpcID() != summonWolf || infos[0].Level != 15 {
		t.Fatalf("PetInfo %+v", infos)
	}
	if len(only[testclient.PetItemList](ps)) == 0 {
		t.Fatalf("no PetItemList: %v", testclient.Names(ps))
	}
	if err := e.srv.WaitSummonWrites(e.ctx); err != nil {
		t.Fatal(err)
	}
	collar := petObjectID(t, e, p, summonWolfCollar)
	row, err := store.LoadPet(e.ctx, collar)
	if err != nil || row.Level != 15 || row.OwnerID != p.ObjectID() {
		t.Fatalf("stored pet %+v, %v", row, err)
	}

	// Food goes into the pet's inventory.
	food := petObjectID(t, e, p, summonWolfFood)
	petItemMove(e, sess, true, food, 5)
	ps = sess.take(t)
	if len(only[testclient.PetInventoryUpdate](ps)) == 0 {
		t.Fatalf("no PetInventoryUpdate: %v", testclient.Names(ps))
	}
	var inPet, inInv int64
	e.onLoop(func() {
		inPet = p.items.pet.Count(summonWolfFood, -1, true)
		inInv = p.inv.Count(summonWolfFood, -1, true)
	})
	if inPet != 5 || inInv != 0 {
		t.Fatalf("food: pet %d, owner %d", inPet, inInv)
	}

	// Hungry: it eats on the next feed tick.
	e.onLoop(func() { sm.pet.fed = 10 })
	e.advance(summon.FeedTick)
	ps = sess.take(t)
	if !slices.Contains(summonMessages(ps), int32(sumMsgPetWasHungryAte)) {
		t.Fatalf("the pet did not eat: %v", summonMessages(ps))
	}
	e.onLoop(func() { inPet, fed = p.items.pet.Count(summonWolfFood, -1, true), sm.pet.fed })
	if inPet != 4 || fed <= 10 {
		t.Fatalf("after eating: food %d, fed %d", inPet, fed)
	}

	// Exp: a level.
	e.onLoop(func() {
		c := e.srv.petCurve(sm)
		e.srv.petAddExp(sm, c.ExpFor(16)-sm.pet.exp)
		level = sm.pet.level
	})
	if level != 16 {
		t.Fatalf("level %d after the exp of level 16", level)
	}
	var enchant int32
	e.onLoop(func() { enchant = p.inv.ByObjectID(collar).Enchant() })
	if enchant != 16 {
		t.Fatalf("collar shows %d", enchant)
	}

	// Back into the collar, and out again.
	e.onLoop(func() { sm.pet.fed = sm.pet.maxMeal() })
	summonAction(e, sess, clientpackets.ActionUseUnsummonPet, false)
	if summonOfPlayer(e, p) != nil {
		t.Fatalf("the pet stayed: %v", summonMessages(sess.take(t)))
	}
	if len(only[testclient.PetDelete](sess.take(t))) != 1 {
		t.Fatal("no PetDelete")
	}
	e.advance(5 * time.Second)
	sm = petSummonFromCollar(t, e, sess, p)
	e.onLoop(func() { level = sm.pet.level })
	if level != 16 {
		t.Fatalf("the pet came back at level %d", level)
	}
	ps = sess.take(t)
	lists := only[testclient.PetItemList](ps)
	if len(lists) == 0 {
		t.Fatalf("no PetItemList: %v", testclient.Names(ps))
	}
	if it, ok := lists[len(lists)-1].FindItem(summonWolfFood); !ok || it.Count != 4 {
		t.Fatalf("pet items %+v", lists[len(lists)-1])
	}

	// The food comes back.
	var petFood int32
	e.onLoop(func() { petFood = p.items.pet.ByItemID(summonWolfFood).ObjectID() })
	petItemMove(e, sess, false, petFood, 4)
	e.onLoop(func() {
		inPet = p.items.pet.Count(summonWolfFood, -1, true)
		inInv = p.inv.Count(summonWolfFood, -1, true)
	})
	if inPet != 0 || inInv != 4 {
		t.Fatalf("food back: pet %d, owner %d", inPet, inInv)
	}
}

// A pet the owner logs out with comes back at login with its level and
// meal; a starving pet does not obey.
func TestPetRestoreOnReconnect(t *testing.T) {
	e, _ := newSummonEnv(t)
	sess, p := summonEnter(t, e, "relog", classHumanFighter, 20,
		storage.Item{ItemID: summonWolfCollar, Count: 1})
	sm := petSummonFromCollar(t, e, sess, p)
	e.onLoop(func() {
		e.srv.petAddExp(sm, e.srv.petCurve(sm).ExpFor(17)-sm.pet.exp)
		sm.pet.fed = 100
	})
	e.advance(time.Second)
	e.srv.Disconnected(sess)
	e.sync()
	if err := e.srv.WaitSummonWrites(e.ctx); err != nil {
		t.Fatal(err)
	}
	sess2 := newSession("relog")
	p2 := e.enter(sess2, 0)
	sm2 := summonOfPlayer(e, p2)
	if sm2 == nil {
		t.Fatal("the pet did not come back")
	}
	var level, fed int
	e.onLoop(func() { level, fed = sm2.pet.level, sm2.pet.fed })
	if level != 17 || fed != 100 {
		t.Fatalf("restored level %d fed %d", level, fed)
	}

	// Starving: it does not obey.
	e.onLoop(func() { sm2.pet.fed = 0 })
	sess2.take(t)
	summonAction(e, sess2, clientpackets.ActionUsePetStop, false)
	if !slices.Contains(summonMessages(sess2.take(t)), int32(sumMsgPetUnresponsive)) {
		t.Fatal("a starving pet obeyed")
	}
}

// A dead pet's corpse decays after its corpse time: its items go back to
// the owner, the collar disappears and the pet's row is deleted.
func TestPetDeathDecay(t *testing.T) {
	e, store := newSummonEnv(t)
	sess, p := summonEnter(t, e, "rip", classHumanFighter, 20,
		storage.Item{ItemID: summonWolfCollar, Count: 1},
		storage.Item{ItemID: summonWolfFood, Count: 3})
	sm := petSummonFromCollar(t, e, sess, p)
	collar := petObjectID(t, e, p, summonWolfCollar)
	petItemMove(e, sess, true, petObjectID(t, e, p, summonWolfFood), 3)
	var exp int64
	e.onLoop(func() {
		exp = sm.pet.exp
		e.srv.Kill(sm.npc, nil)
	})
	ps := sess.take(t)
	if !slices.Contains(summonMessages(ps), int32(sumMsgPetKilledResurrect)) {
		t.Fatalf("no death message: %v", summonMessages(ps))
	}
	var after int64
	e.onLoop(func() { after = sm.pet.exp })
	if after >= exp {
		t.Fatalf("no exp lost: %d → %d", exp, after)
	}
	var corpse time.Duration
	e.onLoop(func() { corpse = e.srv.corpseTime(sm.npc) })
	e.advance(corpse + time.Second)
	if summonOfPlayer(e, p) != nil {
		t.Fatal("the corpse did not decay")
	}
	var hasCollar bool
	var food int64
	e.onLoop(func() {
		hasCollar = p.inv.ByObjectID(collar) != nil
		food = p.inv.Count(summonWolfFood, -1, true)
	})
	if hasCollar || food != 3 {
		t.Fatalf("collar kept %v, food back %d", hasCollar, food)
	}
	if err := e.srv.WaitSummonWrites(e.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadPet(e.ctx, collar); err == nil {
		t.Fatal("the pet's row is still stored")
	}
}

// A Baby Buffalo heals its badly hurt owner on its cast task ("Your pet
// uses $s1."); out of support mode it does not.
func TestBabyPetHealsOwner(t *testing.T) {
	e, _ := newSummonEnv(t)
	sess, p := summonEnter(t, e, "baby", classHumanFighter, 20,
		storage.Item{ItemID: summonBuffaloPipe, Count: 1})
	sm := petSummonFrom(t, e, sess, p, summonBuffaloPipe)
	var kind summon.Kind
	var hasHeal bool
	e.onLoop(func() {
		kind = sm.kind
		hasHeal = sm.pet.baby != nil && sm.pet.baby.majorHeal != nil
	})
	if kind != summon.BabyPet || !hasHeal {
		t.Fatalf("baby pet %v, heal %v", kind, hasHeal)
	}
	sess.take(t)
	var maxHP float64
	e.onLoop(func() {
		maxHP = float64(p.MaxHP())
		p.SetCurrentHP(maxHP * 0.1)
	})
	healed := false
	for range 40 {
		e.advance(time.Second)
		if slices.Contains(summonMessages(sess.take(t)), int32(sumMsgYourPetUses)) {
			healed = true
			break
		}
	}
	if !healed {
		t.Fatal("the baby pet never healed its owner")
	}
	e.advance(10 * time.Second)
	var hp float64
	e.onLoop(func() { hp = p.CurHP })
	if hp <= maxHP*0.1+1 {
		t.Fatalf("owner HP %v of %v after the heal", hp, maxHP)
	}

	e.onLoop(func() {
		e.srv.summonSwitchMode(sm, babySwitchStance)
		p.SetCurrentHP(maxHP * 0.1)
	})
	sess.take(t)
	e.advance(20 * time.Second)
	if slices.Contains(summonMessages(sess.take(t)), int32(sumMsgYourPetUses)) {
		t.Fatal("a baby pet out of support mode healed")
	}
}
