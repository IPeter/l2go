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
