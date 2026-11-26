package scripting_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The castle NPC scripts (Plan 14 M11i), ported from Mobius ai/others,
// against the fake world: one castle, Gludio, owned by the Lions, whose
// leader is the lord. Every top-level name here starts with castle.

const castleNotAvailable = "This function is not available yet."

// castleLoad loads one castle script with the generated constant tables
// (SystemMessageId).
func castleLoad(t *testing.T, file string) *scripttest.Harness {
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
	return scripttest.LoadWithTables(t, scriptsDir, tables, "npcs/"+file+"/"+file+".lua")
}

// castleWorld is the castle, its lord, a member of the owner clan without
// privileges, an outsider and a GM.
type castleWorld struct {
	castle                     *scripttest.Castle
	clan                       *scripttest.Clan
	lord, member, outsider, gm *scripttest.Player
}

func newCastleWorld() *castleWorld {
	w := &castleWorld{
		lord:     &scripttest.Player{ObjectID: 0x10000001, Name: "Lord", Level: 60},
		member:   &scripttest.Player{ObjectID: 0x10000002, Name: "Member", Level: 50},
		outsider: &scripttest.Player{ObjectID: 0x10000003, Name: "Stranger", Level: 40},
		gm:       &scripttest.Player{ObjectID: 0x10000004, Name: "Admin", Level: 80, GM: true},
	}
	w.clan = &scripttest.Clan{ID: 0x30000001, Name: "Lions", Level: 5, Leader: w.lord, Members: []*scripttest.Player{w.lord, w.member}}
	w.lord.Clan, w.member.Clan = w.clan, w.clan
	w.castle = &scripttest.Castle{ID: 1, Name: "Gludio", Owner: w.clan, Period: 3,
		Doors: map[int]bool{19210001: false, 19210002: false, 19210005: false}}
	return w
}

func (w *castleWorld) npc(id int) *scripttest.Npc {
	return &scripttest.Npc{ObjectID: 0x60000000 + int32(id), ID: id, Name: "Castle NPC", Castle: w.castle, MP: 1000, MaxMP: 1000}
}

// castlePage checks that reply shows the Mobius page dir/want, that the page
// exists in data/html and that the reply fills every placeholder of it but
// the server's own (%objectId%); it returns the filled page.
func castlePage(t *testing.T, what string, reply scripting.Reply, ok bool, dir, want string) string {
	t.Helper()
	full := "data/html/scripts/ai/others/" + dir + "/" + want
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != full {
		t.Fatalf("%s: reply %+v, %v; want page %s", what, reply, ok, full)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(full)))
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	page := strings.ReplaceAll(scripting.ReplaceVars(string(b), reply.Vars), "%objectId%", "1")
	if m := regexp.MustCompile(`%[A-Za-z_]+%`).FindString(page); m != "" {
		t.Errorf("%s: placeholder %s left in %s", what, m, want)
	}
	return page
}

func castleText(t *testing.T, what string, reply scripting.Reply, ok bool, want string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyText || reply.Content != want {
		t.Errorf("%s: reply %+v, %v; want the text %q", what, reply, ok, want)
	}
}

