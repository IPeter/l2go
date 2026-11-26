package scenario

import (
	"testing"

	"github.com/IPeter/l2go/internal/testclient"
)

// TestSocialScenarioSpots checks the fixed spots of the social scenarios:
// next to Bitz's Mobius spawn for clan-create, within trade range of each
// other for trade, and bob next to alice in the keltir field for
// party-hunt.
func TestSocialScenarioSpots(t *testing.T) {
	bitz := testclient.Position{X: -83326, Y: 242964, Z: -3720}
	for name, spot := range map[string]testclient.Position{"alice": spotBitzAlice, "bob": spotBitzBob} {
		if d := spot.DistanceTo(bitz); d > talkRange {
			t.Errorf("%s's spot is %.0f units from Bitz", name, d)
		}
	}
	if d := spotDropAlice.DistanceTo(spotDropBob); d > 150 {
		t.Errorf("the trade spots are %.0f units apart, over Mobius's 150", d)
	}
	if d := spotKeltirBob.DistanceTo(keltirField); d > 100 {
		t.Errorf("bob waits %.0f units from the keltir field", d)
	}
	for _, name := range []string{"party-hunt", "trade", "friends", "clan-create"} {
		sc, ok := Find(name)
		if !ok || len(sc.Roles) != 2 {
			t.Errorf("Find(%s) = %+v, %v; want two roles", name, sc, ok)
		}
	}
}
