package game

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The community board scenarios (Plan 14 M11o) run a server on the real
// CommunityBoard pages (data/html, copied by make data), the in-memory
// clan and board stores, and no castle data, so the region board uses its
// fallback castles.

// newBoardEnv starts a server for the board tests. With prev, it reuses
// prev's character store and IDs: a restart. board is the board store.
func newBoardEnv(t *testing.T, prev *testEnv, board storage.BBSStore, configure func(gs *config.GameServer)) *testEnv {
	t.Helper()
	d := testGameData(t)
	dataDir, err := filepath.Abs(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = dataDir
	if configure != nil {
		configure(gs)
	}
	store, ids := newRecordingStore(), memstore.NewIDs()
	if prev != nil {
		store, ids = prev.store, prev.ids
	}
	clock := world.NewFakeClock(testStart)
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		Characters:     store,
		Relations:      store.Relations(),
		Clans:          store.Clans(),
		IDs:            ids,
		Passives:       testPassives(t),
		Categories:     testCategories(t),
		BBS:            board,
		TimeZone:       time.UTC,
		Clock:          clock,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// showBoard presses Alt+B and returns what came back.
func (e *testEnv) showBoard(sess *fakeSession) []testclient.Packet {
	e.t.Helper()
	return e.clanSend(sess, clientpackets.OpRequestShowBoard, int32Body(0))
}

// boardWrite presses a "Write" button: url and up to five values.
func (e *testEnv) boardWrite(sess *fakeSession, url string, args ...string) []testclient.Packet {
	e.t.Helper()
	w := packets.NewWriter(128)
	w.WriteString(url)
	for i := range 5 {
		v := ""
		if i < len(args) {
			v = args[i]
		}
		w.WriteString(v)
	}
	sess.take(e.t)
	e.srv.Handle(sess, clientpackets.OpRequestBBSwrite, w.Bytes())
	e.sync()
	// A mail to a player not in the game looks the name up off the loop.
	e.srv.board.lookups.Wait()
	e.sync()
	return sess.take(e.t)
}

// boardWrites waits for the board's background writes.
func (e *testEnv) boardWrites() {
	e.t.Helper()
	if err := e.srv.board.writer.wait(e.ctx); err != nil {
		e.t.Fatalf("board writes: %v", err)
	}
}

// boardPage returns the board page in ps.
func boardPage(t *testing.T, what string, ps []testclient.Packet) string {
	t.Helper()
	html, ok := testclient.BoardPage(ps)
	if !ok {
		t.Fatalf("%s: no board page in %q (messages %v)", what, testclient.Names(ps), sysMsgs(ps))
	}
	return html
}

func wantContains(t *testing.T, what, page string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(page, w) {
			t.Errorf("%s lacks %q:\n%s", what, w, page)
		}
	}
}

func wantMsgs(t *testing.T, who string, ps []testclient.Packet, want ...int32) {
	t.Helper()
	if got := sysMsgs(ps); !slices.Equal(got, want) {
		t.Errorf("%s got messages %v, want %v", who, got, want)
	}
}

// boardChar enters the world with a new character.
func (e *testEnv) boardChar(account, name string, access storage.AccessLevel) (*fakeSession, *Player) {
	e.t.Helper()
	return e.enterClanChar(clanChar{account: account, name: name, level: 20, access: access})
}

// Demo 1: Alt+B shows the home page in three parts; a page goes into the
// favorites; the region page lists the 9 castles.
func TestBoardHomeFavoritesRegion(t *testing.T) {
	board := memstore.NewBBS()
	e := newBoardEnv(t, nil, board, nil)
	sa, pa := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)

	ps := e.showBoard(sa)
	boards := only[testclient.ShowBoard](ps)
	if len(boards) != 3 {
		t.Fatalf("Alt+B sent %d ShowBoard, want 3", len(boards))
	}
	for i, id := range []string{"101", "102", "103"} {
		gotID, text := boards[i].Part()
		if gotID != id || !boards[i].Show || boards[i].Tabs[0] != "bypass _bbshome" || boards[i].Tabs[7] != "bypass bbs_add_fav" {
			t.Errorf("ShowBoard %d: id %q, show %v, tabs %q", i, gotID, boards[i].Show, boards[i].Tabs)
		}
		if i > 0 && text != "null" {
			t.Errorf("part %s = %q, want Mobius's null", id, text)
		}
	}
	home := boardPage(t, "home", ps)
	wantContains(t, "home", home, `<a action="bypass _bbsgetfav">Favorites</a>`, `<a action="bypass _bbsloc">Regions</a>`)
	if strings.Contains(home, "%fav_count%") || strings.Contains(home, "%clan_count%") || strings.Contains(home, "%region_count%") {
		t.Errorf("placeholders left in the home page:\n%s", home)
	}

	// Bookmark the home page.
	favs := boardPage(t, "favorites", e.bypass(sa, "bbs_add_fav"))
	wantContains(t, "favorites", favs, `<a action="bypass _bbshome">Home</a>`, "2026-11-19 13:00:00", `_bbsdelfav_1"`)
	// Nothing to bookmark twice: the page is used up.
	if ps := e.bypass(sa, "bbs_add_fav"); len(ps) != 0 {
		t.Errorf("a second bbs_add_fav sent %q", testclient.Names(ps))
	}

	// The region list: 9 regions, no owners, no tax.
	regions := boardPage(t, "regions", e.bypass(sa, "_bbsloc"))
	for i, name := range []int{1049, 1052, 1053, 1057, 1060, 1059, 1248, 1247, 1056} {
		wantContains(t, "regions", regions, `<a action="bypass _bbsloc;`+strconv.Itoa(i)+`">&$`+strconv.Itoa(name)+`;</a>`)
	}
	if n := strings.Count(regions, "NPC"); n != 9 {
		t.Errorf("regions show %d NPC owners, want 9", n)
	}
	wantContains(t, "regions", regions, "0.0%")
	e.advance(time.Second)
	favs = boardPage(t, "favorites", e.bypass(sa, "bbs_add_fav"))
	if strings.Index(favs, ">Region</a>") > strings.Index(favs, ">Home</a>") {
		t.Errorf("favorites are not newest first:\n%s", favs)
	}
	var count int
	e.onLoop(func() { count = e.srv.board.favs.Count(pa.ObjectID()) })
	if count != 2 {
		t.Errorf("%d favorites, want 2", count)
	}

	// Delete the home favorite; the store follows.
	favs = boardPage(t, "favorites", e.bypass(sa, "_bbsdelfav_1"))
	if strings.Contains(favs, ">Home</a>") || !strings.Contains(favs, ">Region</a>") {
		t.Errorf("after deleting Home:\n%s", favs)
	}
	e.boardWrites()
	stored, err := board.LoadBBSFavorites(e.ctx)
	if err != nil || len(stored) != 1 || stored[0].Title != "Region" || stored[0].Bypass != "_bbsloc" || stored[0].PlayerID != pa.ObjectID() {
		t.Errorf("stored favorites %+v, %v", stored, err)
	}

	// The memo and homepage boards show their pages.
	wantContains(t, "memo", boardPage(t, "memo", e.bypass(sa, "_bbsmemo")), "Clan Recruitment Hall Forum")
	if _, ok := testclient.BoardPage(e.bypass(sa, "_bbslink")); !ok {
		t.Error("no homepage page")
	}
	// A clan command Mobius does not know, and a write URL it does not know.
	wantContains(t, "unknown clan command", boardPage(t, "clan", e.bypass(sa, "_bbsclan_x")), "Command _bbsclan_x needs development.")
	wantContains(t, "unknown write", boardPage(t, "write", e.boardWrite(sa, "Foo")), "The command: Foo is not implemented yet.")
	if ps := e.boardWrite(sa, "Topic", "x"); len(ps) != 0 {
		t.Errorf("Topic write sent %q", testclient.Names(ps))
	}
	// The drop search stays off: its bypass is no board command.
	if ps := e.bypass(sa, "_bbs_search_item"); len(ps) != 0 {
		t.Errorf("the drop search answered %q", testclient.Names(ps))
	}

	// A restart keeps the favorites.
	e.stop()
	e2 := newBoardEnv(t, e, board, nil)
	sa2, _ := e2.clanLogin("alice")
	favs = boardPage(t, "favorites after a restart", e2.bypass(sa2, "_bbsgetfav"))
	wantContains(t, "favorites after a restart", favs, ">Region</a>")
	// New IDs follow the stored ones.
	boardPage(t, "regions", e2.bypass(sa2, "_bbsloc"))
	wantContains(t, "favorites", boardPage(t, "favorites", e2.bypass(sa2, "bbs_add_fav")), "_bbsdelfav_3")
}