func TestCastleChamberlain(t *testing.T) {
	const (
		script = "CastleChamberlain"
		dir    = "CastleChamberlain"
	)
	h := castleLoad(t, "castle_chamberlain")
	cw := newCastleWorld()
	c := cw.castle
	npc := cw.npc(35100)
	ev := func(p *scripttest.Player, event string) (scripting.Reply, bool) {
		t.Helper()
		return h.Event(script, event, p, npc)
	}

	// First talk: the owner clan and GMs, everyone else is turned away.
	for _, p := range []*scripttest.Player{cw.lord, cw.member, cw.gm} {
		reply, ok := h.FirstTalk(p, npc)
		castlePage(t, p.Name+" first talk", reply, ok, dir, "chamberlain-01.html")
	}
	reply, ok := h.FirstTalk(cw.outsider, npc)
	castlePage(t, "outsider first talk", reply, ok, dir, "chamberlain-04.html")

	// The tax: the lord sets it within the Seal of Strife's limit; a member
	// without the vault privilege only sees it; outsiders get nothing.
	reply, ok = ev(cw.lord, "manage_tax")
	if page := castlePage(t, "manage_tax", reply, ok, dir, "castlesettaxrate.html"); !strings.Contains(page, "0% ~ 15%") {
		t.Errorf("manage_tax page without the 15%% limit:\n%s", page)
	}
	reply, ok = ev(cw.member, "manage_tax")
	castlePage(t, "member manage_tax", reply, ok, dir, "chamberlain-03.html")
	reply, ok = ev(cw.outsider, "manage_tax")
	castlePage(t, "outsider manage_tax", reply, ok, dir, "chamberlain-21.html")
	reply, ok = ev(cw.lord, "set_tax 20")
	castlePage(t, "set_tax 20", reply, ok, dir, "castletoohightaxrate.html")
	if c.TaxPercent != 0 {
		t.Errorf("tax %d after a refused rate", c.TaxPercent)
	}
	reply, ok = ev(cw.lord, "set_tax 10")
	castlePage(t, "set_tax 10", reply, ok, dir, "castleaftersettaxrate.html")
	if c.TaxPercent != 10 {
		t.Errorf("tax %d, want 10", c.TaxPercent)
	}
	c.StrifeOwner = 2 // Dawn: up to 25 %
	reply, ok = ev(cw.lord, "set_tax 20")
	castlePage(t, "set_tax 20 with Dawn", reply, ok, dir, "castleaftersettaxrate.html")
	c.StrifeOwner = 0
	cw.member.ClanPrivileges = []string{"CASTLE_VAULT"}
	reply, ok = ev(cw.member, "set_tax 5")
	castlePage(t, "member with the vault privilege", reply, ok, dir, "castleaftersettaxrate.html")
	cw.member.ClanPrivileges = nil
	reply, ok = ev(cw.member, "set_tax 5")
	castlePage(t, "member without the vault privilege", reply, ok, dir, "chamberlain-21.html")
	c.InProgress = true
	reply, ok = ev(cw.lord, "set_tax 5")
	castlePage(t, "set_tax in a siege", reply, ok, dir, "chamberlain-08.html")
	c.InProgress = false

	// The vault: deposit and withdraw move adena.
	c.Treasury = 1234567
	reply, ok = ev(cw.lord, "manage_vault")
	if page := castlePage(t, "manage_vault", reply, ok, dir, "castlemanagevault.html"); !strings.Contains(page, "1,234,567 Adena") {
		t.Errorf("manage_vault page:\n%s", page)
	}
	cw.lord.Give(57, 5000)
	reply, ok = ev(cw.lord, "deposit 1000")
	castlePage(t, "deposit", reply, ok, dir, "chamberlain-01.html")
	if cw.lord.Items[57] != 4000 || c.Treasury != 1235567 {
		t.Errorf("after the deposit: adena %d, treasury %d", cw.lord.Items[57], c.Treasury)
	}
	h.World().Messages = nil
	ev(cw.lord, "deposit 10000")
	if m := h.World().Messages; len(m) != 1 || m[0].ID != 279 || c.Treasury != 1235567 {
		t.Errorf("deposit beyond the adena: messages %+v, treasury %d", m, c.Treasury)
	}
	reply, ok = ev(cw.lord, "withdraw 567")
	castlePage(t, "withdraw", reply, ok, dir, "chamberlain-01.html")
	if cw.lord.Items[57] != 4567 || c.Treasury != 1235000 {
		t.Errorf("after the withdrawal: adena %d, treasury %d", cw.lord.Items[57], c.Treasury)
	}
	reply, ok = ev(cw.lord, "withdraw 2000000")
	if page := castlePage(t, "withdraw too much", reply, ok, dir, "castlenotenoughbalance.html"); !strings.Contains(page, "2,000,000") {
		t.Errorf("not enough balance page:\n%s", page)
	}
	reply, ok = ev(cw.outsider, "withdraw 1")
	castlePage(t, "outsider withdraw", reply, ok, dir, "chamberlain-21.html")

	// The report is for the lord alone.
	reply, ok = ev(cw.lord, "receive_report")
	page := castlePage(t, "receive_report", reply, ok, dir, "chamberlain-02.html")
	for _, want := range []string{"Lord", "Lions", "Gludio", "Seal Validation", "No Owner"} {
		if !strings.Contains(page, want) {
			t.Errorf("report without %q:\n%s", want, page)
		}
	}
	reply, ok = ev(cw.member, "receive_report")
	castlePage(t, "member receive_report", reply, ok, dir, "chamberlain-21.html")

	// Gates, banishing and the siege window.
	reply, ok = ev(cw.lord, "doors")
	castlePage(t, "doors", reply, ok, dir, "35100-d.html")
	reply, ok = ev(cw.lord, "operate_door 1 19210001 19210002")
	castlePage(t, "open the outer gates", reply, ok, dir, "chamberlain-05.html")
	if !c.Doors[19210001] || !c.Doors[19210002] || c.Doors[19210005] {
		t.Errorf("doors after opening the outer gates: %v", c.Doors)
	}
	reply, ok = ev(cw.lord, "operate_door 0 19210001")
	castlePage(t, "close a gate", reply, ok, dir, "chamberlain-06.html")
	if c.Doors[19210001] {
		t.Errorf("door 19210001 still open")
	}
	reply, ok = ev(cw.member, "doors")
	castlePage(t, "member doors", reply, ok, dir, "chamberlain-21.html")
	reply, ok = ev(cw.lord, "banish_foreigner")
	castlePage(t, "banish", reply, ok, dir, "chamberlain-11.html")
	if c.Banished != 1 {
		t.Errorf("banished %d times, want 1", c.Banished)
	}
	if reply, ok := ev(cw.lord, "list_siege_clans"); ok || !slices.Equal(c.SiegeInfo, []int32{cw.lord.ObjectID}) {
		t.Errorf("list_siege_clans: %+v, %v, siege info shown to %v", reply, ok, c.SiegeInfo)
	}

	// The castle functions: the pages, but no renting yet.
	reply, ok = ev(cw.lord, "manage_functions")
	castlePage(t, "manage_functions", reply, ok, dir, "chamberlain-23.html")
	reply, ok = ev(cw.lord, "siege_functions")
	castlePage(t, "siege_functions", reply, ok, dir, "chamberlain-26.html")
	reply, ok = ev(cw.lord, "recovery")
	castlePage(t, "recovery", reply, ok, dir, "castledeco-AR01.html")
	c.Functions = map[int]scripting.CastleFunction{2: {Level: 300, Lease: 12000, Rate: 604800000, EndDay: 3, EndMonth: 12, EndYear: 2026}}
	reply, ok = ev(cw.lord, "recovery")
	if page := castlePage(t, "recovery with HP", reply, ok, dir, "castledeco-AR01.html"); !strings.Contains(page, `p1="3" p2="12" p3="2026"`) {
		t.Errorf("recovery page without the end date:\n%s", page)
	}
	reply, ok = ev(cw.lord, "HP 300")
	castlePage(t, "HP 300 again", reply, ok, dir, "castledecoalreadyset.html")
	reply, ok = ev(cw.lord, "MP 40")
	if page := castlePage(t, "MP 40", reply, ok, dir, "castledeco-03.html"); !strings.Contains(page, `p1="45000" p2="7"`) {
		t.Errorf("MP 40 page without the fee:\n%s", page)
	}
	reply, ok = ev(cw.lord, "set_func 3 40")
	castleText(t, "set_func", reply, ok, castleNotAvailable)
	reply, ok = ev(cw.lord, "teleport")
	castlePage(t, "teleport", reply, ok, dir, "castlefuncdisabled.html")
	reply, ok = ev(cw.lord, "upgrade_doors 1 2 19210001 19210002")
	if page := castlePage(t, "upgrade_doors", reply, ok, dir, "chamberlain-14.html"); !strings.Contains(page, "3000000") {
		t.Errorf("upgrade_doors page without the price:\n%s", page)
	}
	// Door upgrades: too little adena, then both gates to ratio 2 for
	// 3,000,000; the same level again shows the current one; a door of
	// another castle answers nothing.
	reply, ok = ev(cw.lord, "upgrade_doors_confirm 1 2 19210001 19210002")
	castlePage(t, "upgrade_doors_confirm without the adena", reply, ok, dir, "chamberlain-09.html")
	cw.lord.Give(57, 3000000)
	adena := cw.lord.Items[57]
	reply, ok = ev(cw.lord, "upgrade_doors_confirm 1 2 19210001 19210002")
	castlePage(t, "upgrade_doors_confirm", reply, ok, dir, "chamberlain-16.html")
	if c.DoorUpgrades[19210001] != 2 || c.DoorUpgrades[19210002] != 2 || cw.lord.Items[57] != adena-3000000 {
		t.Errorf("after the upgrade: %v, adena %d", c.DoorUpgrades, cw.lord.Items[57])
	}
	reply, ok = ev(cw.lord, "upgrade_doors_confirm 1 2 19210001 19210002")
	if page := castlePage(t, "upgrade_doors_confirm again", reply, ok, dir, "chamberlain-15.html"); !strings.Contains(page, "2") {
		t.Errorf("the current level is missing:\n%s", page)
	}
	if reply, ok := ev(cw.lord, "upgrade_doors_confirm 1 3 23220001"); ok {
		t.Errorf("another castle's door: %+v", reply)
	}
	reply, ok = ev(cw.member, "upgrade_doors_confirm 1 3 19210001 19210002")
	castlePage(t, "member upgrade_doors_confirm", reply, ok, dir, "chamberlain-21.html")

	// Items: the general items buylist and the crown.
	reply, ok = ev(cw.lord, "products")
	if page := castlePage(t, "products", reply, ok, dir, "chamberlain-22.html"); !strings.Contains(page, "buy 351001") {
		t.Errorf("products page:\n%s", page)
	}
	ev(cw.lord, "buy 351001")
	if a := h.World().Actions; len(a) == 0 || a[len(a)-1] != (scripttest.Action{Kind: "buy_window", Object: npc.ObjectID, Target: cw.lord.ObjectID, Value: 351001, Level: 1}) {
		t.Errorf("buy: actions %+v", a)
	}
	reply, ok = ev(cw.lord, "give_crown")
	castlePage(t, "give_crown", reply, ok, dir, "chamberlain-25.html")
	if cw.lord.Items[6841] != 1 {
		t.Errorf("no crown: %v", cw.lord.Items)
	}
	reply, ok = ev(cw.lord, "give_crown")
	castlePage(t, "give_crown again", reply, ok, dir, "chamberlain-24.html")
	reply, ok = ev(cw.member, "give_crown")
	castlePage(t, "member give_crown", reply, ok, dir, "chamberlain-21.html")
	reply, ok = ev(cw.lord, "manors_cert")
	castlePage(t, "manors_cert", reply, ok, dir, "ssq_notdawnorevent.html")
	reply, ok = ev(cw.lord, "manor-help-02.html")
	castlePage(t, "manor help", reply, ok, dir, "manor-help-02.html")
	if reply, ok := ev(cw.lord, "siege_change"); ok {
		t.Errorf("siege_change answered %+v; Mobius has no handler", reply)
	}
}

