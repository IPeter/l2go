package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// dialogPages are the HTML fixtures of the dialog tests.
var dialogPages = map[string]string{
	"teleporter/30006.htm": "<html><body>Gatekeeper %npcname%:<br>\n" +
		"\t<a action=\"bypass -h npc_%objectId%_Chat 1\">More</a><br>\n" +
		"<a action=\"bypass -h npc_%objectId%_showTeleports\">Teleport</a><br>\n" +
		"<a action=\"bypass -h npc_%objectId%_multisell 002\">Exchange</a><br>\n" +
		"<a action=\"bypass -h npc_%objectId%_showNoblesSelect\">Nobles</a><br>\n" +
		"<a action=\"bypass Script\">Quest</a><!-- the quest link -->\n" +
		"</body></html>\n",
	"teleporter/30006-1.htm":    `<html><body>More from %npcname%.<br><a action="link default/BlessingOfProtection.htm">Blessing</a></body></html>`,
	"teleporter/30006-pk.htm":   `<html><body>Go away! (%objectId%)</body></html>`,
	"teleporter/teleports.htm":  `<html><body>Region where teleporting is possible<br>%locations%<br></body></html>`,
	"teleporter/not_nobles.htm": `<html><body>%npcname%: nobles only.</body></html>`,

	"default/30598.htm": `<html><body>%npcname% (%objectId%)<br>` +
		`<a action="bypass -h npc_%objectId%_Link default/SupportMagic.htm">Support</a><br>` +
		`<a action="bypass -h npc_%objectId%_Link default/secret.htm">Secret</a><br>` +
		`<a action="bypass -h npc_%objectId%_Chat 2">Page two</a><br>` +
		`<a action="bypass -h player_help info.htm#1060">Help</a></body></html>`,
	"default/30598-2.htm":              `<html><body>Page two of %npcname%</body></html>`,
	"default/SupportMagic.htm":         `<html><body>Support at %objectId%<br><a action="bypass -h npc_%objectId%_Chat 0">Back</a></body></html>`,
	"default/BlessingOfProtection.htm": `<html><body>Blessing at %objectId%</body></html>`,
	"default/secret.htm":               `<html><body>Secret</body></html>`,
	"merchant/30001.htm":               `<html><body>%npcname%<br><a action="bypass -h npc_%objectId%_Buy 1">Buy</a><br><a action="bypass -h npc_%objectId%_Sell">Sell</a></body></html>`,
	"petmanager/30731.htm":             `<html><body>%npcname% %objectId%</body></html>`,
	"npcdefault.htm":                   `<html><body>I have nothing to say to you.</body></html>`,
	"noquest.htm":                      `<html><body>No quest here (%objectId%).</body></html>`,
	"help/info.htm":                    `<html><body>Help</body></html>`,
}

func TestNpcPagePath(t *testing.T) {
	w := testWorldData(t)
	exists := func(rel string) bool {
		return slices.Contains([]string{"default/30598.htm", "default/30598-2.htm"}, rel)
	}
	for _, tc := range []struct {
		typ   string
		id    int
		value int
		want  string
	}{
		{"Teleporter", 30006, 0, "teleporter/30006.htm"},
		{"Teleporter", 30006, 1, "teleporter/30006-1.htm"},
		{"DungeonGatekeeper", 31095, 0, "teleporter/31095.htm"},
		{"Merchant", 30001, 0, "merchant/30001.htm"},
		{"ClanHallManager", 35384, 3, "merchant/35384-3.htm"}, // a Merchant
		{"Fisherman", 31562, 0, "fisherman/31562.htm"},
		{"PetManager", 30731, 0, "petmanager/30731.htm"},
		{"Warehouse", 30005, 0, "warehouse/30005.htm"},
		{"Trainer", 30008, 2, "trainer/30008-2.htm"},
		{"VillageMasterFighter", 30066, 0, "villagemaster/30066.htm"},
		{"VillageMasterDElf", 30140, 0, "villagemaster/30140.htm"},
		{"Guard", 30733, 0, "guard/30733.htm"},
		{"QuestGuard", 31038, 0, "guard/31038.htm"},
		{"Adventurer", 31729, 0, "adventurer_guildsman/31729.htm"},
		{"SchemeBuffer", 1000003, 0, "mods/SchemeBuffer/1000003.htm"},
		{"Folk", 30598, 0, "default/30598.htm"},
		{"Folk", 30598, 2, "default/30598-2.htm"},
		{"Folk", 30598, 3, "npcdefault.htm"},
		{"Folk", 30008, 0, "npcdefault.htm"},
		{"Doorman", 30596, 0, "npcdefault.htm"},
	} {
		t.Run(tc.typ+"/"+tc.want, func(t *testing.T) {
			tmpl := &data.NpcTemplate{ID: tc.id, Type: tc.typ}
			if got := npcPagePath(w, tmpl, tc.value, exists); got != tc.want {
				t.Errorf("npcPagePath(%s %d, %d) = %q, want %q", tc.typ, tc.id, tc.value, got, tc.want)
			}
		})
	}
}