// The region page of one region: its castle and its clan halls in Mobius's
// HashMap order, the hall owner from the clan's hideout.
func TestBoardRegionPage(t *testing.T) {
	e := newBoardEnv(t, nil, nil, nil)
	sa, pa := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)
	c := e.foundClan(pa, "Knights")
	e.onLoop(func() { c.HideoutID = 32 })

	page := boardPage(t, "Dion", e.bypass(sa, "_bbsloc;1"))
	wantContains(t, "Dion", page, "Territory Name:", "<td width=\"240\">Dion</td>", "<td width=\"240\">NPC</td>", "01-01-1970 00:00",
		`<td width=200>The Scarlet Barracks</td><td width=200><a action="bypass _bbsclan_clanhome;`+strconv.Itoa(int(c.ID))+`">Knights</a></td><td width=200>Alice</td>`)
	if a, b, d := strings.Index(page, "Scarlet"), strings.Index(page, "Viridian"), strings.Index(page, "Atramental"); a > b || b > d {
		t.Errorf("Dion's halls are not in HashMap order (32, 33, 31):\n%s", page)
	}
	wantContains(t, "Oren", boardPage(t, "Oren", e.bypass(sa, "_bbsloc;3")), "There are no clan halls in this region.")
	for _, bad := range []string{"_bbsloc;9", "_bbsloc;-1", "_bbsloc;x"} {
		if ps := e.bypass(sa, bad); len(ps) != 0 {
			t.Errorf("%s sent %q", bad, testclient.Names(ps))
		}
	}
}