func TestCastleSiegeManager(t *testing.T) {
	const dir = "CastleSiegeManager"
	h := castleLoad(t, "castle_siege_manager")
	cw := newCastleWorld()
	c := cw.castle
	npc := cw.npc(35104)

	reply, ok := h.FirstTalk(cw.lord, npc)
	castlePage(t, "lord in peace", reply, ok, dir, "CastleSiegeManager-01.html")
	reply, ok = h.FirstTalk(cw.member, npc)
	if !ok || reply.Kind != scripting.ReplyNone || !slices.Equal(c.SiegeInfo, []int32{cw.member.ObjectID}) {
		t.Errorf("member in peace: %+v, %v, siege info shown to %v; want the siege window", reply, ok, c.SiegeInfo)
	}
	c.InProgress = true
	reply, ok = h.FirstTalk(cw.lord, npc)
	castlePage(t, "lord in a siege", reply, ok, dir, "CastleSiegeManager.html")
	reply, ok = h.FirstTalk(cw.outsider, npc)
	castlePage(t, "outsider in a siege", reply, ok, dir, "CastleSiegeManager-02.html")
	c.InProgress = false

	// The clan hall siege managers wait for the clan halls.
	reply, ok = h.FirstTalk(cw.outsider, cw.npc(35420))
	castleText(t, "clan hall manager", reply, ok, castleNotAvailable)
}

