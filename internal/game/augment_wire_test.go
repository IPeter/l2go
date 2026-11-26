package game

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/augment"
	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// The config's augment and fishing sections give the rules' Mobius defaults,
// and their keys reach the rules.
func TestAugmentConfigOfSection(t *testing.T) {
	if got := augmentConfig(config.DefaultAugment()); !reflect.DeepEqual(got, augment.DefaultConfig()) {
		t.Errorf("augmentConfig(DefaultAugment) = %+v, want %+v", got, augment.DefaultConfig())
	}
	c := config.DefaultAugment()
	c.TopGradeChance = []int{10, 20, 30, 40}
	c.BlackList = []int{57}
	c.AllowPvPItems, c.AllowDestroy = true, false
	got := augmentConfig(c)
	if got.GradeChances[3] != [4]int{10, 20, 30, 40} || got.GradeChances[0] != [4]int{55, 35, 7, 3} ||
		!slices.Equal(got.Blacklist, []int{57}) || !got.AllowPvP || got.AllowDestroy || got.AllowTrade {
		t.Errorf("augmentConfig = %+v", got)
	}
}

func TestFishingConfigOfSection(t *testing.T) {
	want := fishingConfig{allow: true, champ: fishing.DefaultChampionshipConfig()}
	if got := fishingConfigOf(config.DefaultFishing()); got != want {
		t.Errorf("fishingConfigOf(DefaultFishing) = %+v, want %+v", got, want)
	}
	c := config.Fishing{ChampionshipItem: 4037, ChampionshipReward1: 5, ChampionshipReward5: 1}
	want = fishingConfig{champ: fishing.ChampionshipConfig{RewardItem: 4037, Rewards: [fishing.ChampionshipRanks]int{5, 0, 0, 0, 1}}}
	if got := fishingConfigOf(c); got != want {
		t.Errorf("fishingConfigOf = %+v, want %+v", got, want)
	}
}

// TestCrystallizeAugmented: with altAllowAugmentDestroy off an augmented
// item is refused with ActionFailed and kept (Mobius RequestCrystallizeItem);
// a plain one is crystallized; with it on (the default) the augmented one
// is crystallized too.
func TestCrystallizeAugmented(t *testing.T) {
	e := newEnchantEnv(t, func(gs *config.GameServer) { gs.Augment.AllowDestroy = false })
	sess, p := e.enterWith("smith", tiStart,
		storage.Item{ItemID: enchCompositeArmor, Count: 1, Augmentation: int32(augment.Make(optPDef1545, optPassive))},
		enchanted(enchCompositeArmor, 0))
	e.onLoop(func() { p.skill.known.put(&skills.Skill{Skill: &data.Skill{ID: skillCrystallize, Level: 2}}) })
	var augmented, plain *inventory.Item
	e.onLoop(func() {
		for _, it := range p.inv.AllByItemID(enchCompositeArmor) {
			if it.Augmented() {
				augmented = it
			} else {
				plain = it
			}
		}
	})
	if augmented == nil || plain == nil {
		t.Fatal("the armors did not load")
	}
	crystallize := func(it *inventory.Item) []testclient.Packet {
		e.advance(2 * time.Second) // the transaction flood limit
		sess.take(t)
		return e.handle(sess, clientpackets.OpRequestCrystallizeItem, ints(it.ObjectID(), 1))
	}

	ps := crystallize(augmented)
	if got := testclient.Names(ps); !slices.Equal(got, []string{"ActionFailed"}) || e.countOf(p, enchCompositeArmor) != 2 {
		t.Errorf("augmented, destroy off: %q, %d armors", got, e.countOf(p, enchCompositeArmor))
	}
	// Nor is it destroyed (Mobius Item.isDestroyable).
	e.advance(2 * time.Second)
	sess.take(t)
	ps = e.handle(sess, clientpackets.OpRequestDestroyItem, ints(augmented.ObjectID(), 1))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgCannotDiscard)}) || e.countOf(p, enchCompositeArmor) != 2 {
		t.Errorf("destroy, destroy off: messages %v, %d armors", ids, e.countOf(p, enchCompositeArmor))
	}
	ps = crystallize(plain)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgS1Crystallized), int32(sysMsgEarnedS2S1)}) {
		t.Errorf("plain armor: messages %v", ids)
	}
	e.onLoop(func() { e.srv.augment.cfg.AllowDestroy = true })
	ps = crystallize(augmented)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgS1Crystallized), int32(sysMsgEarnedS2S1)}) || e.countOf(p, enchCompositeArmor) != 0 {
		t.Errorf("augmented, destroy on: messages %v, %d armors", ids, e.countOf(p, enchCompositeArmor))
	}
	if n := e.countOf(p, enchCrystalC); n != 2*576 {
		t.Errorf("%d C crystals, want %d", n, 2*576)
	}
}