// Demo 2 (clan part): the clan page with the notice; the leader edits it.
func TestBoardClan(t *testing.T) {
	e := newBoardEnv(t, nil, memstore.NewBBS(), nil)
	sa, pa := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)
	sb, pb := e.boardChar("bob", "Bob", storage.AccessLevelPlayer)
	sc, _ := e.boardChar("cid", "Cid", storage.AccessLevelPlayer)
	c := e.foundClan(pa, "Knights")
	e.inviteToClan(sa, pa, sb, pb, 0, 1)
	sa.take(t)

	// Below clan level 2: the clan list, with the clan and a link to it.
	ps := e.bypass(sa, "_bbsclan")
	list := boardPage(t, "clan list", ps)
	wantContains(t, "clan list", list, `[GO TO MY CLAN]`, `>Knights</a>`, `<td FIXWIDTH=200 align=center>Alice</td>`)
	wantMsgs(t, "Alice", ps)
	ps = e.bypass(sa, "_bbsclan_clanhome;"+strconv.Itoa(int(c.ID)))
	wantMsgs(t, "Alice at a level 0 clan home", ps, int32(bbsMsgNoCommunity))
	boardPage(t, "clan list again", ps)

	e.onLoop(func() { c.Level = 2 })
	home := boardPage(t, "leader's clan home", e.bypass(sa, "_bbsclan"))
	wantContains(t, "leader's clan home", home, "[Clan Notice]", "CLAN MEMBERS</td><td fixWIDTH=195 align=left height=16>2</td>")
	if member := boardPage(t, "member's clan home", e.bypass(sb, "_bbsclan")); strings.Contains(member, "[Clan Notice]") {
		t.Error("a member sees the notice link")
	}
	// Cid, without a clan, gets the list.
	wantContains(t, "Cid's clan", boardPage(t, "Cid", e.bypass(sc, "_bbsclan")), `<img src="L2UI.SquareBlank" width="0" height="0">&nbsp;&nbsp;`)

	// The leader's notice page is sent whole with its multiedit filled.
	ps = e.bypass(sa, "_bbsclan_clannotice_edit;"+strconv.Itoa(int(c.ID))+";cnotice")
	boards := only[testclient.ShowBoard](ps)
	if len(boards) != 2 {
		t.Fatalf("notice edit sent %d ShowBoard", len(boards))
	}
	if id, html := boards[0].Part(); id != "1001" || !strings.Contains(html, "Write Notice Set _ Content Content Content") {
		t.Errorf("notice page %s: %s", id, html)
	}
	if id, _ := boards[1].Part(); id != "1002" || !strings.Contains(boards[1].Content, "\u0008Alice\u0008"+strconv.Itoa(int(pa.ObjectID()))+"\u0008alice\u00089\u0008") {
		t.Errorf("multiedit fill %q", boards[1].Content)
	}

	// Saving and enabling the notice; a member may do neither.
	wantMsgs(t, "Bob saving the notice", e.boardWrite(sb, "Notice", "Set", "_", "Hacked", "Hacked", "Hacked"))
	wantMsgs(t, "Alice saving the notice", e.boardWrite(sa, "Notice", "Set", "_", "Welcome", "Welcome", "Welcome"), int32(bbsMsgNoticeSaved))
	e.bypass(sb, "_bbsclan_clannotice_enable")
	var text string
	var enabled bool
	e.onLoop(func() { text, enabled = c.Notice.Text, c.Notice.Enabled })
	if text != "Welcome" || enabled {
		t.Fatalf("notice %q enabled %v after the writes, want Welcome, off", text, enabled)
	}
	ps = e.bypass(sa, "_bbsclan_clannotice_enable")
	if id, html := only[testclient.ShowBoard](ps)[0].Part(); id != "1001" || !strings.Contains(html, "_bbsclan_clannotice_disable") {
		t.Errorf("after enabling: %s %s", id, html)
	}
	member := boardPage(t, "member's notice", e.bypass(sb, "_bbsclan_clannotice_edit;"+strconv.Itoa(int(c.ID))))
	wantContains(t, "member's notice", member, "cannot change the clan notice", "<td FIXWIDTH=630 align=left>Welcome</td>")
	if st, err := e.storedClan(c.ID); err != nil || st.Notice.Text != "Welcome" || !st.Notice.Enabled {
		t.Errorf("stored notice %+v, %v", st.Notice, err)
	}

	// The search.
	found := boardPage(t, "search", e.boardWrite(sc, "Search", "-1", "0", "kni", "Clan Name", "Clan Name"))
	wantContains(t, "search", found, ">Knights</a>")
	wantContains(t, "empty search", boardPage(t, "search", e.boardWrite(sc, "Search", "-1", "0", "zed", "Clan Ruler")), "No rulers found matching the name: </font>zed")

	// The clan mail form is the leader's.
	ps = e.bypass(sb, "_bbsclan;mail")
	wantMsgs(t, "Bob at the clan mail", ps, int32(bbsMsgOnlyClanLeader))
	form := boardPage(t, "clan mail", e.bypass(sa, "_bbsclan;mail"))
	wantContains(t, "clan mail", form, "Knights Clan Members", "Write CMail mail "+strconv.Itoa(int(c.ID))+" Title Title Content")
	// Mailing the clan: Alice herself is an invalid target, Bob gets it.
	sb.take(t)
	ps = e.boardWrite(sa, "CMail", "mail", strconv.Itoa(int(c.ID)), "Title", "Meeting", "Tonight")
	wantMsgs(t, "Alice mailing the clan", ps, int32(bbsMsgInvalidTarget), int32(bbsMsgSentMail))
	wantContains(t, "clan home after the mail", boardPage(t, "clan home", ps), "CLAN NAME")
	got := sb.take(t)
	wantMsgs(t, "Bob", got, int32(bbsMsgGotMail))
	if _, ok := testclient.Find[testclient.ExMailArrived](got); !ok {
		t.Error("Bob got no ExMailArrived")
	}
}