func TestCastleMercenaryManager(t *testing.T) {
	const (
		script = "CastleMercenaryManager"
		dir    = "CastleMercenaryManager"
	)
	h := castleLoad(t, "castle_mercenary_manager")
	cw := newCastleWorld()
	c := cw.castle
	npc := cw.npc(35102)

	reply, ok := h.FirstTalk(cw.member, npc)
	castlePage(t, "member without the privilege", reply, ok, dir, "mercmanager-no.html")
	reply, ok = h.FirstTalk(cw.outsider, npc)
	castlePage(t, "outsider", reply, ok, dir, "mercmanager-no.html")
	for _, p := range []*scripttest.Player{cw.lord, cw.gm} {
		reply, ok = h.FirstTalk(p, npc)
		castlePage(t, p.Name, reply, ok, dir, "mercmanager.html")
	}
	cw.member.ClanPrivileges = []string{"CASTLE_MERCENARIES"}
	reply, ok = h.FirstTalk(cw.member, npc)
	castlePage(t, "member with the privilege", reply, ok, dir, "mercmanager.html")
	c.StrifeOwner = 1
	reply, ok = h.FirstTalk(cw.lord, npc)
	castlePage(t, "Dusk", reply, ok, dir, "mercmanager-dusk.html")
	c.StrifeOwner = 0
	c.InProgress = true
	reply, ok = h.FirstTalk(cw.lord, npc)
	castlePage(t, "in a siege", reply, ok, dir, "mercmanager-siege.html")
	reply, ok = h.Event(script, "main", cw.lord, npc)
	castlePage(t, "main in a siege", reply, ok, dir, "mercmanager-siege.html")
	c.InProgress = false

	reply, ok = h.Event(script, "limit", cw.lord, npc)
	if page := castlePage(t, "limit", reply, ok, dir, "mercmanager-limit.html"); !strings.Contains(page, "Gludio") {
		t.Errorf("limit page:\n%s", page)
	}
	c.Name = "Aden"
	reply, ok = h.Event(script, "limit", cw.lord, npc)
	castlePage(t, "Aden's limit", reply, ok, dir, "mercmanager-aden-limit.html")
	c.Name = "Gludio"

	// The tickets: the NPC's buylists, shown without the castle tax, in the
	// seal validation period only.
	if reply, ok := h.Event(script, "buy 2", cw.lord, npc); ok {
		t.Errorf("buy 2: %+v, %v", reply, ok)
	}
	want := scripttest.Action{Kind: "buy_window", Object: npc.ObjectID, Target: cw.lord.ObjectID, Value: 351022}
	if a := h.World().Actions; len(a) != 1 || a[0] != want {
		t.Errorf("buy 2: actions %+v, want %+v", a, want)
	}
	c.Period = 1
	reply, ok = h.Event(script, "buy 1", cw.lord, npc)
	castlePage(t, "buy in the competition", reply, ok, dir, "mercmanager-ssq.html")
}

