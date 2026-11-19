package e2e

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

const (
	// posTolerance is how far apart two positions on the ground may be and
	// still be the same place; zTolerance is the same for the height (a
	// teleport lands 5 units higher, as in Mobius).
	posTolerance = 16
	zTolerance   = 32
	// runSpeed is the run speed UserInfo shows for a new Human Fighter. The
	// speed it walks at has the DEX bonus too (126.5), so times computed
	// with it err on the long side.
	runSpeed = 115
)

// step runs fn as the subtest name and stops the flow if it fails. A
// skipped step does not stop it.
func (w *walk) step(t *testing.T, name string, fn func(t *testing.T)) {
	t.Helper()
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

// stepCtx returns the context of a step: the flow's, cut to stepTimeout
// plus extra (the game time the step lets pass), and cancelled when the
// step ends.
func (w *walk) stepCtx(t *testing.T, extra time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(w.ctx, stepTimeout+extra)
	t.Cleanup(cancel)
	return ctx
}

// closeAtEnd closes c when the whole flow ends, not the step.
func (w *walk) closeAtEnd(c interface{ Close() error }) {
	w.top.Cleanup(func() { _ = c.Close() })
}

// let lets d of game time pass: the in-process stack moves its fake clock
// ahead at once, while against a live server the walk waits for real.
func (w *walk) let(t *testing.T, d time.Duration) {
	t.Helper()
	if w.f.passTime != nil {
		w.f.passTime(t, d)
		return
	}
	select {
	case <-time.After(d):
	case <-w.ctx.Done():
		t.Fatalf("the flow timed out while letting %v of game time pass", d)
	}
}

// needHTML skips the step when the NPC dialog pages are missing.
func (w *walk) needHTML(t *testing.T) {
	t.Helper()
	if fi, err := os.Stat(w.f.htmlDir); err != nil || !fi.IsDir() {
		t.Skipf("no NPC dialog pages in %q: they are not in git; make data copies them from the Mobius checkout", w.f.htmlDir)
	}
}

// needStore skips the step unless the walk reaches into the store: only the
// in-process stack (TestFullFlow) makes B a GM with adena.
func (w *walk) needStore(t *testing.T) {
	t.Helper()
	if w.f.makeGM == nil || w.f.giveItems == nil {
		t.Skip("needs B to be a GM with adena, which only the in-process stack (TestFullFlow) can arrange in its store")
	}
}

// login logs pl in the way the client does, on the login server and then on
// the game server, and returns the character list. The game connection stays
// open until the flow ends.
func (w *walk) login(t *testing.T, ctx context.Context, pl *player) testclient.CharacterList {
	t.Helper()
	keys, err := testclient.LoginAndSelect(ctx, w.f.loginAddr, pl.typed, w.f.password, serverID)
	if err != nil {
		t.Fatalf("logging in as %s: %v", pl.typed, err)
	}
	c, err := testclient.DialGame(ctx, w.gameAddr, testclient.InterludeProtocol)
	if c != nil {
		w.closeAtEnd(c)
	}
	if err != nil {
		t.Fatalf("DialGame(%s): %v", w.gameAddr, err)
	}
	list, err := c.AuthLogin(ctx, pl.typed, keys)
	if err != nil {
		t.Fatalf("AuthLogin: %v", err)
	}
	pl.conn, pl.keys = c, keys
	return list
}

// enter selects the character in slot 0 of pl's last character list and
// enters the world with it.
func enter(t *testing.T, ctx context.Context, pl *player) (testclient.CharSelected, testclient.EnterWorldResult) {
	t.Helper()
	sel, err := pl.conn.SelectCharacter(ctx, 0)
	if err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	res, err := pl.conn.EnterWorld(ctx)
	if err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}
	return sel, res
}

// await waits for the next packet of type T on c that match accepts (nil
// accepts any T). The packets before it are skipped; the world model still
// sees them.
func await[T testclient.Packet](t *testing.T, ctx context.Context, c *testclient.GameConn, match func(T) bool) T {
	t.Helper()
	v, err := testclient.Await(ctx, c, match)
	if err != nil {
		t.Fatalf("waiting for %s: %v", v.PacketName(), err)
	}
	return v
}