// wantPage checks a page: its NPC, and its text with placeholders already
// filled.
func wantPage(t *testing.T, what string, got testclient.NpcHtmlMessage, npc world.ObjectID, html string) {
	t.Helper()
	if got.ObjectID != int32(npc) || got.HTML != html || got.ItemID != 0 {
		t.Errorf("%s: page of %d %q, want of %d %q", what, got.ObjectID, got.HTML, npc, html)
	}
}

func TestFirstPagesByType(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})

	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	lector := e.spawnNpc(npcLector, eastOf(tiStart, 80))
	martin := e.spawnNpc(npcMartin, eastOf(tiStart, -80))
	roien := e.spawnNpc(30008, eastOf(tiStart, 100)) // a Folk without a default page
	id := func(n *Npc) string { return strconv.Itoa(int(n.ID())) }

	// The gatekeeper: its folder, the placeholders, tabs, line feeds and
	// comments gone, then ActionFailed.
	ps := e.talk(sess, p, roxxy)
	checkNames(t, "talking to the gatekeeper", ps, "NpcHtmlMessage", "ActionFailed")
	wantPage(t, "gatekeeper", onePage(t, ps), roxxy.ID(), "<html><body>Gatekeeper Roxxy:<br>"+
		`<a action="bypass -h npc_`+id(roxxy)+`_Chat 1">More</a><br>`+
		`<a action="bypass -h npc_`+id(roxxy)+`_showTeleports">Teleport</a><br>`+
		`<a action="bypass -h npc_`+id(roxxy)+`_multisell 002">Exchange</a><br>`+
		`<a action="bypass -h npc_`+id(roxxy)+`_showNoblesSelect">Nobles</a><br>`+
		`<a action="bypass Script">Quest</a></body></html>`)
	var last world.ObjectID
	e.onLoop(func() { last = p.LastNpc })
	if last != roxxy.ID() {
		t.Errorf("LastNpc = %d, want the gatekeeper %d", last, roxxy.ID())
	}

	// A Folk NPC: default/<id>.htm.
	ps = e.talk(sess, p, guide)
	if pg := onePage(t, ps); !strings.HasPrefix(pg.HTML, "<html><body>Newbie Guide ("+id(guide)+")<br>") {
		t.Errorf("newbie guide page %q", pg.HTML)
	}
	// A Folk NPC without a page of its own: npcdefault.htm.
	wantPage(t, "Folk without a page", onePage(t, e.talk(sess, p, roien)), roien.ID(), "<html><body>I have nothing to say to you.</body></html>")
	// A merchant whose page exists, and the pet manager (no ActionFailed).
	if pg := onePage(t, e.talk(sess, p, lector)); !strings.HasPrefix(pg.HTML, "<html><body>Lector<br>") {
		t.Errorf("merchant page %q", pg.HTML)
	}
	ps = e.talk(sess, p, martin)
	checkNames(t, "talking to the pet manager", ps, "NpcHtmlMessage")
	wantPage(t, "pet manager", onePage(t, ps), martin.ID(), "<html><body>Martin "+id(martin)+"</body></html>")
}

func TestMissingPage(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	wilford := e.spawnNpc(npcWilford, eastOf(tiStart, 50))
	ps := e.talk(sess, p, wilford)
	checkNames(t, "talking to the warehouse keeper", ps, "NpcHtmlMessage", "ActionFailed")
	wantPage(t, "missing page", onePage(t, ps), wilford.ID(), "<html><body>My Text is missing:<br>data/html/warehouse/30005.htm</body></html>")
}