func TestCastleTeleporter(t *testing.T) {
	const (
		script = "CastleTeleporter"
		dir    = "CastleTeleporter"
	)
	h := castleLoad(t, "castle_teleporter")
	cw := newCastleWorld()
	c := cw.castle
	npc := cw.npc(35095)

	reply, ok := h.FirstTalk(cw.outsider, npc)
	castlePage(t, "first talk", reply, ok, dir, "teleporter-01.html")
	reply, ok = h.Event(script, "TELEPORT_START", cw.lord, npc)
	castlePage(t, "start", reply, ok, dir, "teleporter-03.html")
	if npc.ScriptValue != 1 {
		t.Errorf("script value %d after the start, want 1", npc.ScriptValue)
	}
	reply, ok = h.FirstTalk(cw.outsider, npc)
	castlePage(t, "first talk while waiting", reply, ok, dir, "teleporter-03.html")

	h.Advance(29 * time.Second)
	if c.Ousted != 0 {
		t.Fatal("ousted before 30 seconds")
	}
	h.Advance(time.Second)
	if c.Ousted != 1 || npc.ScriptValue != 0 {
		t.Errorf("after 30 seconds: ousted %d, script value %d", c.Ousted, npc.ScriptValue)
	}
	want := scripttest.Action{Kind: "region_say", Object: npc.ObjectID, Value: 1, Text: "The defenders of Gludio castle will be teleported to the inner castle."}
	if a := h.World().Actions; !slices.Contains(a, want) {
		t.Errorf("actions %+v, want %+v", a, want)
	}

	// In a siege with every control tower down, the wait is 8 minutes.
	c.InProgress = true
	reply, ok = h.FirstTalk(cw.outsider, npc)
	castlePage(t, "towers down", reply, ok, dir, "teleporter-02.html")
	h.Event(script, "teleport_start", cw.lord, npc)
	h.Advance(7 * time.Minute)
	if c.Ousted != 1 {
		t.Fatal("ousted before 8 minutes")
	}
	h.Advance(time.Minute)
	if c.Ousted != 2 {
		t.Errorf("ousted %d times after 8 minutes, want 2", c.Ousted)
	}
}

