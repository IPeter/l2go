package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// Playing together (Plan 12, Milestone 9), all off by default:
//
//   - -party N: hunting bots form parties of N. Bot 0 leads bots 1 to N-1,
//     bot N leads the next ones, and so on; a leader invites its missing
//     members now and then, a member accepts only its leader's invitation
//     and attacks the monster its leader targets.
//   - -trade P: with probability P an action of a wandering bot is a trade
//     with the nearest player (it walks up first when it stands too far):
//     it puts its first tradeable item in and confirms; the partner bot
//     accepts the request and confirms after it.
//   - -friend P: with probability P an action of a wandering bot is a
//     friend message to an online friend, or, without one, a friend
//     invitation to a player in sight, which a bot accepts.
//   - -ping D: every D each bot sends RequestSkillCoolTime and times the
//     SkillCoolTime that answers ("ping"). The game server answers it on
//     the world loop at once, so its round trip is what a player feels of
//     the tick time; the report gives its percentiles at the number of bots
//     in the world (Plan 12 Phase 9: the tick-time report at 500 bots).

// socialConfig is the command line of the social behaviour.
type socialConfig struct {
	party  int
	trade  float64
	friend float64
	ping   time.Duration
}

// maxPartySize is the most members a party has (Mobius).
const maxPartySize = 9

// addFlags registers the flags of c on fs.
func (c *socialConfig) addFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.party, "party", 0, fmt.Sprintf("hunting bots form parties of this size, 2 to %d (0: no parties)", maxPartySize))
	fs.Float64Var(&c.trade, "trade", 0, "probability that an action of a wandering bot is a trade with the nearest player")
	fs.Float64Var(&c.friend, "friend", 0, "probability that an action of a wandering bot is a friend message (or a friend invitation)")
	fs.DurationVar(&c.ping, "ping", 0, "time between two pings of a bot, the world loop round trip of the report (0: no pings)")
}

// validate checks c for a run in mode.
func (c socialConfig) validate(mode string) error {
	switch {
	case c.party != 0 && (c.party < 2 || c.party > maxPartySize):
		return fmt.Errorf("-party must be 0 or between 2 and %d", maxPartySize)
	case c.party != 0 && mode != modeHunt:
		return fmt.Errorf("-party needs -mode %s", modeHunt)
	case c.trade < 0 || c.friend < 0 || c.trade+c.friend > 1:
		return errors.New("-trade and -friend are probabilities with a sum of at most 1")
	case (c.trade > 0 || c.friend > 0) && mode != modeWander:
		return fmt.Errorf("-trade and -friend need -mode %s", modeWander)
	case c.ping < 0:
		return errors.New("-ping must not be negative")
	}
	return nil
}

// enabled reports whether bots play together at all.
func (c socialConfig) enabled() bool { return c.party > 0 || c.trade > 0 || c.friend > 0 }

// describe prints the social settings of a run, nothing when they are off.
func (c socialConfig) describe(out io.Writer) {
	if c.party > 0 {
		fmt.Fprintf(out, "bots: hunting in parties of %d\n", c.party)
	}
	if c.trade > 0 || c.friend > 0 {
		fmt.Fprintf(out, "bots: trade with probability %g, friend messages with probability %g\n", c.trade, c.friend)
	}
	if c.ping > 0 {
		fmt.Fprintf(out, "bots: each bot pings the world loop every %v\n", c.ping)
	}
}

// Timing and ranges of the social behaviour.
const (
	// partyInviteGap is how long a leader waits after an invitation before
	// the next one: Mobius lets a player have one open request.
	partyInviteGap = 5 * time.Second
	// partyCheck is how often a leader with a full party checks it.
	partyCheck = 15 * time.Second
	// tradeRange is how near a partner must stand for a trade (Mobius
	// refuses one over 150 units), with a margin for the estimates.
	tradeRange = 120
)

// socialBot is a bot's social state.
type socialBot struct {
	ready bool
	// leader is the party leader's character name for a member, empty for
	// a leader and without parties; members are the names a leader invites.
	leader   string
	members  []string
	inviteAt time.Time
	lastPing time.Time
	// tradeSeen is when the bot first saw its open trade window; confirmed
	// is set once it confirmed that trade.
	tradeSeen time.Time
	confirmed bool
}

