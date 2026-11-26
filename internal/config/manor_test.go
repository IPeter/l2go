package config

import (
	"strings"
	"testing"
)

// The Mobius General.ini manor values and Rates.ini RateDropManor.
func TestDefaultManor(t *testing.T) {
	want := Manor{AllowManor: true, RefreshTime: 20, ApproveTime: 6, MaintenanceMin: 6, RateDropManor: 1}
	if got := Default().GameServers[0].Manor; got != want {
		t.Errorf("Manor = %+v, want %+v", got, want)
	}
}

// Every key is read, a section left out keeps its defaults, and explicit
// false and 0 are kept.
func TestParseManor(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "manor": {"_comment": "c", "allowManor": false,
		"altManorRefreshTime": 21, "altManorRefreshMin": 30, "altManorApproveTime": 7, "altManorApproveMin": 15,
		"altManorMaintenanceMin": 0, "rateDropManor": 3}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Manor{Comment: "c", RefreshTime: 21, RefreshMin: 30, ApproveTime: 7, ApproveMin: 15, RateDropManor: 3}
	if got := cfg.GameServers[0].Manor; got != want {
		t.Errorf("Manor = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].Manor; got != DefaultManor() {
		t.Errorf("left out: Manor = %+v, want the defaults", got)
	}
}

func TestValidateManor(t *testing.T) {
	gs := Default().GameServers[0]
	if errs := gs.validateManor(); len(errs) != 0 {
		t.Errorf("defaults: %v", errs)
	}
	gs.Manor.RefreshTime = 24
	gs.Manor.RateDropManor = 0
	errs := gs.validateManor()
	wants := []string{"manor.altManorRefreshTime: 24", "manor.rateDropManor: 0"}
	if len(errs) != len(wants) {
		t.Fatalf("validateManor = %v, want %d errors", errs, len(wants))
	}
	for i, want := range wants {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %v, want %q", i, errs[i], want)
		}
	}
	cfg := Default()
	cfg.GameServers[0].Manor.ApproveMin = 60
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].manor.altManorApproveMin") {
		t.Errorf("Validate = %v, want the manor error with its path", err)
	}
}