func TestCastleWarehouse(t *testing.T) {
	const (
		script = "CastleWarehouse"
		dir    = "CastleWarehouse"
	)
	h := castleLoad(t, "castle_warehouse")
	cw := newCastleWorld()
	npc := cw.npc(35099)

	reply, ok := h.FirstTalk(cw.outsider, npc)
	castlePage(t, "first talk", reply, ok, dir, "warehouse-01.html")
	reply, ok = h.Event(script, "warehouse-02.html", cw.outsider, npc)
	if page := castlePage(t, "private warehouse", reply, ok, dir, "warehouse-02.html"); !strings.Contains(page, "_DepositP") {
		t.Errorf("warehouse-02:\n%s", page)
	}
	reply, ok = h.Event(script, "warehouse-04.html", cw.lord, npc)
	castlePage(t, "lord's blood alliances", reply, ok, dir, "warehouse-04.html")
	reply, ok = h.Event(script, "warehouse-04.html", cw.member, npc)
	castlePage(t, "member's blood alliances", reply, ok, dir, "warehouse-no.html")
	reply, ok = h.Event(script, "Receive", cw.lord, npc)
	castlePage(t, "receive none", reply, ok, dir, "warehouse-05.html")
	reply, ok = h.Event(script, "Exchange", cw.lord, npc)
	castlePage(t, "exchange without", reply, ok, dir, "warehouse-08.html")
	cw.lord.Give(9911, 1)
	reply, ok = h.Event(script, "Exchange", cw.lord, npc)
	castlePage(t, "exchange", reply, ok, dir, "warehouse-07.html")
	if cw.lord.Items[9911] != 0 || cw.lord.Items[9910] != 30 {
		t.Errorf("after the exchange: %v", cw.lord.Items)
	}
}

