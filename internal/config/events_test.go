package config

import (
	"strings"
	"testing"
)

func TestEventsConfig(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameServers": [{"id": 1, "events": {"dates": {
		"L2Day": {"active": "26 11 2026-03 12 2026", "dropPeriod": "27 11-01 12"},
		"Christmas": {"active": "01 12-24 12"}}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.GameServers[0].Events
	if e.ItemMaxLevelDifference != 9 {
		t.Errorf("eventItemMaxLevelDifference = %d, want the default 9", e.ItemMaxLevelDifference)
	}
	if d := e.Dates["L2Day"]; d.Active != "26 11 2026-03 12 2026" || d.DropPeriod != "27 11-01 12" {
		t.Errorf("L2Day dates %+v", d)
	}
	if errs := cfg.GameServers[0].validateEvents(); len(errs) != 0 {
		t.Errorf("validateEvents: %v", errs)
	}
	if DefaultEvents().Dates != nil {
		t.Error("the default moves an event")
	}
}

func TestEventsConfigErrors(t *testing.T) {
	g := GameServer{Events: Events{ItemMaxLevelDifference: -1, Dates: map[string]EventDates{
		"A": {Active: "26-11-2026"},
		"B": {Active: "26 11-03 12", DropPeriod: "x"},
	}}}
	var msgs []string
	for _, err := range g.validateEvents() {
		msgs = append(msgs, err.Error())
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"eventItemMaxLevelDifference", "events.dates.A.active", "events.dates.B.dropPeriod"} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in\n%s", want, all)
		}
	}
}