// TestAugmentWhileFishing: a fishing player is told "You cannot augment
// items while fishing." (Mobius AbstractRefinePacket.isValid), through the
// fishing feature's OnAugmentCheck hook.
func TestAugmentWhileFishing(t *testing.T) {
	e, sess, p, _ := augmentEnv(t, 46)
	sword := objectOf(e, p, itemStormbringer)
	e.onLoop(func() { e.srv.fishing.states[p] = &fishState{group: -1} })
	var st augment.Player
	e.onLoop(func() { st = e.srv.augmentPlayer(p) })
	if !st.Fishing {
		t.Error("augmentPlayer: Fishing = false while fishing")
	}
	ps := exOn(e, sess, clientpackets.ExOpRequestConfirmTargetItem, sword)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{augment.MsgFishing, augment.MsgNotSuitable}) {
		t.Errorf("fishing: messages %v, want %d then %d", ids, augment.MsgFishing, augment.MsgNotSuitable)
	}
	if len(only[testclient.ExPutItemResultForVariationMake](ps)) != 0 {
		t.Error("fishing: the sword was accepted")
	}
	e.onLoop(func() { delete(e.srv.fishing.states, p) })
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmTargetItem, sword)
	if got := only[testclient.ExPutItemResultForVariationMake](ps); len(got) != 1 || got[0].Result != 1 {
		t.Errorf("after fishing: %+v %q", got, testclient.Names(ps))
	}
}

// TestFishingActionUseSocial: the social actions of the action window
// (RequestActionUse 12-66) play while idle and are refused while fishing
// with "You cannot do that while fishing." (Mobius tryBroadcastSocial).
func TestFishingActionUseSocial(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	sess, p := e.enterFisher(fisher{account: "waver", rod: true, lure: fishTestLure, lures: 5, skills: fishSkills})

	actionUse(e, sess, 12) // Greeting
	if got := only[testclient.SocialAction](sess.take(t)); len(got) != 1 || got[0] != (testclient.SocialAction{ObjectID: p.ObjectID(), ActionID: 2}) {
		t.Errorf("greeting: %+v, want social action 2", got)
	}
	actionUse(e, sess, 66) // Shyness
	if got := only[testclient.SocialAction](sess.take(t)); len(got) != 1 || got[0].ActionID != 15 {
		t.Errorf("shyness: %+v, want social action 15", got)
	}

	r := e.fishScript(rnd.Between(90, 250, 200), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
	castUse(e, sess, fishing.SkillFishing, false, false)
	sess.take(t)
	r.Done()
	actionUse(e, sess, 13) // Victory
	ps := sess.take(t)
	checkFish(t, "victory while fishing", ps, "msg 1471")
	if got := only[testclient.SocialAction](ps); len(got) != 0 {
		t.Errorf("a social action while fishing: %+v", got)
	}
}

// TestMultisellKeepsAugmentation: a list that keeps enchant levels carries
// the augmentation of the ingredient taken to the product (Mobius
// MultiSellChoose); a list that does not, gives a plain product.
func TestMultisellKeepsAugmentation(t *testing.T) {
	e := newExchangeEnv(t, nil)
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 60))
	aug := int32(augment.Make(optPDef1545, optActive3188))
	sess, p := e.enterWith("smith", tiVillage,
		storage.Item{ItemID: itemLongSword, Count: 1, Enchant: 3, Augmentation: aug},
		storage.Item{ItemID: itemLongSword, Count: 1, Augmentation: aug},
		storage.Item{ItemID: itemAdena, Count: 1000})

	e.exchangeBypass(sess, p, guide, "exc_multisell 990002")
	e.choose(sess, testListKeep, 100003, 1)
	var claymore, claymoreAug int32 = -1, -1
	e.onLoop(func() {
		if it := p.inv.ByItemID(itemClaymore); it != nil {
			claymore, claymoreAug = it.Enchant(), it.Augmentation()
		}
	})
	if claymore != 3 || claymoreAug != aug {
		t.Errorf("claymore +%d augmentation %d, want +3 and %d", claymore, claymoreAug, aug)
	}

	// The plain view takes the +0 sword at exactly +0: its augmentation goes
	// to the +0 claymore too.
	e.exchangeBypass(sess, p, guide, "multisell 990002")
	e.choose(sess, testListKeep, 100000, 1)
	var augs []int32
	e.onLoop(func() {
		for _, it := range p.inv.AllByItemID(itemClaymore) {
			augs = append(augs, it.Augmentation())
		}
	})
	if !slices.Equal(augs, []int32{aug, aug}) || e.countOf(p, itemLongSword) != 0 {
		t.Errorf("claymore augmentations %v, swords %d; want two of %d and none", augs, e.countOf(p, itemLongSword), aug)
	}
}

// A list that does not keep enchant levels gives plain products: the
// ingredient's augmentation is lost, as in Mobius.
func TestMultisellPlainListDropsAugmentation(t *testing.T) {
	e := newExchangeEnv(t, nil)
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 60))
	sess, p := e.enterWith("smith", tiVillage,
		storage.Item{ItemID: itemLongSword, Count: 1, Augmentation: int32(augment.Make(optPDef1545, optActive3188))},
		storage.Item{ItemID: itemMithrilOre, Count: 2},
		storage.Item{ItemID: itemAdena, Count: 1000})
	e.exchangeBypass(sess, p, guide, "multisell 990001")
	e.choose(sess, testListBasic, 300000, 1) // the augmented sword for one ore
	e.choose(sess, testListBasic, 100000, 1) // two ore and 100 adena for a sword
	var augs []int32
	e.onLoop(func() {
		for _, it := range p.inv.AllByItemID(itemLongSword) {
			augs = append(augs, it.Augmentation())
		}
	})
	if !slices.Equal(augs, []int32{0}) {
		t.Errorf("long sword augmentations %v, want one plain sword", augs)
	}
}