func TestCastleBlacksmith(t *testing.T) {
	const (
		script = "CastleBlacksmith"
		dir    = "CastleBlacksmith"
	)
	h := castleLoad(t, "castle_blacksmith")
	cw := newCastleWorld()
	npc := cw.npc(35098)

	for _, p := range []*scripttest.Player{cw.lord, cw.gm} {
		reply, ok := h.FirstTalk(p, npc)
		castlePage(t, p.Name, reply, ok, dir, "35098-01.html")
	}
	reply, ok := h.FirstTalk(cw.member, npc)
	castlePage(t, "member without the manor privilege", reply, ok, dir, "no.html")
	cw.member.ClanPrivileges = []string{"CASTLE_MANOR"}
	reply, ok = h.FirstTalk(cw.member, npc)
	castlePage(t, "member with the manor privilege", reply, ok, dir, "35098-01.html")
	reply, ok = h.FirstTalk(cw.outsider, npc)
	castlePage(t, "outsider", reply, ok, dir, "no.html")

	reply, ok = h.Event(script, "35098-02.html", cw.lord, npc)
	if page := castlePage(t, "crops", reply, ok, dir, "35098-02.html"); !strings.Contains(page, "_multisell 350980001") {
		t.Errorf("crop page:\n%s", page)
	}
	if reply, ok := h.Event(script, "35098-02.html", cw.outsider, npc); ok {
		t.Errorf("outsider got the crop page: %+v", reply)
	}
}

func TestCastleCourtMagician(t *testing.T) {
	const (
		script = "CastleCourtMagician"
		dir    = "CastleCourtMagician"
	)
	h := castleLoad(t, "castle_court_magician")
	cw := newCastleWorld()
	npc := cw.npc(35648)

	reply, ok := h.FirstTalk(cw.member, npc)
	castlePage(t, "member", reply, ok, dir, "courtmagician.html")
	reply, ok = h.FirstTalk(cw.outsider, npc)
	castlePage(t, "outsider", reply, ok, dir, "courtmagician-01.html")
	reply, ok = h.Event(script, "squadSkill", cw.outsider, npc)
	castlePage(t, "outsider squadSkill", reply, ok, dir, "courtmagician-01.html")
	reply, ok = h.Event(script, "squadSkill", cw.member, npc)
	castlePage(t, "squadSkill", reply, ok, dir, "courtmagician-05.html")

	// Talismans for 10 epaulettes.
	reply, ok = h.Event(script, "giveTalisman", cw.member, npc)
	castlePage(t, "no epaulettes", reply, ok, dir, "courtmagician-06.html")
	cw.member.Give(9912, 10)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 50), rnd.Get(46, 0)))
	reply, ok = h.Event(script, "giveTalisman", cw.member, npc)
	castlePage(t, "talisman", reply, ok, dir, "courtmagician-04.html")
	if cw.member.Items[9912] != 0 || cw.member.Items[9914] != 1 {
		t.Errorf("after the trade: %v", cw.member.Items)
	}

	// The clan gate.
	reply, ok = h.Event(script, "clanTeleport", cw.member, npc)
	castlePage(t, "no clan gate", reply, ok, dir, "courtmagician-02.html")
	cw.lord.Effects = []int{3632}
	cw.lord.X, cw.lord.Y, cw.lord.Z = 100, 200, 300
	if reply, ok := h.Event(script, "clanTeleport", cw.member, npc); ok {
		t.Errorf("clanTeleport: %+v, %v", reply, ok)
	}
	if cw.member.X != 100 || cw.member.Y != 200 || cw.member.Z != 300 {
		t.Errorf("member at %d,%d,%d; want at the leader", cw.member.X, cw.member.Y, cw.member.Z)
	}
}

