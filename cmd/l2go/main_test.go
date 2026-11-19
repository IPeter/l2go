package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
		{
			[]string{"game", "-dev-reload", "-config", "x.json"},
			options{command: "game", serverID: 1, devReload: true, configPath: "x.json", logLevel: slog.LevelInfo},
		},
		{[]string{"scripts", "check"}, options{command: "scripts", scriptsDir: "scripts", logLevel: slog.LevelInfo}},
		{
			[]string{"scripts", "check", "-dir", "/opt/l2go/scripts", "-log-level", "error"},
			options{command: "scripts", scriptsDir: "/opt/l2go/scripts", logLevel: slog.LevelError},
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
		{"login", "-dev-reload"}, // only for game
		{"scripts"},
		{"scripts", "run"},
		{"scripts", "check", "-config", "x.json"},
		{"scripts", "check", "extra"},
	}
	for _, args := range tests {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("parseArgs(%q) succeeded, want an error", args)
		}
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"help"}, {"login", "-h"}, {"game", "-help"}, {"scripts", "-h"}, {"scripts", "check", "-h"}} {
		if _, err := parseArgs(args, io.Discard); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("parseArgs(%q) = %v, want flag.ErrHelp", args, err)
		}
	}
}

func TestCheckScripts(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	// The scripts shipped with the server.
	var stdout, stderr bytes.Buffer
	dir := filepath.Join("..", "..", "scripts")
	if err := checkScripts(t.Context(), dir, &stdout, &stderr, logger); err != nil {
		t.Fatalf("checkScripts(%s) = %v\n%s", dir, err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") {
		t.Errorf("stdout = %q", stdout.String())
	}

	// Every problem is printed with file and line.
	dir = t.TempDir()
	for name, src := range map[string]string{
		"a.lua": "local x = = 1",
		"b.lua": "\nerror('boom')",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if err := checkScripts(t.Context(), dir, &stdout, &stderr, logger); !errors.Is(err, errReported) {
		t.Fatalf("checkScripts = %v, want errReported", err)
	}
	for _, want := range []string{"a.lua:1:", "b.lua:2: boom", "2 problem(s)"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr is missing %q:\n%s", want, stderr.String())
		}
	}

	// A missing folder is an ordinary error.
	if err := checkScripts(t.Context(), filepath.Join(dir, "missing"), &stdout, &stderr, logger); err == nil || errors.Is(err, errReported) {
		t.Errorf("checkScripts(missing) = %v", err)
	}
}