func TestChatAndLinkPages(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	rid, gid := strconv.Itoa(int(roxxy.ID())), strconv.Itoa(int(guide.ID()))

	e.talk(sess, p, roxxy)
	// "Chat 1": page 1 of the gatekeeper, then the npc_ bypass's ActionFailed.
	ps := e.bypass(sess, "npc_"+rid+"_Chat 1")
	checkNames(t, "Chat 1", ps, "NpcHtmlMessage", "ActionFailed", "ActionFailed")
	wantPage(t, "Chat 1", onePage(t, ps), roxxy.ID(), `<html><body>More from Roxxy.<br><a action="link default/BlessingOfProtection.htm">Blessing</a></body></html>`)

	// The page's "link" opens a page as it is, without placeholders filled.
	ps = e.link(sess, "default/BlessingOfProtection.htm")
	checkNames(t, "RequestLinkHtml", ps, "NpcHtmlMessage")
	wantPage(t, "link", onePage(t, ps), roxxy.ID(), "<html><body>Blessing at %objectId%</body></html>")
	// That page had no links: the same link again is refused, and so is any
	// other.
	noPage(t, "a link no longer offered", e.link(sess, "default/BlessingOfProtection.htm"))
	noPage(t, "a link never offered", e.link(sess, "default/secret.htm"))
	noPage(t, "a link with ..", e.link(sess, "../secret.htm"))

	// "Link" opens the pages of Mobius's list with %objectId% filled.
	e.talk(sess, p, guide)
	ps = e.bypass(sess, "npc_"+gid+"_Link default/SupportMagic.htm")
	wantPage(t, "Link", onePage(t, ps), guide.ID(), `<html><body>Support at `+gid+`<br><a action="bypass -h npc_`+gid+`_Chat 0">Back</a></body></html>`)
	// Back to the first page with "Chat 0".
	if pg := onePage(t, e.bypass(sess, "npc_"+gid+"_Chat 0")); !strings.HasPrefix(pg.HTML, "<html><body>Newbie Guide") {
		t.Errorf("Chat 0 page %q", pg.HTML)
	}
	// Page 2 of a Folk NPC.
	wantPage(t, "Chat 2", onePage(t, e.bypass(sess, "npc_"+gid+"_Chat 2")), guide.ID(), "<html><body>Page two of Newbie Guide</body></html>")

	// A page outside Mobius's list gets a page without text.
	e.talk(sess, p, guide)
	ps = e.bypass(sess, "npc_"+gid+"_Link default/secret.htm")
	if pg := onePage(t, ps); pg.HTML != "" || pg.ObjectID != int32(guide.ID()) {
		t.Errorf("Link outside the list: page %+v, want an empty one", pg)
	}
	// ... which leaves the NPC page without links.
	noPage(t, "after the empty page", e.bypass(sess, "npc_"+gid+"_Chat 2"))
}

func TestPlayerHelpIsAnItemPage(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	e.talk(sess, p, guide)
	pg := onePage(t, e.bypass(sess, "player_help info.htm#1060"))
	if pg.ObjectID != 0 || pg.ItemID != 1060 || pg.HTML != "<html><body>Help</body></html>" {
		t.Errorf("player_help page %+v", pg)
	}
}

func TestBypassValidationAndRange(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	rid, gid := strconv.Itoa(int(roxxy.ID())), strconv.Itoa(int(guide.ID()))

	// Before any page, nothing is offered.
	if ps := e.bypass(sess, "npc_"+rid+"_Chat 1"); len(ps) != 0 {
		t.Errorf("bypass before any page answered %q", testclient.Names(ps))
	}
	e.talk(sess, p, roxxy)
	// A link of the page works; one that is not on it does nothing.
	if ps := e.bypass(sess, "npc_"+rid+"_Chat 2"); len(ps) != 0 {
		t.Errorf("bypass not on the page answered %q", testclient.Names(ps))
	}
	if ps := e.bypass(sess, "npc_"+gid+"_Chat 1"); len(ps) != 0 {
		t.Errorf("bypass of another NPC answered %q", testclient.Names(ps))
	}
	// A new page replaces the links of the last one.
	e.talk(sess, p, guide)
	if ps := e.bypass(sess, "npc_"+rid+"_Chat 1"); len(ps) != 0 {
		t.Errorf("link of the previous page answered %q", testclient.Names(ps))
	}

	// Out of the NPC's interaction range (250) the page's links do nothing.
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, eastOf(tiStart, -50+251)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	if ps := e.bypass(sess, "npc_"+gid+"_Chat 2"); len(ps) != 0 {
		t.Errorf("bypass out of range answered %q", testclient.Names(ps))
	}
	// Back in range they work again.
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, eastOf(tiStart, -50+250)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	onePage(t, e.bypass(sess, "npc_"+gid+"_Chat 2"))
}

