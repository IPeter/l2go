package main

import (
	"errors"
	"flag"
	"io"
	"testing"
	"time"
)

func TestParseFlagsDefaults(t *testing.T) {
	cfg, err := parseFlags(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.n != 50 || cfg.login != "127.0.0.1:2106" || cfg.serverID != 1 || cfg.game != "" || cfg.prefix != "bot" {
		t.Errorf("config = %+v", cfg)
	}
	if cfg.rate != 10 || cfg.duration != 0 || cfg.radius != 300 || cfg.report != 10*time.Second || cfg.queue != 64 {
		t.Errorf("config = %+v", cfg)
	}
	if cfg.seed == 0 {
		t.Error("seed 0 was not replaced by a random one")
	}
	if cfg.mode != modeWander || cfg.class != classFighter || cfg.skill != 0 || cfg.cast != 5*time.Second || cfg.buffs != 0 || cfg.huntRange != 3000 {
		t.Errorf("mode %q, class %q, skill %d, cast %v, buffs %d, hunt range %v", cfg.mode, cfg.class, cfg.skill, cfg.cast, cfg.buffs, cfg.huntRange)
	}
}

func TestParseFlagsHunt(t *testing.T) {
	tests := []struct {
		args  []string
		class string
		skill int32
	}{
		{[]string{"-mode", "hunt"}, classMystic, skillWindStrike},
		{[]string{"-mode", "hunt", "-class", "fighter"}, classFighter, 0},
		{[]string{"-mode", "hunt", "-class", "fighter", "-skill", "3"}, classFighter, 3},
		{[]string{"-mode", "hunt", "-skill", "1184"}, classMystic, 1184},
		{[]string{"-class", "mystic"}, classMystic, skillWindStrike},
	}
	for _, tt := range tests {
		cfg, err := parseFlags(tt.args, io.Discard)
		if err != nil {
			t.Errorf("parseFlags(%q): %v", tt.args, err)
			continue
		}
		if cfg.class != tt.class || cfg.skill != tt.skill {
			t.Errorf("parseFlags(%q): class %q, skill %d; want %q, %d", tt.args, cfg.class, cfg.skill, tt.class, tt.skill)
		}
	}
	cfg, err := parseFlags([]string{"-mode", "hunt", "-buffs", "10", "-cast", "3s", "-hunt-range", "1500"}, io.Discard)
	if err != nil || cfg.mode != modeHunt || cfg.buffs != 10 || cfg.cast != 3*time.Second || cfg.huntRange != 1500 {
		t.Errorf("hunt flags: %+v, %v", cfg, err)
	}
}

func TestParseFlags(t *testing.T) {
	cfg, err := parseFlags([]string{
		"-n", "500", "-login", "10.0.0.1:2106", "-server", "2", "-game", "10.0.0.1:7777",
		"-prefix", "load", "-rate", "50", "-duration", "5m", "-seed", "42", "-say", "0.2", "-talk", "0.1",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.n != 500 || cfg.serverID != 2 || cfg.game != "10.0.0.1:7777" || cfg.prefix != "load" ||
		cfg.rate != 50 || cfg.duration != 5*time.Minute || cfg.seed != 42 || cfg.say != 0.2 || cfg.talk != 0.1 {
		t.Errorf("config = %+v", cfg)
	}
}

func TestParseFlagsErrors(t *testing.T) {
	tests := [][]string{
		{"-n", "0"},
		{"-prefix", "Bot"},
		{"-prefix", "1bot"},
		{"-prefix", "bot-1"},
		{"-prefix", ""},
		{"-prefix", "abcdefghijk"}, // 11 + 4 digits > 14
		{"-server", "0"},
		{"-server", "256"},
		{"-rate", "0"},
		{"-say", "0.8", "-talk", "0.5"},
		{"-say", "-0.1"},
		{"-step", "0s"},
		{"-queue", "0"},
		{"-password", ""},
		{"-duration", "-1s"},
		{"-log-level", "loud"},
		{"extra"},
		{"-mode", "fight"},
		{"-class", "archer"},
		{"-skill", "-1"},
		{"-skill", "3000000000"},
		{"-cast", "-1s"},
		{"-buffs", "-1"},
		{"-buffs", "11"},
		{"-hunt-range", "0"},
	}
	for _, args := range tests {
		if _, err := parseFlags(args, io.Discard); err == nil {
			t.Errorf("parseFlags(%q) accepted it", args)
		}
	}
	if _, err := parseFlags([]string{"-h"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h gives %v, want flag.ErrHelp", err)
	}
}

func TestValidPrefix(t *testing.T) {
	for p, want := range map[string]bool{"bot": true, "b": true, "load2": true, "": false, "Bot": false, "2bot": false, "b_t": false} {
		if got := validPrefix(p); got != want {
			t.Errorf("validPrefix(%q) = %v, want %v", p, got, want)
		}
	}
}