// partyGroup returns the leader of bot id's party of size bots, and the
// members that leader invites, of n bots in all.
func partyGroup(id, size, n int) (leader int, members []int) {
	leader = id - id%size
	for m := leader + 1; m < min(leader+size, n); m++ {
		members = append(members, m)
	}
	return leader, members
}

// charName is the character name of bot id.
func charName(cfg config, id int) string {
	_, char := botNames(cfg.prefix, id, cfg.n)
	return char + classSuffix(cfg.class)
}

// initSocial fills in the party names.
func (b *bot) initSocial() {
	s := &b.soc
	s.ready = true
	if b.cfg.social.party < 2 || b.cfg.mode != modeHunt {
		return
	}
	leader, members := partyGroup(b.id, b.cfg.social.party, b.cfg.n)
	if leader != b.id {
		s.leader = charName(b.cfg, leader)
		return
	}
	for _, m := range members {
		s.members = append(s.members, charName(b.cfg, m))
	}
}

// social runs before the bot's action: the ping when due, then, with the
// social flags, the answers to invitations and requests, the open trade,
// the party, and a trade or friend action of a wandering bot. done is set
// when it used up the action.
func (b *bot) social(ctx context.Context, gc *testclient.GameConn) (done bool, err error) {
	s := &b.soc
	if !s.ready {
		b.initSocial()
	}
	now := time.Now()
	if b.cfg.social.ping > 0 && now.Sub(s.lastPing) >= b.cfg.social.ping {
		s.lastPing = now
		if err := b.ping(ctx, gc); err != nil {
			return true, err
		}
	}
	if !b.cfg.social.enabled() {
		return false, nil
	}
	if err := b.answerRequests(ctx, gc); err != nil {
		return true, err
	}
	if busy, err := b.tradeWindow(ctx, gc, now); busy || err != nil {
		return true, err
	}
	if b.cfg.mode == modeHunt && b.cfg.social.party >= 2 {
		if err := b.keepParty(ctx, gc, now); err != nil {
			return true, err
		}
		b.assist(gc.World(), now)
		return false, nil
	}
	if b.cfg.mode != modeWander || b.keeper || b.cfg.social.trade+b.cfg.social.friend <= 0 {
		return false, nil
	}
	switch r := b.rng.Float64(); {
	case r < b.cfg.social.trade:
		return true, b.tradeNearest(ctx, gc)
	case r < b.cfg.social.trade+b.cfg.social.friend:
		return true, b.friendChat(ctx, gc)
	}
	return false, nil
}

// ping times the round trip of RequestSkillCoolTime ("ping").
func (b *bot) ping(ctx context.Context, gc *testclient.GameConn) error {
	t0 := time.Now()
	if err := gc.RequestSkillCoolTime(ctx); err != nil {
		return staged("ping", err)
	}
	if _, err := testclient.Await[testclient.SkillCoolTime](ctx, gc, nil); err != nil {
		return staged("ping", err)
	}
	b.stats.Observe("ping", time.Since(t0))
	return nil
}

// answerRequests answers what came since the last action: a member takes
// its leader's party invitation (others are declined), a trading bot takes
// trade requests and a chatting bot friend invitations.
func (b *bot) answerRequests(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	if inv, ok := w.TakePartyInvite(); ok {
		answer := testclient.PartyAnswerDeclined
		if b.soc.leader != "" && inv.Requestor == b.soc.leader {
			answer = testclient.PartyAnswerAccepted
			b.stats.Count("party join")
		}
		if err := gc.AnswerPartyInvite(ctx, answer); err != nil {
			return staged("party", err)
		}
	}
	if _, ok := w.TakeTradeRequest(); ok {
		if err := gc.AnswerTradeRequest(ctx, b.cfg.social.trade > 0 && !b.keeper); err != nil {
			return staged("trade", err)
		}
	}
	if _, ok := w.TakeFriendInvite(); ok {
		accept := b.cfg.social.friend > 0
		if err := gc.AnswerFriendInvite(ctx, accept); err != nil {
			return staged("friend", err)
		}
		if accept {
			b.stats.Count("friend added")
		}
	}
	return nil
}