// until is await that also returns the packets that came before.
func until[T testclient.Packet](t *testing.T, ctx context.Context, c *testclient.GameConn, match func(T) bool) (T, []testclient.Packet) {
	t.Helper()
	var (
		got    T
		before []testclient.Packet
	)
	_, err := c.WaitFor(ctx, func(p testclient.Packet) bool {
		if v, ok := p.(T); ok && (match == nil || match(v)) {
			got = v
			return true
		}
		before = append(before, p)
		return false
	})
	if err != nil {
		t.Fatalf("waiting for %s: %v", got.PacketName(), err)
	}
	return got, before
}

// ping sends RequestSkillCoolTime and returns the packets that came before
// its answer, SkillCoolTime. The server handles a client's packets one at a
// time in order and sends a connection's packets in order, so whatever the
// requests before the ping made the server send at once came before.
func ping(t *testing.T, ctx context.Context, c *testclient.GameConn) []testclient.Packet {
	t.Helper()
	if err := c.RequestSkillCoolTime(ctx); err != nil {
		t.Fatalf("RequestSkillCoolTime: %v", err)
	}
	_, before := until[testclient.SkillCoolTime](t, ctx, c, nil)
	return before
}

// none reports an error for each packet of type T in ps that match accepts
// (nil accepts any T).
func none[T testclient.Packet](t *testing.T, ps []testclient.Packet, what string, match func(T) bool) {
	t.Helper()
	for _, p := range ps {
		if v, ok := p.(T); ok && (match == nil || match(v)) {
			t.Errorf("%s: the server sent %s %+v", what, v.PacketName(), v)
		}
	}
}

// waitClosed waits until the server closes c and returns the packets that
// came before.
func waitClosed(t *testing.T, ctx context.Context, c *testclient.GameConn) []testclient.Packet {
	t.Helper()
	var seen []testclient.Packet
	_, err := c.WaitFor(ctx, func(p testclient.Packet) bool {
		seen = append(seen, p)
		return false
	})
	if !errors.Is(err, testclient.ErrServerClosed) {
		t.Fatalf("waiting for the server to close the connection: %v", err)
	}
	return seen
}

// awaitSay waits on c for the chat message text on channel typ.
func awaitSay(t *testing.T, ctx context.Context, c *testclient.GameConn, typ testclient.ChatType, text string) testclient.CreatureSay {
	t.Helper()
	return await(t, ctx, c, func(m testclient.CreatureSay) bool { return m.Type == typ && m.Text == text })
}

// checkSay checks who a chat message comes from: the sender's object ID and
// the name the client shows.
func checkSay(t *testing.T, what string, m testclient.CreatureSay, from int32, name string) {
	t.Helper()
	if m.ObjectID != from || m.Name != name {
		t.Errorf("%s: from object %d named %q, want %d named %q", what, m.ObjectID, m.Name, from, name)
	}
}

// pageLink is a link of a dialog page.
type pageLink struct {
	// bypass is what the client sends for it (RequestBypassToServer): the
	// action after "bypass" and its "-h" flag.
	bypass string
	// text is what the link shows.
	text string
}

// pageLinks returns the bypass links of a dialog page, in order.
func pageLinks(html string) []pageLink {
	const attr = `action="bypass `
	var links []pageLink
	rest := html
	for {
		i := strings.Index(rest, attr)
		if i < 0 {
			return links
		}
		rest = rest[i+len(attr):]
		end := strings.IndexByte(rest, '"')
		if end < 0 {
			return links
		}
		l := pageLink{bypass: strings.TrimSpace(rest[:end])}
		if cmd, ok := strings.CutPrefix(l.bypass, "-h "); ok {
			l.bypass = strings.TrimSpace(cmd)
		}
		rest = rest[end+1:]
		if gt := strings.IndexByte(rest, '>'); gt >= 0 {
			if lt := strings.Index(rest[gt+1:], "</a>"); lt >= 0 {
				l.text = strings.TrimSpace(rest[gt+1 : gt+1+lt])
			}
		}
		links = append(links, l)
	}
}