// Demo 2 (mail part): mail reaches a player online with the arrival
// notice, and one offline at login; reading, replying, archiving,
// deleting, searching and the sending rules.
func TestBoardMail(t *testing.T) {
	board := memstore.NewBBS()
	e := newBoardEnv(t, nil, board, nil)
	sa, pa := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)
	sb, pb := e.boardChar("bob", "Bob", storage.AccessLevelPlayer)
	e.createClanChar(clanChar{account: "cid", name: "Cid", level: 20})
	e.boardChar("gm", "Gamemaster", storage.AccessLevelAdmin)
	sb.take(t)

	ps := e.boardWrite(sa, "Mail", "Send", "%postId%", "bob;Cid", "Hello <b>", "Line 1\nLine 2")
	wantMsgs(t, "Alice", ps, int32(bbsMsgSentMail))
	sent := boardPage(t, "sent box", ps)
	wantContains(t, "sent box", sent, "Sent Box&nbsp;(1)", `<a action="bypass _bbsmail">Inbox</a>&nbsp;(0)`, "<td width=150>Alice</td>", "Hello <b>")
	got := sb.take(t)
	wantMsgs(t, "Bob", got, int32(bbsMsgGotMail))
	if _, ok := testclient.Find[testclient.ExMailArrived](got); !ok {
		t.Error("Bob got no ExMailArrived")
	}

	// The mails are stored; after a restart, Cid gets the notice when
	// entering.
	e.boardWrites()
	stored, err := board.LoadBBSMails(context.Background())
	if err != nil || len(stored) != 3 {
		t.Fatalf("stored mails %+v, %v", stored, err)
	}
	if stored[2].Location != "sentbox" || stored[2].ReceiverID != pa.ObjectID() || stored[0].ReceiverID != pb.ObjectID() || !stored[0].Unread || stored[0].Message != "Line 1<br1>Line 2" {
		t.Errorf("stored mails %+v", stored)
	}

	e.stop()
	e = newBoardEnv(t, e, board, nil)
	sc := newSession("cid")
	e.enter(sc, 0)
	e.sync()
	got = sc.take(t)
	if _, ok := testclient.Find[testclient.ExMailArrived](got); !ok || !slices.Contains(sysMsgs(got), int32(bbsMsgGotMail)) {
		t.Errorf("Cid's login has no mail notice: %q %v", testclient.Names(got), sysMsgs(got))
	}
	sa, _ = e.clanLogin("alice")
	sb, _ = e.clanLogin("bob")
	sg, _ := e.clanLogin("gm")

	// Bob reads it: the inbox lists it unread, the view escapes it, then it
	// is read.
	inbox := boardPage(t, "inbox", e.bypass(sb, "_bbsmail"))
	mailID := strconv.Itoa(int(stored[0].ID))
	wantContains(t, "inbox", inbox, `<a action="bypass _bbsmail;view;`+mailID+`"><font color="LEVEL">Hello <b></font></a>`, "2026-11-19 13:00", "Inbox&nbsp;(1)")
	view := boardPage(t, "view", e.bypass(sb, "_bbsmail;view;"+mailID))
	wantContains(t, "view", view, "<td width=\"285\">Alice</td>", "<td>bob;Cid</td>", "Hello &lt;b&gt;", "Line 1&lt;br1&gt;Line 2", "_bbsmail;reply;"+mailID)
	inbox = boardPage(t, "inbox", e.bypass(sb, "_bbsmail;inbox;1"))
	if strings.Contains(inbox, "LEVEL") {
		t.Error("the mail is still unread")
	}
	reply := boardPage(t, "reply", e.bypass(sb, "_bbsmail;reply;"+mailID))
	wantContains(t, "reply", reply, `list="Alice"`, `<a action="bypass _bbsmail;view;`+mailID+`">Hello <b></a>`)
	// The sender's own copy replies to the same recipients.
	sentID := strconv.Itoa(int(stored[2].ID))
	wantContains(t, "own reply", boardPage(t, "own reply", e.bypass(sa, "_bbsmail;reply;"+sentID)), `list="bob;Cid"`)
	// Someone else's mail is not found: the last list again.
	wantContains(t, "foreign mail", boardPage(t, "foreign", e.bypass(sa, "_bbsmail;view;"+mailID)), "Inbox&nbsp;(0)")

	// Search by title and by writer.
	found := boardPage(t, "search", e.boardWrite(sb, "Mail", "Search;inbox", "0", "SearchTarget", "Title", "hello"))
	wantContains(t, "title search", found, "_bbsmail;view;"+mailID)
	none := boardPage(t, "search", e.boardWrite(sb, "Mail", "Search;inbox", "0", "SearchTarget", "Writer", "zed"))
	if strings.Contains(none, "_bbsmail;view;") {
		t.Error("a writer search for zed found the mail")
	}

	// Archive, then delete.
	archive := boardPage(t, "archive", e.bypass(sb, "_bbsmail;store;"+mailID))
	wantContains(t, "archive", archive, "Mail Archive&nbsp;(1)", "Inbox</a>&nbsp;(0)", "_bbsmail;view;"+mailID)
	boardPage(t, "after delete", e.bypass(sb, "_bbsmail;del;"+mailID))
	e.boardWrites()
	stored, _ = board.LoadBBSMails(context.Background())
	if len(stored) != 2 || stored[0].ReceiverID == pb.ObjectID() {
		t.Errorf("after Bob's delete: %+v", stored)
	}

	// The rules: unknown, self, GM, too many, blocked, refusing all.
	wantMsgs(t, "to nobody and self", e.boardWrite(sa, "Mail", "Send", "0", "Nobody;Alice", "x", "y"), int32(bbsMsgInvalidTarget), int32(bbsMsgInvalidTarget))
	wantMsgs(t, "to a GM", e.boardWrite(sa, "Mail", "Send", "0", "Gamemaster", "x", "y"), int32(bbsMsgNoMailToGM))
	wantMsgs(t, "to six", e.boardWrite(sa, "Mail", "Send", "0", "a;b;c;d;e;f", "x", "y"), int32(bbsMsgFiveRecipients))
	friendSend(e, sb, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAdd, "Alice"))
	wantMsgs(t, "to a blocker", e.boardWrite(sa, "Mail", "Send", "0", "Bob", "x", "y"), int32(bbsMsgBlockedYou))
	friendSend(e, sb, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRemove, "Alice"))
	friendSend(e, sb, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockRefuseAll, ""))
	wantMsgs(t, "to one refusing all", e.boardWrite(sa, "Mail", "Send", "0", "Bob", "x", "y"), int32(bbsMsgBlockingEverything))
	// A GM ignores those.
	wantMsgs(t, "a GM to six", e.boardWrite(sg, "Mail", "Send", "0", "Bob;Alice;Cid;x;y;z", "x", "y"),
		int32(bbsMsgInvalidTarget), int32(bbsMsgInvalidTarget), int32(bbsMsgInvalidTarget), int32(bbsMsgSentMail))

	// The daily limit: 10 sent in 24 hours.
	friendSend(e, sb, clientpackets.OpRequestBlock, blockBody(clientpackets.BlockAcceptAll, ""))
	for range 9 {
		e.boardWrite(sa, "Mail", "Send", "0", "Bob", "spam", "z")
	}
	wantMsgs(t, "the 11th mail", e.boardWrite(sa, "Mail", "Send", "0", "Bob", "x", "y"), int32(bbsMsgDailyLimit))
	e.advance(24 * time.Hour)
	wantMsgs(t, "a day later", e.boardWrite(sa, "Mail", "Send", "0", "Bob", "x", "y"), int32(bbsMsgSentMail))

	// A long list goes out in two parts.
	ps = e.bypass(sb, "_bbsmail")
	parts := only[testclient.ShowBoard](ps)
	if id, text := parts[1].Part(); id != "102" || text == "null" {
		t.Errorf("Bob's full inbox is one part (%s %d)", id, len(text))
	}
	wantContains(t, "two-part inbox", boardPage(t, "inbox", ps), "Inbox&nbsp;(11)", `<td><a action="bypass _bbsmail;INBOX;2"> 2 </a></td>`)
}