func TestCastleWyvernManager(t *testing.T) {
	const (
		script = "WyvernManager"
		dir    = "WyvernManager"
	)
	h := castleLoad(t, "wyvern_manager")
	cw := newCastleWorld()
	c := cw.castle
	npc := cw.npc(35101)

	reply, ok := h.FirstTalk(cw.lord, npc)
	if page := castlePage(t, "lord", reply, ok, dir, "wyvernmanager-01.html"); !strings.Contains(page, "my lord Gludio") || !strings.Contains(page, "level 55") {
		t.Errorf("lord's page:\n%s", page)
	}
	reply, ok = h.FirstTalk(cw.member, npc)
	castlePage(t, "member", reply, ok, dir, "wyvernmanager-02.html")
	reply, ok = h.Event(script, "Return", cw.member, npc)
	castlePage(t, "member Return", reply, ok, dir, "wyvernmanager-02.html")
	reply, ok = h.Event(script, "Help", cw.member, npc)
	castlePage(t, "Help", reply, ok, dir, "wyvernmanager-03.html")

	// On foot: the strider message and its page.
	reply, ok = h.Event(script, "RideWyvern", cw.lord, npc)
	castlePage(t, "RideWyvern", reply, ok, dir, "wyvernmanager-05.html")
	if m := h.World().Messages; len(m) != 1 || m[0].ID != 1249 {
		t.Errorf("RideWyvern messages %+v", m)
	}

	// On a strider below level 55: the page without the message.
	cw.lord.Mount = &scripttest.Mount{Type: scripting.MountStrider, NpcID: 12526, Level: 54}
	reply, ok = h.Event(script, "RideWyvern", cw.lord, npc)
	castlePage(t, "RideWyvern low strider", reply, ok, dir, "wyvernmanager-05.html")
	if m := h.World().Messages; len(m) != 1 {
		t.Errorf("RideWyvern on a strider sent %+v", m[1:])
	}

	// A level 55 strider without the fee, or a member who is not the lord.
	cw.lord.Mount.Level = 55
	reply, ok = h.Event(script, "RideWyvern", cw.lord, npc)
	castlePage(t, "RideWyvern without crystals", reply, ok, dir, "wyvernmanager-06.html")
	cw.member.Mount = &scripttest.Mount{Type: scripting.MountStrider, NpcID: 12527, Level: 70}
	cw.member.Give(1460, 25)
	reply, ok = h.Event(script, "RideWyvern", cw.member, npc)
	castlePage(t, "member RideWyvern", reply, ok, dir, "wyvernmanager-06.html")
	if cw.member.Items[1460] != 25 || cw.member.Mount.NpcID != 12527 {
		t.Errorf("the member paid or changed mount: %d crystals, %+v", cw.member.Items[1460], cw.member.Mount)
	}

	// The lord pays 25 B-grade crystals and trades the strider for a wyvern
	// of his level that eats.
	cw.lord.Give(1460, 30)
	reply, ok = h.Event(script, "RideWyvern", cw.lord, npc)
	castlePage(t, "lord RideWyvern", reply, ok, dir, "wyvernmanager-04.html")
	if m := cw.lord.Mount; m == nil || m.Type != scripting.MountWyvern || m.NpcID != 12621 || m.Level != cw.lord.Level || m.ControlItem != 0 || !m.UseFood {
		t.Errorf("the lord rides %+v", m)
	}
	if cw.lord.Items[1460] != 5 {
		t.Errorf("the lord has %d crystals left, want 5", cw.lord.Items[1460])
	}
	cw.lord.Mount = nil

	// Dusk owning the Seal of Strife in the seal validation period.
	c.StrifeOwner = 1
	reply, ok = h.FirstTalk(cw.lord, npc)
	castlePage(t, "Dusk", reply, ok, dir, "wyvernmanager-dusk.html")
	reply, ok = h.Event(script, "RideWyvern", cw.lord, npc)
	castlePage(t, "RideWyvern for Dusk", reply, ok, dir, "wyvernmanager-dusk.html")
	c.StrifeOwner = 0

	// The clan hall managers: no hall has a lord yet.
	hall := cw.npc(35419)
	reply, ok = h.FirstTalk(cw.lord, hall)
	castlePage(t, "clan hall", reply, ok, dir, "wyvernmanager-02.html")
	reply, ok = h.Event(script, "Help", cw.lord, hall)
	castlePage(t, "clan hall Help", reply, ok, dir, "wyvernmanager-03b.html")
}
