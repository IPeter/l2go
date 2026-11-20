package game

import (
	"encoding/binary"
	"testing"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/testclient"
)

// compassCodes returns the codes of the ExSetCompassZoneCode packets in ps.
func compassCodes(ps []testclient.Packet) []serverpackets.CompassZone {
	var out []serverpackets.CompassZone
	for _, p := range ps {
		u, ok := p.(testclient.Unknown)
		if sub, ex := u.ExOpcode(); !ok || !ex || sub != serverpackets.ExOpSetCompassZoneCode || len(u.Body) != 6 {
			continue
		}
		out = append(out, serverpackets.CompassZone(binary.LittleEndian.Uint32(u.Body[2:]))) //nolint:gosec // same bits
	}
	return out
}

// The compass code goes out when the zones are worked out and the code
// changed (Mobius Player.revalidateZone): on entering, but not again for
// the same zones.
func TestCompassZoneCode(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess := newSession("hero")
	ch := e.createCharacter("hero", "Hero", classHumanFighter, 0)
	ch.X, ch.Y, ch.Z = tiStart.X, tiStart.Y, tiStart.Z
	if err := e.store.Update(e.ctx, &ch); err != nil {
		t.Fatalf("Update: %v", err)
	}
	p := e.enter(sess, 0)
	var want serverpackets.CompassZone
	e.onLoop(func() { want = compassZone(p) })
	if want != serverpackets.CompassPeace {
		t.Errorf("compass code at the Talking Island start %#x, want peace %#x", want, serverpackets.CompassPeace)
	}
	if got := compassCodes(sess.take(t)); len(got) != 1 || got[0] != want {
		t.Errorf("compass codes on entering %#x, want [%#x]", got, want)
	}

	e.onLoop(func() { e.srv.revalidateZones(p, true) })
	if got := compassCodes(sess.take(t)); len(got) != 0 {
		t.Errorf("compass codes for the same zones %#x, want none", got)
	}
}