// The friends board: the lists with their selections, deleting friends and
// unblocking; the block list's links need the page that offered them.
func TestBoardFriends(t *testing.T) {
	e := newBoardEnv(t, nil, nil, nil)
	sa, pa := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)
	sb, pb := e.boardChar("bob", "Bob", storage.AccessLevelPlayer)
	cid := e.createClanChar(clanChar{account: "cid", name: "Cid", level: 20})
	befriend(t, e, sa, sb)
	e.onLoop(func() { pa.social.blocks = append(pa.social.blocks, socialEntry{id: cid.ObjectID, name: "Cid"}) })
	bobID := strconv.Itoa(int(pb.ObjectID()))

	page := boardPage(t, "friends", e.bypass(sa, "_friendlist"))
	wantContains(t, "friends", page, `<a action="bypass _friend;select;`+bobID+`">[Select]</a>&nbsp;Bob (on)<br1>`)
	page = boardPage(t, "selected", e.bypass(sa, "_friend;select;"+bobID))
	wantContains(t, "selected", page, `<a action="bypass _friend;deselect;`+bobID+`">[Deselect]</a>&nbsp;Bob (on)`)
	wantContains(t, "friend mail", boardPage(t, "friend mail", e.bypass(sa, "_friend;mail")), "Write Mail mail Bob Title Title Content")
	// Mobius's friend mail button goes to the mail board, which does not
	// take "mail": nothing happens.
	if ps := e.boardWrite(sa, "Mail", "mail", "Bob", "Title", "x", "y"); len(ps) != 0 {
		t.Errorf("the friend mail button answered %q", testclient.Names(ps))
	}
	wantContains(t, "confirm", boardPage(t, "confirm", e.bypass(sa, "_friend;delconfirm")), "bypass _friend;delall")

	sb.take(t)
	ps := e.bypass(sa, "_friend;del")
	wantMsgs(t, "Alice", ps, int32(sysMsgDeletedFromFriends))
	if fl, ok := testclient.Find[testclient.FriendList](ps); !ok || len(fl.Friends) != 0 {
		t.Errorf("Alice's FriendList %+v", fl)
	}
	if fl, ok := testclient.Find[testclient.FriendList](sb.take(t)); !ok || len(fl.Friends) != 0 {
		t.Errorf("Bob's FriendList %+v (Mobius forgets to update it)", fl)
	}
	if rels := relationsOf(t, e, pb.ObjectID()); len(rels) != 0 {
		t.Errorf("Bob's stored relations %+v", rels)
	}
	var bobFriends int
	e.onLoop(func() { bobFriends = len(pb.social.friends) })
	if bobFriends != 0 {
		t.Error("Bob still lists Alice")
	}

	// The block list: a link of the page works, one no page offered does not.
	cidID := strconv.Itoa(int(cid.ObjectID))
	if ps := e.bypass(sa, "_block;select;"+cidID); len(ps) != 0 {
		t.Errorf("_block without its page answered %q", testclient.Names(ps))
	}
	page = boardPage(t, "blocks", e.bypass(sa, "_friendblocklist"))
	wantContains(t, "blocks", page, `[Select]</a>&nbsp;Cid (off)<br1>`)
	page = boardPage(t, "block selected", e.bypass(sa, "_block;select;"+cidID))
	wantContains(t, "block selected", page, `[Deselect]</a>&nbsp;Cid (off)`)
	ps = e.bypass(sa, "_block;del")
	wantMsgs(t, "unblocking", ps, int32(sysMsgRemovedFromIgnore))
	if strings.Contains(boardPage(t, "blocks after", ps), "Cid") {
		t.Error("Cid is still blocked")
	}
}

// The board off: every request answers "offline", and no mail notice comes
// at login.
func TestBoardDisabled(t *testing.T) {
	e := newBoardEnv(t, nil, nil, func(gs *config.GameServer) { gs.CommunityBoard.Enabled = false })
	sa, _ := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)
	wantMsgs(t, "Alt+B", e.showBoard(sa), int32(bbsMsgOffline))
	wantMsgs(t, "a bypass", e.bypass(sa, "_bbsloc"), int32(bbsMsgOffline))
	wantMsgs(t, "a write", e.boardWrite(sa, "Mail", "Send"), int32(bbsMsgOffline))
}

// Karma keeps a player off the home page (CommunityKarmaDisabled), not off
// the other boards.
func TestBoardKarma(t *testing.T) {
	e := newBoardEnv(t, nil, nil, nil)
	sa, pa := e.boardChar("alice", "Alice", storage.AccessLevelPlayer)
	e.onLoop(func() { pa.char.Karma = 100 })
	ps := e.showBoard(sa)
	if _, ok := testclient.BoardPage(ps); ok {
		t.Error("a player with karma got the home page")
	}
	boardPage(t, "regions with karma", e.bypass(sa, "_bbsloc"))
}