// tradeWindow looks after an open trade window: the bot confirms once its
// partner did, and cancels a trade that stays open longer than -timeout.
// busy is set while the window is open: the bot stays where it is.
func (b *bot) tradeWindow(ctx context.Context, gc *testclient.GameConn, now time.Time) (busy bool, err error) {
	s := &b.soc
	t, open := gc.World().Trade()
	if !open {
		s.tradeSeen, s.confirmed = time.Time{}, false
		return false, nil
	}
	if s.tradeSeen.IsZero() {
		s.tradeSeen = now
	}
	switch {
	case t.PartnerConfirmed && !s.confirmed:
		s.confirmed = true
		if err := gc.ConfirmTrade(ctx); err != nil {
			return true, staged("trade", err)
		}
	case now.Sub(s.tradeSeen) > b.cfg.timeout:
		b.stats.Count("trade canceled")
		if _, err := gc.CancelTrade(ctx); err != nil {
			return true, staged("trade", err)
		}
	}
	return true, nil
}

// keepParty makes a leader invite the first member its party window lacks,
// one invitation per partyInviteGap.
func (b *bot) keepParty(ctx context.Context, gc *testclient.GameConn, now time.Time) error {
	s := &b.soc
	if len(s.members) == 0 || now.Before(s.inviteAt) {
		return nil
	}
	p, _ := gc.World().Party()
	for _, name := range s.members {
		if p.HasMember(name) {
			continue
		}
		s.inviteAt = now.Add(partyInviteGap)
		b.stats.Count("party invite")
		return staged("party", gc.RequestJoinParty(ctx, name, testclient.LootRandom))
	}
	s.inviteAt = now.Add(partyCheck)
	return nil
}

// assist makes a member that fights nothing hunt its leader's target.
func (b *bot) assist(w *testclient.World, now time.Time) {
	if b.soc.leader == "" || b.target != 0 {
		return
	}
	leader, ok := w.FindPlayer(b.soc.leader)
	if !ok {
		return
	}
	if o, ok := assistTarget(leader, w.Npcs()); ok {
		b.target, b.attacking, b.fightStart = o.ObjectID, 0, now
	}
}

// assistTarget returns the living attackable monster of objs the leader
// targets.
func assistTarget(leader testclient.Object, objs []testclient.Object) (testclient.Object, bool) {
	if leader.TargetID == 0 {
		return testclient.Object{}, false
	}
	for _, o := range objs {
		if o.ObjectID == leader.TargetID && o.Kind == testclient.KindNpc && o.Attackable && !o.Dead {
			return o, true
		}
	}
	return testclient.Object{}, false
}