// requireLink fails the test unless the page offers the link cmd, and
// returns cmd.
func requireLink(t *testing.T, page testclient.NpcHtmlMessage, cmd string) string {
	t.Helper()
	links := pageLinks(page.HTML)
	for _, l := range links {
		if l.bypass == cmd {
			return cmd
		}
	}
	t.Fatalf("the page does not offer %q; its links: %+v", cmd, links)
	return ""
}

// placeholderRe finds a placeholder the server left in a page, such as
// %objectId%.
var placeholderRe = regexp.MustCompile(`%[A-Za-z]+%`)

// checkPage checks that a dialog page belongs to the object objectID, shows
// each of want and has no placeholder left.
func checkPage(t *testing.T, page testclient.NpcHtmlMessage, objectID int32, want ...string) {
	t.Helper()
	if page.ObjectID != objectID {
		t.Errorf("the page belongs to object %d, want %d", page.ObjectID, objectID)
	}
	for _, s := range want {
		if !strings.Contains(page.HTML, s) {
			t.Errorf("the page does not show %q:\n%s", s, page.HTML)
		}
	}
	if p := placeholderRe.FindString(page.HTML); p != "" {
		t.Errorf("the page still has the placeholder %s:\n%s", p, page.HTML)
	}
}

// bypass sends each of cmds as a dialog link.
func bypass(t *testing.T, ctx context.Context, c *testclient.GameConn, cmds ...string) {
	t.Helper()
	for _, cmd := range cmds {
		if err := c.Bypass(ctx, cmd); err != nil {
			t.Fatalf("Bypass(%q): %v", cmd, err)
		}
	}
}

// follow sends the dialog link cmd and returns the page it opens.
func follow(t *testing.T, ctx context.Context, c *testclient.GameConn, cmd string) testclient.NpcHtmlMessage {
	t.Helper()
	bypass(t, ctx, c, cmd)
	return await[testclient.NpcHtmlMessage](t, ctx, c, nil)
}

// numberRe finds the numbers of a text.
var numberRe = regexp.MustCompile(`-?[0-9]+`)

// messageLocation returns the place a location message names: the first
// three numbers of its parameters, from number parameters (the /loc
// messages) or from a text such as "x, y, z" (Mobius CURRENT_LOCATION_S1,
// GM command answers).
func messageLocation(m testclient.SystemMessage) ([3]int64, bool) {
	var nums []int64
	for _, p := range m.Params {
		switch p.Type {
		case testclient.SMParamInt, testclient.SMParamLong:
			nums = append(nums, p.Number)
		case testclient.SMParamText:
			for _, s := range numberRe.FindAllString(p.Text, -1) {
				if n, err := strconv.ParseInt(s, 10, 64); err == nil {
					nums = append(nums, n)
				}
			}
		}
	}
	if len(nums) < 3 {
		return [3]int64{}, false
	}
	return [3]int64{nums[0], nums[1], nums[2]}, true
}

// awaitLocation waits on c for a system message naming a place within
// posTolerance of want on the ground. The height may differ: a teleport
// lands 5 units higher.
func awaitLocation(t *testing.T, ctx context.Context, c *testclient.GameConn, want testclient.Position) {
	t.Helper()
	var others [][3]int64
	_, err := c.WaitFor(ctx, func(p testclient.Packet) bool {
		m, ok := p.(testclient.SystemMessage)
		if !ok {
			return false
		}
		loc, ok := messageLocation(m)
		if !ok {
			return false
		}
		if math.Hypot(float64(loc[0]-int64(want.X)), float64(loc[1]-int64(want.Y))) <= posTolerance {
			return true
		}
		others = append(others, loc)
		return false
	})
	if err != nil {
		t.Fatalf("waiting for a location message naming %v: %v (other places named: %v)", want, err, others)
	}
}

// dist2D returns the distance between p and q on the ground.
func dist2D(p, q testclient.Position) float64 {
	return math.Hypot(float64(p.X)-float64(q.X), float64(p.Y)-float64(q.Y))
}

// near reports whether got is the place want: within posTolerance on the
// ground and zTolerance in height.
func near(got, want testclient.Position) bool {
	return dist2D(got, want) <= posTolerance && math.Abs(float64(got.Z)-float64(want.Z)) <= zTolerance
}

