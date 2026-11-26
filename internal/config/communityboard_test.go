package config

import "testing"

func TestCommunityBoardConfig(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameServers": [{"id": 1}, {"id": 2, "communityBoard": {"enableCommunityBoard": false, "communityBoardPeaceOnly": true}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.GameServers[0].CommunityBoard, DefaultCommunityBoard(); got != want {
		t.Errorf("default section = %+v, want %+v", got, want)
	}
	if want := (CommunityBoard{Enabled: true, Default: "_bbshome", KarmaDisabled: true}); DefaultCommunityBoard() != want {
		t.Errorf("DefaultCommunityBoard = %+v, want the Mobius defaults %+v", DefaultCommunityBoard(), want)
	}
	cb := cfg.GameServers[1].CommunityBoard
	if cb.Enabled || !cb.PeaceOnly || cb.Default != "_bbshome" || !cb.KarmaDisabled {
		t.Errorf("partial section = %+v", cb)
	}
	if errs := (GameServer{CommunityBoard: CommunityBoard{Default: " "}}).validateCommunityBoard(); len(errs) != 1 {
		t.Errorf("an empty bbsDefault: %v", errs)
	}
	if errs := cfg.GameServers[0].validateCommunityBoard(); len(errs) != 0 {
		t.Errorf("validateCommunityBoard: %v", errs)
	}
}
