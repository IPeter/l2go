package e2e

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The playing-together part of Plan 12's demo checklist (Milestone 9),
// walked by two test clients: a store purchase, a trade, a party that hunts
// a monster and shares its exp, a clan founded at a village master with the
// second player taken in, and the friend notices. TestSocial runs it against
// the in-process stack, TestSmokeSocial (build tag smoke) against running
// servers, where the steps that need GM rights or adena skip.

// Items, NPCs and places of the social flow, from the Mobius data.
const (
	npcGremlin = 20001 // a level 1 monster (29 exp, 2 SP)
	npcBitz    = 30026 // the Talking Island village master

	// itemStem is Stem, a stackable, tradeable material. The starting
	// Dagger cannot be traded (Mobius is_tradable false); the Tutorial Guide
	// can, so the trade without a GM setup moves it.
	itemStem = 1864

	// socialAdena and socialStems are what the in-process store gives each
	// player.
	socialAdena = 1000
	socialStems = 20
	// A's sell store offers storeStems Stems at storePrice each, and B buys
	// them all. In the trade B gives A tradeStems Stems for tradeAdena.
	storeStems = 5
	storePrice = 10
	tradeStems = 3
	tradeAdena = 100

	// socialClan is the clan A founds.
	socialClan = "Hunters"
	// clanLevel is the level a clan founder needs (Mobius ClanTable).
	clanLevel = 10

	// floodPause outlasts the 1 s flood guard of the transactions (a
	// purchase, a trade confirmation).
	floodPause = 2 * time.Second
	// fightTimeout bounds a level 1 fight against a Gremlin.
	fightTimeout = time.Minute
	// sitAnimation outlasts the animation of sitting down (2.5 s in
	// Mobius), during which the character cannot stand up.
	sitAnimation = 3 * time.Second
)

var (
	// fieldSpot is open ground on the road south-west of the Human Fighter
	// start, 400 units from the nearest guard: far enough from NPCs for a
	// store (stores.minRangeFromNpc, 100).
	fieldSpot = roadSpot
	// bitzSpot is next to the village master Bitz at (-83326, 242964, -3720).
	bitzSpot = testclient.Position{X: -83290, Y: 242990, Z: -3720}
)

// socialFlow is one walk through the social checklist.
type socialFlow struct {
	loginAddr string
	// gameAddr is the game server; empty means the address the server list
	// announces.
	gameAddr string
	password string
	// accounts and names are A's and B's accounts (they must not exist yet)
	// and character names (they must be free).
	accounts, names [2]string
	// htmlDir is the folder of the NPC dialog pages; the clan step skips
	// without it.
	htmlDir string
	// prepare makes the stored character of account a GM with socialAdena
	// adena and socialStems Stems, while it is not in the world. Only the in-process
	// stack has it; without it the steps that need GM rights or adena skip.
	prepare func(t *testing.T, account string)
	// passTime lets d of game time pass at once; nil waits in real time.
	passTime func(t *testing.T, d time.Duration)
}

// socialWalk is the state the steps of one social flow share.
type socialWalk struct {
	f   socialFlow
	top *testing.T
	ctx context.Context
	// a and b are the players' game connections; aID and bID their
	// characters.
	a, b     *testclient.GameConn
	aID, bID int32
}

// run walks the flow. Every step is a subtest, and a failed step stops the
// flow.
func (f socialFlow) run(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), flowTimeout)
	defer cancel()
	w := &socialWalk{f: f, top: t, ctx: ctx}
	w.step(t, "A and B enter the world", w.enter)
	w.step(t, "M9.7 store purchase", w.storePurchase)
	w.step(t, "M9.6 trade", w.trade)
	w.step(t, "M9.1 M9.2 party hunt with shared exp", w.partyHunt)
	w.step(t, "M9.10 clan create and invite", w.clan)
	w.step(t, "M9.4 friends and their notices", w.friends)
}