// checkPosition reports an error unless got is near want.
func checkPosition(t *testing.T, what string, got, want testclient.Position) {
	t.Helper()
	if !near(got, want) {
		t.Errorf("%s at %v, want %v", what, got, want)
	}
}

// moveTo walks pl to dest with a mouse click, from where the walk knows pl
// stands, and returns the server's MoveToLocation of that walk.
func moveTo(t *testing.T, ctx context.Context, pl *player, dest testclient.Position) testclient.MoveToLocation {
	t.Helper()
	if err := pl.conn.SendMove(ctx, testclient.MoveRequest{Target: dest, Origin: pl.pos}); err != nil {
		t.Fatalf("MoveToLocation: %v", err)
	}
	return awaitMove(t, ctx, pl.conn, pl.id, dest)
}

// awaitMove waits on c for a MoveToLocation of the object id that goes to
// dest. The server sends a walk's MoveToLocation again while the walk goes
// on and when it arrives, so one of an earlier walk may still be queued.
func awaitMove(t *testing.T, ctx context.Context, c *testclient.GameConn, id int32, dest testclient.Position) testclient.MoveToLocation {
	t.Helper()
	var others []testclient.Position
	p, err := c.WaitFor(ctx, func(p testclient.Packet) bool {
		m, ok := p.(testclient.MoveToLocation)
		if !ok || m.ObjectID != id {
			return false
		}
		if near(m.Destination(), dest) {
			return true
		}
		others = append(others, m.Destination())
		return false
	})
	if err != nil {
		t.Fatalf("waiting for a MoveToLocation of %d to %v: %v (other destinations: %v)", id, dest, err, others)
	}
	return p.(testclient.MoveToLocation)
}

// awaitText waits on c for a text system message (Mobius sendMessage) that
// contains part.
func awaitText(t *testing.T, ctx context.Context, c *testclient.GameConn, part string) testclient.SystemMessage {
	t.Helper()
	return await(t, ctx, c, func(m testclient.SystemMessage) bool {
		return m.ID == testclient.SysMsgText && slices.ContainsFunc(m.Texts(), func(s string) bool {
			return strings.Contains(s, part)
		})
	})
}

// checkMove checks a MoveToLocation: it starts close to from and ends at to.
func checkMove(t *testing.T, what string, m testclient.MoveToLocation, from, to testclient.Position) {
	t.Helper()
	if !near(m.Destination(), to) {
		t.Errorf("%s goes to %v, want %v", what, m.Destination(), to)
	}
	if d := dist2D(m.Origin(), from); d > 50 {
		t.Errorf("%s starts at %v, %.0f units from %v", what, m.Origin(), d, from)
	}
}

// walkTime is how long a new Human Fighter takes to run from from to to,
// with room for the tick rounding and a slow machine.
func walkTime(from, to testclient.Position) time.Duration {
	secs := from.DistanceTo(to) / runSpeed
	return time.Duration(secs*1.1*float64(time.Second)) + 2*time.Second
}

// findNpc waits until c sees an NPC of template id and returns the nearest
// one.
func findNpc(t *testing.T, ctx context.Context, c *testclient.GameConn, id int32) testclient.Object {
	t.Helper()
	var o testclient.Object
	poll(t, ctx, fmt.Sprintf("the NpcInfo of an NPC %d", id), func() (bool, error) {
		var ok bool
		o, ok = c.World().FindNpc(id)
		return ok, nil
	})
	return o
}

// findCharInfo returns the CharInfo of the player id in ps.
func findCharInfo(ps []testclient.Packet, id int32) (testclient.CharInfoPacket, bool) {
	for _, c := range testclient.FindAll[testclient.CharInfoPacket](ps) {
		if c.ObjectID == id {
			return c, true
		}
	}
	return testclient.CharInfoPacket{}, false
}

// subsequence reports whether want appears in got in order, with anything
// in between.
func subsequence(want, got []string) bool {
	i := 0
	for _, g := range got {
		if i < len(want) && g == want[i] {
			i++
		}
	}
	return i == len(want)
}
