package scripting

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The five Mobius ferry schedules load from scripts/vehicles.
func TestVehicleRoutesOfTheScripts(t *testing.T) {
	e := newEngine(t, nil, Options{
		Dir:     filepath.Join("..", "..", "scripts"),
		Include: func(p string) bool { return strings.HasPrefix(p, "vehicles/") },
	})
	routes := e.VehicleRoutes()
	type summary struct {
		name  string
		boat  int
		steps int
		dock  int
	}
	var got []summary
	for _, r := range routes {
		got = append(got, summary{r.Name, r.BoatID, len(r.Steps), r.Dock})
		if r.StartDelay != 180*time.Second {
			t.Errorf("%s starts after %v, want 3m", r.Name, r.StartDelay)
		}
	}
	want := []summary{
		{"GiranTalking", 2, 19, 0},
		{"GludinRune", 3, 20, 2},
		{"InnadrilTour", 4, 10, 0},
		{"RunePrimeval", 5, 5, 3},
		{"TalkingGludin", 1, 18, 1},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("routes:\n got %+v\nwant %+v", got, want)
	}

	tg := routes[4]
	if tg.Start != (Location{X: -96622, Y: 261660, Z: -3610, Heading: 32768}) {
		t.Errorf("TalkingGludin starts at %+v", tg.Start)
	}
	leave := tg.Steps[3]
	if leave.Undock != 1 || leave.Announce == nil || !slices.Equal(leave.Announce.Messages, []int{985}) ||
		leave.Horn == nil || *leave.Horn != (Location{X: -96622, Y: 261660, Z: -3610}) ||
		leave.Pay == nil || *leave.Pay != (VehiclePay{Item: 1074, Count: 1, Oust: Location{X: -96777, Y: 258970, Z: -3623}}) ||
		len(leave.Path) != 10 || leave.Path[0] != (VehiclePoint{X: -121385, Y: 261660, Z: -3610, Speed: 180, Rotation: 800}) ||
		leave.Wait != 300*time.Second {
		t.Errorf("TalkingGludin step 3: %+v", leave)
	}
	if near := leave.Announce.Near; near[0] != (Location{X: -96622, Y: 261660, Z: -3610}) || near[1] != (Location{X: -95686, Y: 150514, Z: -3610}) {
		t.Errorf("step 3 is heard near %+v, want the Talking Island and Gludin docks", near)
	}
	call := tg.Steps[1]
	if !slices.Equal(call.Announce.Messages, []int{982, 983}) || call.Wait != 40*time.Second {
		t.Errorf("TalkingGludin step 1: %+v", call)
	}
	dock := tg.Steps[7]
	if dock.Await != 2 || dock.Busy == nil || !slices.Equal(dock.Busy.Messages, []int{1485}) || len(dock.Path) != 1 || dock.Wait != 0 {
		t.Errorf("TalkingGludin step 7: %+v", dock)
	}
	if arrive := tg.Steps[17]; arrive.Dock != 1 || !slices.Equal(arrive.Announce.Messages, []int{979, 980}) {
		t.Errorf("TalkingGludin step 17: %+v", arrive)
	}
	// Innadril's horn is part of the announcement near the dock.
	if h := routes[2].Steps[3].Announce.Horn; h == nil || *h != (Location{X: 111264, Y: 226240, Z: -3610}) {
		t.Errorf("Innadril departure horn: %v", h)
	}
}

func TestVehicleRouteErrors(t *testing.T) {
	const good = `{ name = "R", boat = 1, start = {1, 2, 3}, steps = { { wait = 1000 } } }`
	for _, tt := range []struct {
		name, src, want string
	}{
		{"unknown field", `vehicles.add_route{ name = "R", boat = 1, start = {1, 2, 3}, steps = { { wiat = 1 } } }`, "unknown field wiat"},
		{"no steps", `vehicles.add_route{ name = "R", boat = 1, start = {1, 2, 3}, steps = {} }`, "non-empty list"},
		{"no boat", `vehicles.add_route{ name = "R", start = {1, 2, 3}, steps = { {} } }`, "boat must be a positive integer"},
		{"short point", `vehicles.add_route{ name = "R", boat = 1, start = {1, 2}, steps = { {} } }`, "a point {x, y, z"},
		{"half speeds", `vehicles.add_route{ name = "R", boat = 1, start = {1, 2, 3}, steps = { { path = { {1, 2, 3, 4} } } } }`, "both the speed"},
		{"twice", "vehicles.add_route" + good + "\nvehicles.add_route" + good, "already declared"},
	} {
		dir := writeScripts(t, map[string]string{"vehicles/r.lua": tt.src})
		_, err := New(context.Background(), Options{Dir: dir})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error with %q", tt.name, err, tt.want)
		}
	}
}

// A path point without speeds sails at Mobius's defaults.
func TestVehicleRouteDefaults(t *testing.T) {
	e := newEngine(t, map[string]string{"vehicles/r.lua": `vehicles.add_route{
		name = "R", boat = 7, start = {1, 2, 3},
		steps = { { path = { {10, 20, 30} }, pay = { item = 0, oust = {4, 5, 6} } } },
	}`}, Options{})
	r := e.VehicleRoutes()
	if len(r) != 1 || r[0].StartDelay != 0 || r[0].Steps[0].Path[0] != (VehiclePoint{X: 10, Y: 20, Z: 30, Speed: 350, Rotation: 4000}) ||
		*r[0].Steps[0].Pay != (VehiclePay{Count: 1, Oust: Location{X: 4, Y: 5, Z: 6}}) {
		t.Errorf("routes: %+v", r)
	}
}