// tradePartner returns the player of objs nearest to me within maxDist that
// can trade (not in a store), and its distance. Ties go to the lower object
// ID.
func tradePartner(me testclient.Position, objs []testclient.Object, maxDist float64) (testclient.Object, float64, bool) {
	var (
		best     testclient.Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		if o.Kind != testclient.KindPlayer || o.Dead {
			continue
		}
		if ci, ok := o.Info.(testclient.CharInfoPacket); ok && ci.PrivateStoreType != 0 {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if d > maxDist {
			continue
		}
		if !found || d < bestDist || (d == bestDist && o.ObjectID < best.ObjectID) {
			best, bestDist, found = o, d, true
		}
	}
	return best, bestDist, found
}

// tradeOffer returns the item a bot puts into a trade: the first one of the
// trade window with a count, one of it.
func tradeOffer(items []testclient.TradeItem) (testclient.TradeOffer, bool) {
	for _, it := range items {
		if it.Count > 0 {
			return testclient.TradeOffer{ObjectID: it.ObjectID, Count: 1}, true
		}
	}
	return testclient.TradeOffer{}, false
}

// tradeNearest is a trade with the nearest player within -talk-range
// ("trade": the request to the TradeDone, the partner's answers included).
// A partner too far away is walked up to first; without one the bot walks.
func (b *bot) tradeNearest(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	partner, d, ok := tradePartner(w.Me().Pos, w.Players(), b.cfg.talkRange)
	if !ok {
		return b.walk(ctx, gc)
	}
	if d > tradeRange {
		return b.walkTo(ctx, gc, partner.Pos)
	}
	t0 := time.Now()
	if err := gc.RequestTrade(ctx, partner.ObjectID); err != nil {
		return staged("trade", err)
	}
	start, err := testclient.Await(ctx, gc, func(s testclient.TradeStart) bool { return s.PartnerID == partner.ObjectID })
	if err != nil {
		b.stats.Count("trade refused")
		return staged("trade", err)
	}
	if offer, ok := tradeOffer(start.Items); ok {
		if _, err := gc.AddTradeItem(ctx, offer.ObjectID, offer.Count); err != nil {
			return staged("trade", err)
		}
	}
	b.soc.confirmed = true
	if err := gc.ConfirmTrade(ctx); err != nil {
		return staged("trade", err)
	}
	done, err := testclient.Await[testclient.TradeDone](ctx, gc, nil)
	if err != nil {
		return staged("trade", err)
	}
	b.stats.Observe("trade", time.Since(t0))
	if done.Result == testclient.TradeDoneFinished {
		b.stats.Count("trade")
	} else {
		b.stats.Count("trade canceled")
	}
	return nil
}

// walkTo walks the bot to a point next to dest ("move").
func (b *bot) walkTo(ctx context.Context, gc *testclient.GameConn, dest testclient.Position) error {
	now := time.Now()
	dest.X += 30
	me := gc.World().Me().ObjectID
	if err := gc.SendMove(ctx, testclient.MoveRequest{Target: dest, Origin: b.position(now)}); err != nil {
		return staged("move", err)
	}
	m, err := testclient.Await(ctx, gc, func(m testclient.MoveToLocation) bool { return m.ObjectID == me })
	if err != nil {
		return staged("move", err)
	}
	b.stats.Observe("move", time.Since(now))
	b.from, b.to, b.departed = m.Origin(), m.Destination(), time.Now()
	return nil
}

// friendChat sends a phrase to a random online friend ("friend message"),
// or, without one, invites a player in sight who is no friend yet ("friend
// invite"). Without anyone to invite the bot walks.
func (b *bot) friendChat(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	friends := w.Friends()
	if f, ok := pickOnlineFriend(friends, b.rng); ok {
		if err := gc.SendFriendMessage(ctx, f.Name, phrases[b.rng.IntN(len(phrases))]); err != nil {
			return staged("friend", err)
		}
		b.stats.Count("friend message")
		return nil
	}
	o, ok := friendCandidate(w.Me().Pos, w.Players(), friends, b.cfg.talkRange)
	if !ok {
		return b.walk(ctx, gc)
	}
	if err := gc.InviteFriend(ctx, o.Name); err != nil {
		return staged("friend", err)
	}
	b.stats.Count("friend invite")
	return nil
}

// pickOnlineFriend returns a random online friend.
func pickOnlineFriend(friends []testclient.Friend, rng *rand.Rand) (testclient.Friend, bool) {
	var online []testclient.Friend
	for _, f := range friends {
		if f.Online {
			online = append(online, f)
		}
	}
	if len(online) == 0 {
		return testclient.Friend{}, false
	}
	return online[rng.IntN(len(online))], true
}

// friendCandidate returns the player of objs nearest to me within maxDist
// who is not among friends. Ties go to the lower object ID.
func friendCandidate(me testclient.Position, objs []testclient.Object, friends []testclient.Friend, maxDist float64) (testclient.Object, bool) {
	known := make(map[string]bool, len(friends))
	for _, f := range friends {
		known[f.Name] = true
	}
	var (
		best     testclient.Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		if o.Kind != testclient.KindPlayer || known[o.Name] {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if d > maxDist {
			continue
		}
		if !found || d < bestDist || (d == bestDist && o.ObjectID < best.ObjectID) {
			best, bestDist, found = o, d, true
		}
	}
	return best, found
}

// writeTickLine adds the world loop line to a report that has pings: the
// round trip percentiles at the number of bots in the world.
func writeTickLine(b *strings.Builder, r Report) {
	for _, l := range r.Latency {
		if l.Op != "ping" {
			continue
		}
		fmt.Fprintf(b, "world loop round trip at %d bots in the world: p50 %s, p90 %s, p99 %s, max %s (%d pings)\n",
			r.States[stateInWorld], fmtDur(l.P50), fmtDur(l.P90), fmtDur(l.P99), fmtDur(l.Max), l.N)
		return
	}
}
