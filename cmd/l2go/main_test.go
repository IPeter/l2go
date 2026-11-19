package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args []string
		want options
	}{
		{[]string{"login"}, options{command: "login", logLevel: slog.LevelInfo}},
		{
			[]string{"login", "-config", "/etc/l2go/server.json", "-log-level", "debug"},
			options{command: "login", configPath: "/etc/l2go/server.json", logLevel: slog.LevelDebug},
		},
		{[]string{"game"}, options{command: "game", serverID: 1, logLevel: slog.LevelInfo}},
		{
			[]string{"game", "-id", "3", "-log-level", "WARN"},
			options{command: "game", serverID: 3, logLevel: slog.LevelWarn},
		},
	}
	for _, tt := range tests {
		got, err := parseArgs(tt.args, io.Discard)
		if err != nil {
			t.Errorf("parseArgs(%q) error: %v", tt.args, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseArgs(%q) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := [][]string{
		nil,
		{"serve"},
		{"login", "-id", "1"}, // -id only exists for game
		{"game", "-id", "x"},
		{"login", "-log-level", "loud"},
		{"login", "extra"},
	}
	for _, args := range tests {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("parseArgs(%q) succeeded, want an error", args)
		}
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"help"}, {"login", "-h"}, {"game", "-help"}} {
		if _, err := parseArgs(args, io.Discard); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseArgs(%q) = %v, want flag.ErrHelp", args, err)
		}
	}
}