func TestNotYetPages(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	lector := e.spawnNpc(npcLector, eastOf(tiStart, 50))
	lid := strconv.Itoa(int(lector.ID()))
	notYet := `<html><body>Lector:<br>I am sorry, this service is not available yet.<br><br>` +
		`<a action="bypass -h npc_` + lid + `_Chat 0">Back</a></body></html>`

	for _, cmd := range []string{"Buy 1", "Sell"} {
		e.talk(sess, p, lector)
		ps := e.bypass(sess, "npc_"+lid+"_"+cmd)
		checkNames(t, cmd, ps, "NpcHtmlMessage", "ActionFailed")
		wantPage(t, cmd, onePage(t, ps), lector.ID(), notYet)
	}
	// Its Back link returns to the first page.
	if pg := onePage(t, e.bypass(sess, "npc_"+lid+"_Chat 0")); !strings.HasPrefix(pg.HTML, "<html><body>Lector<br>") {
		t.Errorf("Back page %q", pg.HTML)
	}
}

func TestQuestWindowWithoutQuests(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	e.talk(sess, p, roxxy)
	// "Script" without the npc_ prefix goes to the page's NPC.
	ps := e.bypass(sess, "Script")
	checkNames(t, "the quest link", ps, "NpcHtmlMessage", "ActionFailed")
	wantPage(t, "no quest", onePage(t, ps), roxxy.ID(), "<html><body>No quest here ("+strconv.Itoa(int(roxxy.ID()))+").</body></html>")
}

func TestKarmaAtTheGatekeeper(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "pk", name: "Pk", pos: tiStart, karma: 100})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))

	// The gatekeeper's -pk page, %objectId% filled.
	ps := e.talk(sess, p, roxxy)
	checkNames(t, "a player with karma at the gatekeeper", ps, "NpcHtmlMessage", "ActionFailed")
	wantPage(t, "pk page", onePage(t, ps), roxxy.ID(), "<html><body>Go away! ("+strconv.Itoa(int(roxxy.ID()))+")</body></html>")
	// Other NPCs talk as usual.
	if pg := onePage(t, e.talk(sess, p, guide)); !strings.HasPrefix(pg.HTML, "<html><body>Newbie Guide") {
		t.Errorf("newbie guide page %q", pg.HTML)
	}
}

func TestEmptyBypassDisconnects(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart})
	e.bypass(sess, "")
	if final, _, closed := sess.closedWith(t); !closed || final == nil || final.PacketName() != "LeaveWorld" {
		t.Errorf("after an empty bypass: closed %v with %v, want LeaveWorld", closed, final)
	}
	if e.inWorld(p) {
		t.Error("the player is still in the world")
	}
	e.store.waitSave(t, p.ObjectID())
}

func TestAdminBypassRunsGMCommands(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	var got [][]string
	e.onLoop(func() {
		if err := e.srv.RegisterGMCommand("dialogtest_echo", storage.AccessLevelAdmin, func(p *Player, args []string) { got = append(got, args) }); err != nil {
			t.Errorf("RegisterGMCommand: %v", err)
		}
	})
	gmSess, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	e.onLoop(func() {
		e.srv.ShowHTML(gm, 0, `<html><body><a action="bypass -h admin_dialogtest_echo a b">Echo</a></body></html>`)
	})
	gmSess.take(t)
	e.bypass(gmSess, "admin_dialogtest_echo a b")
	e.bypass(gmSess, "admin_dialogtest_echo c") // not on the page
	var calls [][]string
	e.onLoop(func() { calls = slices.Clone(got) })
	if len(calls) != 1 || !slices.Equal(calls[0], []string{"a", "b"}) {
		t.Errorf("GM command calls %q, want [[a b]]", calls)
	}
}

func TestGMsSeeThePagePaths(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	sess.take(t)
	ps := e.talk(sess, gm, guide)
	says := only[testclient.CreatureSay](ps)
	if len(says) != 1 || says[0].Name != "HTML" || says[0].Text != "html/default/30598.htm" {
		t.Errorf("GM debug lines %+v, want one for html/default/30598.htm", says)
	}
	onePage(t, ps)
}
