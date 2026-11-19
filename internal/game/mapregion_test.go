package game

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

func TestLocCommand(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiVillage})

	loc := func() []testclient.Packet {
		t.Helper()
		e.srv.Handle(sess, clientpackets.OpBypassUserCmd, int32Body(clientpackets.UserCmdLoc))
		e.sync()
		return sess.take(t)
	}

	// In the Talking Island village: message 910, "Current location : $s1,
	// $s2, $s3 (Near Talking Island Village)", with the coordinates.
	ps := loc()
	checkNames(t, "/loc", ps, "SystemMessage")
	m := ps[0].(testclient.SystemMessage)
	pos := e.position(p)
	if m.ID != 910 || len(m.Params) != 3 ||
		m.Params[0].Number != int64(pos.X) || m.Params[1].Number != int64(pos.Y) || m.Params[2].Number != int64(pos.Z) {
		t.Errorf("/loc in the village: %+v, want 910 with %v", m, pos)
	}
	for i, prm := range m.Params {
		if prm.Type != testclient.SMParamInt {
			t.Errorf("parameter %d has type %d, want an int", i, prm.Type)
		}
	}

	// Where the map has no region: the coordinates as text.
	nowhere := world.Position{X: world.WorldXMin + 10, Y: world.WorldYMin + 10, Z: 0}
	if _, ok := testWorldData(t).MapRegionAt(int(nowhere.X), int(nowhere.Y)); ok {
		t.Skip("the world corner has a map region in the data")
	}
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, nowhere); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	sess.take(t)
	if msgs := messages(loc()); len(msgs) != 1 || msgs[0] != "Current location: -131062, -262134, 0" {
		t.Errorf("/loc without a region: %q", msgs)
	}
}

func TestRestartRegionLocID(t *testing.T) {
	w := testWorldData(t)
	def := w.DefaultRegion().LocID
	for _, r := range w.MapRegions {
		if got := restartRegionLocID(w, data.RaceHuman, r.Name); r.BannedRaces[data.RaceHuman] == "" && got != r.LocID {
			t.Errorf("region %s: %d, want its own %d", r.Name, got, r.LocID)
		}
		for race, other := range r.BannedRaces {
			target, _ := w.MapRegion(other)
			if got := restartRegionLocID(w, race, r.Name); target != nil && target.BannedRaces[race] == "" && got != target.LocID {
				t.Errorf("region %s bans race %v to %s: %d, want %d", r.Name, race, other, got, target.LocID)
			}
		}
	}
	if got := restartRegionLocID(w, data.RaceHuman, "no_such_region"); got != def {
		t.Errorf("a missing region gives %d, want the default %d", got, def)
	}
	if got := restartRegionLocID(w, data.RaceHuman, ""); got != def {
		t.Errorf("no region gives %d, want the default %d", got, def)
	}
}

func TestLocMessageParams(t *testing.T) {
	for _, tc := range []struct {
		id   int
		want bool
	}{{910, true}, {926, true}, {1537, true}, {1924, true}, {927, false}, {2189, false}, {3065, false}} {
		if got := locMessageHasCoords(tc.id); got != tc.want {
			t.Errorf("locMessageHasCoords(%d) = %v, want %v", tc.id, got, tc.want)
		}
	}
}