func (w *socialWalk) step(t *testing.T, name string, fn func(t *testing.T)) {
	t.Helper()
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

// gm reports whether the flow can use GM rights and adena.
func (w *socialWalk) gm() bool { return w.f.prepare != nil }

// needGM skips a step that needs GM rights or adena.
func (w *socialWalk) needGM(t *testing.T) {
	t.Helper()
	if !w.gm() {
		t.Skip("needs GM rights and adena, which only the in-process stack (TestSocial) can arrange in its store")
	}
}

// wait lets d of game time pass.
func (w *socialWalk) wait(t *testing.T, d time.Duration) {
	t.Helper()
	if w.f.passTime != nil {
		w.f.passTime(t, d)
		return
	}
	select {
	case <-time.After(d):
	case <-w.ctx.Done():
		t.Fatalf("the flow timed out while waiting %v", d)
	}
}

// stepCtx is the context of a step: the flow's, cut to stepTimeout plus
// extra.
func (w *socialWalk) stepCtx(t *testing.T, extra time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(w.ctx, stepTimeout+extra)
	t.Cleanup(cancel)
	return ctx
}

// gameAddress returns where the game server listens: the flow's address,
// or the one the server list announces.
func (w *socialWalk) gameAddress(t *testing.T, ctx context.Context, account string) string {
	t.Helper()
	if w.f.gameAddr != "" {
		return w.f.gameAddr
	}
	c := loggedIn(t, ctx, w.f.loginAddr, account, w.f.password)
	var addr string
	poll(t, ctx, "game server 1 to be up in the server list", func() (bool, error) {
		list, err := c.ServerList(ctx)
		if err != nil {
			return false, err
		}
		s, ok := findServer(list, serverID)
		if ok && s.Up {
			addr = net.JoinHostPort(s.IP.String(), strconv.Itoa(int(s.Port)))
		}
		return ok && s.Up, nil
	})
	w.f.gameAddr = addr
	return addr
}

// login logs account in on the login server and the game server and
// returns the game connection at the character list. The connection stays
// open until the flow ends.
func (w *socialWalk) login(t *testing.T, ctx context.Context, account string) (*testclient.GameConn, testclient.CharacterList) {
	t.Helper()
	addr := w.gameAddress(t, ctx, account)
	keys, err := testclient.LoginAndSelect(ctx, w.f.loginAddr, account, w.f.password, serverID)
	if err != nil {
		t.Fatalf("logging in as %s: %v", account, err)
	}
	c, err := testclient.DialGame(ctx, addr, testclient.InterludeProtocol)
	if c != nil {
		w.top.Cleanup(func() { _ = c.Close() })
	}
	if err != nil {
		t.Fatalf("DialGame(%s): %v", addr, err)
	}
	list, err := c.AuthLogin(ctx, account, keys)
	if err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
	return c, list
}

// enterAs logs account in, creates its Human Fighter name when the account
// has none (and prepares it), and enters the world with it.
func (w *socialWalk) enterAs(t *testing.T, ctx context.Context, account, name string) (*testclient.GameConn, testclient.EnterWorldResult) {
	t.Helper()
	c, list := w.login(t, ctx, account)
	if _, ok := list.Find(name); !ok {
		res, err := c.CreateCharacter(ctx, humanFighter(name))
		if err != nil || !res.Created {
			t.Fatalf("CreateCharacter %s: %+v, %v", name, res, err)
		}
		if w.f.prepare != nil {
			w.f.prepare(t, strings.ToLower(account))
		}
	}
	if _, err := c.SelectCharacter(ctx, 0); err != nil {
		t.Fatalf("SelectCharacter %s: %v", name, err)
	}
	res, err := c.EnterWorld(ctx)
	if err != nil {
		t.Fatalf("EnterWorld %s: %v", name, err)
	}
	return c, res
}

// enter brings A and B into the world. With GM rights both go to
// fieldSpot, a few steps apart; otherwise they stay at the Human Fighter
// start, where B walks up to A.
func (w *socialWalk) enter(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	var res testclient.EnterWorldResult
	w.a, res = w.enterAs(t, ctx, w.f.accounts[0], w.f.names[0])
	w.aID = res.UserInfo.ObjectID
	w.b, res = w.enterAs(t, ctx, w.f.accounts[1], w.f.names[1])
	w.bID = res.UserInfo.ObjectID
	if w.gm() {
		teleport(t, ctx, w.a, fieldSpot)
		teleport(t, ctx, w.b, testclient.Position{X: fieldSpot.X + 60, Y: fieldSpot.Y, Z: fieldSpot.Z})
	} else {
		w.walkUp(t, ctx)
	}
	w.seeEachOther(t, ctx)
}

// walkUp walks B next to A and waits until it arrives.
func (w *socialWalk) walkUp(t *testing.T, ctx context.Context) {
	t.Helper()
	from, to := w.b.World().Me().Pos, w.a.World().Me().Pos
	to.X += 40
	if from.DistanceTo(to) < 20 {
		return
	}
	if _, err := w.b.MoveTo(ctx, to.X, to.Y, to.Z); err != nil {
		t.Fatalf("B walks to A: %v", err)
	}
	w.wait(t, walkTime(from, to))
}

// seeEachOther waits until A and B see each other.
func (w *socialWalk) seeEachOther(t *testing.T, ctx context.Context) {
	t.Helper()
	poll(t, ctx, "A and B to see each other", func() (bool, error) {
		_, aSees := w.a.World().FindPlayer(w.f.names[1])
		_, bSees := w.b.World().FindPlayer(w.f.names[0])
		return aSees && bSees, nil
	})
}

// awaitCount waits until c's inventory holds want of itemID.
func awaitCount(t *testing.T, ctx context.Context, who string, c *testclient.GameConn, itemID int32, want int64) {
	t.Helper()
	poll(t, ctx, fmt.Sprintf("%s to have %d of item %d (has %d)", who, want, itemID, c.World().ItemCount(itemID)), func() (bool, error) {
		return c.World().ItemCount(itemID) == want, nil
	})
}

// findItem returns c's item of itemID.
func findItem(t *testing.T, who string, c *testclient.GameConn, itemID int32) testclient.Item {
	t.Helper()
	it, ok := c.World().FindItem(itemID)
	if !ok {
		t.Fatalf("%s has no item %d: %+v", who, itemID, c.World().Items())
	}
	return it
}

// storePurchase: A opens a sell store of Stems with a message, B sees the
// message, walks up and buys them all; the adena moves and the sold out
// store closes (checklist item 7).
func (w *socialWalk) storePurchase(t *testing.T) {
	w.needGM(t)
	ctx := w.stepCtx(t, 0)
	a, b := w.a, w.b
	setup, err := a.OpenSellStoreSetup(ctx)
	if err != nil {
		t.Fatalf("OpenSellStoreSetup: %v", err)
	}
	var stems testclient.StoreSellItem
	if i := slices.IndexFunc(setup.Available, func(it testclient.StoreSellItem) bool { return it.ItemID == itemStem }); i >= 0 {
		stems = setup.Available[i]
	} else {
		t.Fatalf("the store setup does not offer the Stems: %+v", setup.Available)
	}
	const message = "Stems, cheap"
	if err := a.SetSellStoreMessage(ctx, message); err != nil {
		t.Fatalf("SetSellStoreMessage: %v", err)
	}
	if _, err := a.OpenSellStore(ctx, false, []testclient.StoreSellOffer{{ObjectID: stems.ObjectID, Count: storeStems, Price: storePrice}}); err != nil {
		t.Fatalf("OpenSellStore: %v", err)
	}
	if m := await(t, ctx, b, func(m testclient.PrivateStoreMsgSell) bool { return m.ObjectID == w.aID }); m.Message != message {
		t.Errorf("B sees the store message %q, want %q", m.Message, message)
	}

	if _, err := b.Target(ctx, w.aID); err != nil {
		t.Fatalf("B targets A: %v", err)
	}
	l, err := b.ViewSellStore(ctx, w.aID)
	if err != nil {
		t.Fatalf("ViewSellStore: %v", err)
	}
	it, ok := l.Find(stems.ObjectID)
	if !ok || it.Price != storePrice || it.Count != storeStems || l.Adena != socialAdena {
		t.Fatalf("store window %+v, want %d Stems at %d and B's %d adena", l, storeStems, storePrice, socialAdena)
	}
	m, err := b.Purchase(ctx, w.aID, []testclient.StorePurchase{{ObjectID: stems.ObjectID, Count: storeStems, Price: storePrice}})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if m.ID != testclient.SysMsgYouPurchasedCount {
		t.Errorf("purchase message %d, want %d", m.ID, testclient.SysMsgYouPurchasedCount)
	}
	const total = storeStems * storePrice
	awaitCount(t, ctx, "A", a, itemAdena, socialAdena+total)
	awaitCount(t, ctx, "A", a, itemStem, socialStems-storeStems)
	awaitCount(t, ctx, "B", b, itemAdena, socialAdena-total)
	awaitCount(t, ctx, "B", b, itemStem, socialStems+storeStems)
	poll(t, ctx, "A's sold-out store to close", func() (bool, error) {
		return a.World().Me().UserInfo.PrivateStoreType == 0, nil
	})
	// The store sat A down; it stands for the steps after, once the sitting
	// animation is over (Mobius ignores the request until then).
	if a.World().Me().Sitting {
		w.wait(t, sitAnimation)
		if err := a.RequestActionUse(ctx, testclient.ActionSitStand, false, false); err != nil {
			t.Fatalf("RequestActionUse(sit/stand): %v", err)
		}
		poll(t, ctx, "A to stand up", func() (bool, error) { return !a.World().Me().Sitting, nil })
	}
	w.wait(t, floodPause)
}

// trade: B gives A Stems for A's adena (with the GM setup; otherwise its
// Tutorial Guide for nothing); both confirm and the items move (checklist
// item 6).
func (w *socialWalk) trade(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a, b := w.a, w.b
	itemID, count := int32(itemTutorialGuide), int64(1)
	if w.gm() {
		itemID, count = itemStem, tradeStems
	}
	aItems, bItems := a.World().ItemCount(itemID), b.World().ItemCount(itemID)
	aAdena, bAdena := a.World().ItemCount(itemAdena), b.World().ItemCount(itemAdena)
	give := []testclient.TradeOffer{{ObjectID: findItem(t, "B", b, itemID).ObjectID, Count: int32(count)}}
	var take []testclient.TradeOffer
	paid := int64(0)
	if w.gm() {
		take = []testclient.TradeOffer{{ObjectID: findItem(t, "A", a, itemAdena).ObjectID, Count: tradeAdena}}
		paid = tradeAdena
	}
	res, err := testclient.Trade(ctx, b, a, give, take)
	if err != nil {
		t.Fatalf("Trade: %v", err)
	}
	if res.Requester.Result != testclient.TradeDoneFinished || res.Partner.Result != testclient.TradeDoneFinished {
		t.Fatalf("TradeDone %+v, want finished on both sides", res)
	}
	awaitCount(t, ctx, "A", a, itemID, aItems+count)
	awaitCount(t, ctx, "B", b, itemID, bItems-count)
	awaitCount(t, ctx, "A", a, itemAdena, aAdena-paid)
	awaitCount(t, ctx, "B", b, itemAdena, bAdena+paid)
	for who, c := range map[string]*testclient.GameConn{"A": a, "B": b} {
		if !slices.ContainsFunc(c.World().SystemMessages(), func(m testclient.SystemMessage) bool { return m.ID == testclient.SysMsgTradeSuccessful }) {
			t.Errorf("%s did not get %q", who, "Your trade is successful.")
		}
		if _, open := c.World().Trade(); open {
			t.Errorf("%s's trade window is still open", who)
		}
	}
	w.wait(t, floodPause)
}

// partyHunt: A invites B with loot "Random", B accepts and both see the
// party window. A spawns a Gremlin (GM) and kills it: both get the same
// share of its exp. A leaves, and the party of two disperses (checklist
// items 1-3).
func (w *socialWalk) partyHunt(t *testing.T) {
	ctx := w.stepCtx(t, fightTimeout)
	a, b := w.a, w.b
	party, err := testclient.FormParty(ctx, a, testclient.LootRandom, b)
	if err != nil {
		t.Fatalf("FormParty: %v", err)
	}
	if party.LeaderID != w.aID || party.Loot != testclient.LootRandom || len(party.Members) != 1 {
		t.Errorf("A's party window %+v", party)
	}
	poll(t, ctx, "B's party window to show A leading", func() (bool, error) {
		p, ok := b.World().Party()
		return ok && p.LeaderID == w.aID && p.HasMember(w.f.names[0]), nil
	})

	if w.gm() {
		w.huntGremlin(t, ctx)
	} else {
		t.Log("the hunt needs a GM to spawn the monster; the party only")
	}

	// A leaves: a party of two disperses. The leader goes first and gets
	// PartySmallWindowDeleteAll; B, left alone, gets only A's
	// PartySmallWindowDelete (Mobius Party.removePartyMember).
	if _, err := a.LeaveParty(ctx); err != nil {
		t.Fatalf("A leaves: %v", err)
	}
	await(t, ctx, b, func(d testclient.PartySmallWindowDelete) bool { return d.ObjectID == w.aID })
	poll(t, ctx, "both party windows to close", func() (bool, error) {
		_, aIn := a.World().Party()
		_, bIn := b.World().Party()
		return !aIn && !bIn, nil
	})
}

// huntGremlin: A spawns a Gremlin where it stands and kills it; A and B,
// level 1 both, get the same exp.
func (w *socialWalk) huntGremlin(t *testing.T, ctx context.Context) {
	t.Helper()
	a, b := w.a, w.b
	if err := a.CancelTarget(ctx); err != nil {
		t.Fatalf("CancelTarget: %v", err)
	}
	if err := a.GMCommand(ctx, "spawn "+strconv.Itoa(npcGremlin)); err != nil {
		t.Fatalf("GMCommand(spawn): %v", err)
	}
	var gremlin testclient.Object
	poll(t, ctx, "the spawned Gremlin", func() (bool, error) {
		me := a.World().Me().Pos
		for _, o := range a.World().Npcs() {
			if o.NpcID == npcGremlin && !o.Dead && o.Pos.DistanceTo(me) < 200 {
				gremlin = o
				return true, nil
			}
		}
		return false, nil
	})
	if _, err := a.Attack(ctx, gremlin.ObjectID); err != nil {
		t.Fatalf("Attack: %v", err)
	}
	if _, err := a.FightUntilDead(ctx, gremlin.ObjectID); err != nil {
		t.Fatalf("FightUntilDead: %v", err)
	}
	expOf := func(who string, c *testclient.GameConn) int64 {
		t.Helper()
		var exp int64
		poll(t, ctx, who+"'s exp message", func() (bool, error) {
			for _, m := range c.World().SystemMessages() {
				if m.ID == testclient.SysMsgEarnedExpAndSP && len(m.Params) > 0 {
					exp = m.Params[0].Number
					return true, nil
				}
			}
			return false, nil
		})
		return exp
	}
	aExp, bExp := expOf("A", a), expOf("B", b)
	// Two level 1 members share alike: round(29 × 1.1 / 2) with the default
	// rates, the same for both.
	if aExp <= 0 || aExp != bExp {
		t.Errorf("exp: A %d, B %d; want the same share for both", aExp, bExp)
	}
	t.Logf("each member earned %d exp", aExp)
}

// clan: A reaches level 10 (GM), founds a clan at the village master Bitz
// and takes B in; both clan windows show both (checklist item 10).
func (w *socialWalk) clan(t *testing.T) {
	w.needGM(t)
	if fi, err := os.Stat(filepath.Join(w.f.htmlDir, "villagemaster")); err != nil || !fi.IsDir() {
		t.Skipf("no village master pages in %q: make data copies them from the Mobius checkout", w.f.htmlDir)
	}
	ctx := w.stepCtx(t, 0)
	a, b := w.a, w.b
	if _, err := a.LevelTo(ctx, clanLevel); err != nil {
		t.Fatalf("LevelTo: %v", err)
	}
	teleport(t, ctx, a, bitzSpot)
	teleport(t, ctx, b, testclient.Position{X: bitzSpot.X + 40, Y: bitzSpot.Y, Z: bitzSpot.Z})
	bitz := findNpc(t, ctx, a, npcBitz)
	all, err := a.CreateClan(ctx, bitz.ObjectID, socialClan)
	if err != nil {
		t.Fatalf("CreateClan: %v", err)
	}
	if all.LeaderName != w.f.names[0] || len(all.Members) != 1 || all.Members[0].ObjectID != w.aID {
		t.Errorf("the new clan window %+v", all)
	}
	join, err := testclient.JoinClan(ctx, a, b, testclient.PledgeMain)
	if err != nil {
		t.Fatalf("JoinClan: %v", err)
	}
	if join.ClanID != all.ClanID {
		t.Errorf("B joined clan %d, want %d", join.ClanID, all.ClanID)
	}
	if add := await(t, ctx, a, func(p testclient.PledgeShowMemberListAdd) bool { return p.Name == w.f.names[1] }); add.ObjectID != w.bID {
		t.Errorf("A's clan window adds %+v", add)
	}
	list, err := b.ClanMembers(ctx)
	if err != nil {
		t.Fatalf("ClanMembers: %v", err)
	}
	if list.ClanName != socialClan {
		t.Errorf("B's clan window shows clan %q", list.ClanName)
	}
	for _, name := range w.f.names {
		if _, ok := list.Member(name); !ok {
			t.Errorf("B's clan window lacks %s: %+v", name, list.Members)
		}
	}
}

// friends: A invites B into its friend list, B accepts; a friend message
// arrives; B logs out and in again and A gets both notices (checklist item
// 4; TestFriendsSeeEachOther checks the lists in more detail).
func (w *socialWalk) friends(t *testing.T) {
	ctx := w.stepCtx(t, 0)
	a, b := w.a, w.b
	aName, bName := w.f.names[0], w.f.names[1]
	if err := a.InviteFriend(ctx, bName); err != nil {
		t.Fatalf("InviteFriend: %v", err)
	}
	if req, err := b.AwaitFriendInvite(ctx); err != nil || req.Requestor != aName {
		t.Fatalf("B's invitation %+v, %v", req, err)
	}
	if _, err := b.AcceptFriendInvite(ctx); err != nil {
		t.Fatalf("AcceptFriendInvite: %v", err)
	}
	poll(t, ctx, "A's friend list to show B online", func() (bool, error) {
		return slices.ContainsFunc(a.World().Friends(), func(f testclient.Friend) bool { return f.Name == bName && f.Online }), nil
	})
	if err := a.SendFriendMessage(ctx, bName, "good hunt"); err != nil {
		t.Fatalf("SendFriendMessage: %v", err)
	}
	if m := await[testclient.L2FriendSay](t, ctx, b, nil); m.Sender != aName || m.Message != "good hunt" {
		t.Errorf("B's friend message %+v", m)
	}

	if err := b.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if st := await[testclient.FriendStatus](t, ctx, a, nil); st.Online || st.Name != bName {
		t.Errorf("A's FriendStatus at B's logout %+v", st)
	}
	w.b, _ = w.enterAs(t, ctx, w.f.accounts[1], bName)
	if m := await(t, ctx, a, func(m testclient.SystemMessage) bool { return m.ID == testclient.SysMsgFriendLoggedIn }); !slices.Equal(m.Texts(), []string{bName}) {
		t.Errorf("A's log-in notice %+v", m)
	}
	poll(t, ctx, "A's friend list to show B online again", func() (bool, error) {
		return slices.ContainsFunc(a.World().Friends(), func(f testclient.Friend) bool { return f.Name == bName && f.Online }), nil
	})
}

// TestSocial runs the social flow against the in-process stack: both
// players are GMs with adena, and the game clock jumps over the waits.
func TestSocial(t *testing.T) {
	s := startStack(t)
	f := socialFlow{
		loginAddr: s.loginAddr,
		gameAddr:  s.gameAddr,
		password:  "Secret-1",
		accounts:  [2]string{"sociala", "socialb"},
		names:     [2]string{"Corvin", "Delia"},
		htmlDir:   filepath.Join("..", "..", "data", "html"),
		prepare: func(t *testing.T, account string) {
			s.makeGM(t, account)
			s.giveItems(t, account, itemAdena, socialAdena)
			s.giveItems(t, account, itemStem, socialStems)
		},
		passTime: s.passTime,
	}
	f.run(t)
	if n := s.server.HackAttempts(); n != 0 {
		t.Errorf("hack attempts: %d; the social flow should cause none", n)
	}
}
