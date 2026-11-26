package scripting_test

import (
	"path/filepath"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The Seven Signs priests (scripts/npcs/seven_signs, Mobius SignsPriest,
// DawnPriest, DuskPriest) against the fake world.

const ssqPath = "data/html/seven_signs/"

func ssqHarness(t *testing.T) (*scripttest.Harness, *scripttest.Player, *scripttest.Npc, *scripttest.Npc) {
	t.Helper()
	consts, err := data.LoadConstants(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	tables := make(map[string]map[string]any)
	for _, tb := range consts.Tables {
		m := make(map[string]any, tb.Len())
		for k, v := range tb.Ints {
			m[k] = v
		}
		for k, v := range tb.Strings {
			m[k] = v
		}
		tables[tb.Name] = m
	}
	h := scripttest.LoadWithTables(t, "../../scripts", tables, "npcs/seven_signs/signs_priest.lua")
	w := h.World()
	w.SevenSignsCycle() // makes the maps
	w.SevenSigns.Period = 1
	w.SevenSigns.Cycle = 1
	w.SevenSigns.Config = scripting.SevenSignsConfig{MaxPlayerContrib: 1000000, JoinDawnFee: 50000, ManorsAgreementID: 6388,
		CastleForDawn: true, CastleForDusk: true}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 40, ClassID: 1, Items: map[int]int64{57: 60000}}
	dawn := &scripttest.Npc{ObjectID: 2, ID: 31078, Types: []string{"DawnPriest", "SignsPriest"}}
	dusk := &scripttest.Npc{ObjectID: 3, ID: 31085, Types: []string{"DuskPriest", "SignsPriest"}}
	return h, hero, dawn, dusk
}

func ssqWant(t *testing.T, what string, r scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || r.Content != page {
		t.Errorf("%s: %+v (%v), want %s", what, r, ok, page)
	}
}

func TestSignsPriestFirstTalk(t *testing.T) {
	h, hero, dawn, dusk := ssqHarness(t)
	w := h.World()
	r, ok := h.FirstTalk(hero, dawn)
	ssqWant(t, "competition, no cabal", r, ok, ssqPath+"dawn_priest_1a.htm")
	w.SevenSigns.Cabals[hero.ObjectID] = 2
	r, ok = h.FirstTalk(hero, dawn)
	ssqWant(t, "competition, dawn", r, ok, ssqPath+"dawn_priest_1b.htm")
	r, ok = h.FirstTalk(hero, dusk)
	ssqWant(t, "dusk priest to a dawn member", r, ok, ssqPath+"dusk_priest_3b.htm")
	w.SevenSigns.Period, w.SevenSigns.Winner, w.SevenSigns.SealOwners[2] = 3, 2, 2
	r, ok = h.FirstTalk(hero, dawn)
	ssqWant(t, "validation, dawn won with gnosis", r, ok, ssqPath+"dawn_priest_2a.htm")
	w.SevenSigns.SealOwners[2] = 0
	r, ok = h.FirstTalk(hero, dawn)
	ssqWant(t, "validation, dawn won without gnosis", r, ok, ssqPath+"dawn_priest_2c.htm")
	w.SevenSigns.Period = 2
	r, ok = h.FirstTalk(hero, dawn)
	ssqWant(t, "results", r, ok, ssqPath+"dawn_priest_5.htm")
}

func TestSignsPriestJoin(t *testing.T) {
	h, hero, dawn, dusk := ssqHarness(t)
	w := h.World()
	// A first class (level 1) joins Dusk for strife: no fee.
	r, ok := h.Event("SignsPriest", "SevenSigns 33 1", hero, dusk)
	ssqWant(t, "participate", r, ok, ssqPath+"signs_33_dusk.htm")
	r, ok = h.Event("SignsPriest", "SevenSigns 4 1 3", hero, dusk)
	ssqWant(t, "joined", r, ok, ssqPath+"signs_4_dusk.htm")
	if w.SevenSigns.Cabals[1] != 1 || w.SevenSigns.Seals[1] != 3 {
		t.Errorf("cabal %d seal %d", w.SevenSigns.Cabals[1], w.SevenSigns.Seals[1])
	}
	r, ok = h.Event("SignsPriest", "SevenSigns 33 2", hero, dawn)
	ssqWant(t, "already a member", r, ok, ssqPath+"signs_33_dawn_member.htm")

	// A second class without a castle pays 50,000 adena to join Dawn.
	other := &scripttest.Player{ObjectID: 5, Name: "Other", Level: 50, ClassID: 2, Items: map[int]int64{57: 60000}}
	r, ok = h.Event("SignsPriest", "SevenSigns 33 2", other, dawn)
	ssqWant(t, "fee page", r, ok, ssqPath+"signs_33_dawn_fee.htm")
	r, ok = h.Event("SignsPriest", "SevenSigns 4 2 1", other, dawn)
	ssqWant(t, "dawn joined", r, ok, ssqPath+"signs_4_dawn.htm")
	if other.Items[57] != 10000 || w.SevenSigns.Cabals[5] != 2 {
		t.Errorf("adena %d cabal %d", other.Items[57], w.SevenSigns.Cabals[5])
	}
	// The certificate (6388) is taken instead of adena.
	third := &scripttest.Player{ObjectID: 6, Name: "Third", Level: 50, ClassID: 2, Items: map[int]int64{57: 10, 6388: 1}}
	h.Event("SignsPriest", "SevenSigns 4 2 2", third, dawn)
	if third.Items[6388] != 0 || w.SevenSigns.Cabals[6] != 2 {
		t.Errorf("certificate %d cabal %d", third.Items[6388], w.SevenSigns.Cabals[6])
	}
	// A starting class may not join; nor a wrong side.
	newbie := &scripttest.Player{ObjectID: 7, Name: "Newbie", Level: 10, ClassID: 0}
	r, ok = h.Event("SignsPriest", "SevenSigns 4 2 2", newbie, dawn)
	ssqWant(t, "first class", r, ok, ssqPath+"signs_33_dawn_firstclass.htm")
	h.Event("SignsPriest", "SevenSigns 4 1 2", other, dawn)
	if w.SevenSigns.Cabals[5] != 2 {
		t.Error("a crafted join changed the side")
	}
}

// The chamberlain sells a lord of Dawn the Lord of the Manor's Certificates
// in the competition, up to the castle's quantity (Mobius manors_cert).
func TestCastleChamberlainDawnTickets(t *testing.T) {
	h := castleLoad(t, "castle_chamberlain")
	cw := newCastleWorld()
	cw.castle.Period = 1
	w := h.World()
	w.SevenSignsCycle()
	w.SevenSigns.Config = scripting.SevenSignsConfig{DawnTicketQuantity: 20, DawnTicketPrice: 1000, DawnTicketBundle: 10, ManorsAgreementID: 6388}
	cw.lord.Items = map[int]int64{57: 25000}
	npc := cw.npc(35100)
	ev := func(event string) (scripting.Reply, bool) { return h.Event("CastleChamberlain", event, cw.lord, npc) }
	const dir = "CastleChamberlain"
	reply, ok := ev("manors_cert")
	castlePage(t, "not dawn", reply, ok, dir, "ssq_notdawnorevent.html")
	w.SevenSigns.Cabals[cw.lord.ObjectID] = 2
	reply, ok = ev("manors_cert")
	if page := castlePage(t, "sale", reply, ok, dir, "ssq_selldawnticket.html"); reply.Vars["DawnTicketLeft"] != "20" || reply.Vars["DawnTicketPrice"] != "10000" {
		t.Errorf("sale vars %v\n%s", reply.Vars, page)
	}
	ev("manors_cert_confirm")
	ev("manors_cert_confirm")
	if cw.lord.Items[6388] != 20 || cw.lord.Items[57] != 5000 || w.SevenSigns.Tickets[1] != 2 {
		t.Errorf("certificates %d adena %d sold %d", cw.lord.Items[6388], cw.lord.Items[57], w.SevenSigns.Tickets[1])
	}
	reply, ok = ev("manors_cert_confirm")
	castlePage(t, "sold out", reply, ok, dir, "ssq_notenoughticket.html")
}

func TestSignsPriestStones(t *testing.T) {
	h, hero, dawn, _ := ssqHarness(t)
	w := h.World()
	w.SevenSigns.Cabals[hero.ObjectID] = 2
	hero.Items[6362] = 10 // red
	hero.Items[6360] = 4  // blue
	r, ok := h.Event("SignsPriest", "SevenSigns 6 3", hero, dawn)
	if !ok || r.Content != ssqPath+"signs_6_dawn_contribute.htm" || r.Vars["stoneCount"] != "10" || r.Vars["stoneItemId"] != "6362" {
		t.Fatalf("contribute page %+v", r)
	}
	r, ok = h.Event("SignsPriest", "SevenSigns 21 6362 4", hero, dawn)
	ssqWant(t, "contributed", r, ok, ssqPath+"signs_6_dawn.htm")
	if hero.Items[6362] != 6 || w.SevenSigns.Contributions[1] != 40 {
		t.Errorf("red %d contribution %d", hero.Items[6362], w.SevenSigns.Contributions[1])
	}
	// All at once.
	h.Event("SignsPriest", "SevenSigns 6 4", hero, dawn)
	if hero.Items[6362] != 0 || hero.Items[6360] != 0 || w.SevenSigns.Contributions[1] != 40+60+12 {
		t.Errorf("after all: red %d blue %d contribution %d", hero.Items[6362], hero.Items[6360], w.SevenSigns.Contributions[1])
	}
	r, ok = h.Event("SignsPriest", "SevenSigns 6 4", hero, dawn)
	ssqWant(t, "no stones", r, ok, ssqPath+"signs_6_dawn_no_stones.htm")
	// Exchange for ancient adena.
	hero.Items[6361] = 3
	r, ok = h.Event("SignsPriest", "SevenSigns 18 6361 2", hero, dawn)
	ssqWant(t, "exchanged", r, ok, ssqPath+"signs_18_dawn.htm")
	if hero.Items[6361] != 1 || hero.Items[5575] != 10 {
		t.Errorf("green %d ancient adena %d", hero.Items[6361], hero.Items[5575])
	}
	// The reward in the seal validation period, for the winners.
	w.SevenSigns.Period, w.SevenSigns.Winner = 3, 2
	r, ok = h.Event("SignsPriest", "SevenSigns 9", hero, dawn)
	ssqWant(t, "reward", r, ok, ssqPath+"signs_9_dawn_a.htm")
	if hero.Items[5575] != 10+112 {
		t.Errorf("ancient adena %d after the reward", hero.Items[5575])
	}
	// The record.
	r, ok = h.Event("SignsPriest", "SevenSigns 2", hero, dawn)
	ssqWant(t, "record", r, ok, ssqPath+"signs_2_dawn.htm")
	if hero.Items[5707] != 1 || hero.Items[57] != 60000-500 {
		t.Errorf("record %d adena %d", hero.Items[5707], hero.Items[57])
	}
	r, ok = h.Event("SignsPriest", "SevenSigns 20", hero, dawn)
	if !ok || r.Kind != scripting.ReplyHTML {
		t.Errorf("seal status %+v", r)
	}
	r, ok = h.Event("SignsPriest", "SevenSignsDesc 14", hero, dawn)
	ssqWant(t, "description", r, ok, ssqPath+"desc_14.htm")
}
