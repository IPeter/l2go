package scenario

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// nameAlphabet makes the random part of account and character names. It
// has no vowels, so a generated name never contains one of the forbidden
// words of Mobius and L2Go (annou, amou, ...: all need an "a").
const nameAlphabet = "bcdfghjklmnpqrstvwxz0123456789"

// nameSuffixLen is the length of the random part. With the two-letter
// prefix, accounts stay within the 14 characters of the login packet and
// characters within the 16 of the game protocol.
const nameSuffixLen = 10

// randomSuffix returns n random characters of nameAlphabet.
func randomSuffix(rng *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = nameAlphabet[rng.IntN(len(nameAlphabet))]
	}
	return string(b)
}

// newIdentity returns fresh names for role: a lower-case account (the
// servers store accounts in lower case) and a capitalized character name,
// with independent random parts.
func newIdentity(rng *rand.Rand, role string) Identity {
	return Identity{
		Role:    role,
		Account: "rd" + randomSuffix(rng, nameSuffixLen),
		Char:    "Rd" + randomSuffix(rng, nameSuffixLen),
	}
}

// Link is a dialog link: a bypass (RequestBypassToServer) or a page link
// (RequestLinkHtml).
type Link struct {
	Bypass bool
	// Target is what the client sends: the bypass command without "bypass
	// -h", or the page path.
	Target string
}

func (l Link) String() string {
	if l.Bypass {
		return "bypass " + l.Target
	}
	return "link " + l.Target
}

// Send sends the link the way the client does when it is clicked.
func (l Link) Send(ctx context.Context, gc *testclient.GameConn) error {
	if l.Bypass {
		return gc.Bypass(ctx, l.Target)
	}
	return gc.LinkHtml(ctx, l.Target)
}

// linkRe matches the action of a dialog link: action="bypass -h cmd",
// action="bypass cmd" or action="link page".
var linkRe = regexp.MustCompile(`(?i)action\s*=\s*"\s*(bypass(?:\s+-h)?|link)\s+([^"]+?)\s*"`)

// FirstLink returns the first link of a dialog page.
func FirstLink(html string) (Link, bool) {
	m := linkRe.FindStringSubmatch(html)
	if m == nil {
		return Link{}, false
	}
	return Link{Bypass: !strings.EqualFold(m[1], "link"), Target: m[2]}, true
}

// NearestTalkable returns the NPC of objs (shown by NpcInfo, not
// attackable) nearest to me, and its distance. Ties go to the lower
// template ID, then the lower object ID.
func NearestTalkable(me testclient.Position, objs []testclient.Object) (testclient.Object, float64, bool) {
	var (
		best     testclient.Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		if o.Kind != testclient.KindNpc || o.Attackable {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if !found || d < bestDist || (d == bestDist && cmp.Or(cmp.Compare(o.NpcID, best.NpcID), cmp.Compare(o.ObjectID, best.ObjectID)) < 0) {
			best, bestDist, found = o, d, true
		}
	}
	return best, bestDist, found
}

// walkDistance is how far enter-two-players walks.
const walkDistance = 500

// walkTime is how long the character of u takes to walk dist units, plus a
// second for the packets after the arrival, at most 15 s.
func walkTime(u testclient.UserInfo, dist float64) time.Duration {
	mult := u.MoveMultiplier
	if mult <= 0 {
		mult = 1
	}
	speed := float64(u.WalkSpd) * mult
	if u.Running {
		speed = float64(u.RunSpd) * mult
	}
	if speed <= 0 {
		speed = 100
	}
	d := time.Duration(dist/speed*float64(time.Second)) + time.Second
	return min(d, 15*time.Second)
}

// pollInterval is how often waitSees looks at the world model.
const pollInterval = 50 * time.Millisecond

// waitSees waits until viewer's world model shows other's character.
func waitSees(ctx context.Context, viewer, other *Player) error {
	t := time.NewTicker(pollInterval)
	defer t.Stop()
	for {
		if _, ok := viewer.game.World().FindPlayer(other.Char); ok {
			return nil
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return fmt.Errorf("%s does not see %s: %w", viewer.Role, other.Role, ctx.Err())
		}
	}
}

// awaitChat waits for p to receive the chat message text of type typ.
func awaitChat(ctx context.Context, p *Player, typ testclient.ChatType, text string) error {
	_, err := testclient.Await(ctx, p.game, func(m testclient.CreatureSay) bool {
		return m.Type == typ && m.Text == text
	})
	if err != nil {
		return fmt.Errorf("%s: %w", p.Role, err)
	}
	return nil
}

// awaitMove waits for viewer to see the object id start moving.
func awaitMove(ctx context.Context, viewer *Player, id int32) error {
	_, err := testclient.Await(ctx, viewer.game, func(m testclient.MoveToLocation) bool { return m.ObjectID == id })
	return err
}

// awaitDelete waits for viewer to see the object id disappear.
func awaitDelete(ctx context.Context, viewer *Player, id int32) error {
	_, err := testclient.Await(ctx, viewer.game, func(d testclient.DeleteObject) bool { return d.ObjectID == id })
	return err
}
