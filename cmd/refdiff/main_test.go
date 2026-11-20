package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/cmd/refdiff/recording"
	"github.com/IPeter/l2go/cmd/refdiff/scenario"
)

func TestParseArgsDefaults(t *testing.T) {
	opts, err := parseArgs([]string{"login-create-enter"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	wantL2Go := scenario.Target{Label: "l2go", LoginAddr: "127.0.0.1:2106", ServerID: 1, Password: "refdiff1"}
	wantRef := scenario.Target{Label: "reference", LoginAddr: "127.0.0.1:12106", ServerID: 2, Password: "refdiff1"}
	if opts.l2go != wantL2Go || opts.ref != wantRef {
		t.Errorf("targets = %+v, %+v; want %+v, %+v", opts.l2go, opts.ref, wantL2Go, wantRef)
	}
	if opts.scenario != "login-create-enter" || opts.side != "reference" || opts.json || opts.full || opts.stepTimeout <= 0 {
		t.Errorf("options = %+v", opts)
	}
	if !opts.defaultIgnores || opts.sample || opts.swings != scenario.DefaultSwings {
		t.Errorf("default ignores %v, sample %v, swings %d", opts.defaultIgnores, opts.sample, opts.swings)
	}
}

func TestParseArgsSample(t *testing.T) {
	opts, err := parseArgs([]string{"-sample", "-swings", "50", "-default-ignores=false", "sample-sword"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !opts.sample || opts.swings != 50 || opts.defaultIgnores || opts.scenario != "sample-sword" {
		t.Errorf("options = %+v", opts)
	}
}

func TestIgnorePatterns(t *testing.T) {
	sc, ok := scenario.Find("npc-talk")
	if !ok {
		t.Fatal("no npc-talk scenario")
	}
	opts := options{ignore: []string{"UserInfo.MaxLoad"}, defaultIgnores: true}
	got := ignorePatterns(sc, opts)
	if want := append(slices.Clone(sc.Ignore), "UserInfo.MaxLoad"); !slices.Equal(got, want) {
		t.Errorf("ignorePatterns = %q, want %q", got, want)
	}
	if !slices.Contains(got, "NpcInfo.Heading") || !slices.Contains(got, "SocialAction") {
		t.Errorf("npc-talk does not ignore the NPC noise: %q", got)
	}
	got[0] = "changed"
	if sc2, _ := scenario.Find("npc-talk"); sc2.Ignore[0] == "changed" {
		t.Error("ignorePatterns shares the scenario's slice")
	}
	opts.defaultIgnores = false
	if got := ignorePatterns(sc, opts); !slices.Equal(got, []string{"UserInfo.MaxLoad"}) {
		t.Errorf("without default ignores = %q", got)
	}
}

func TestParseArgsFlags(t *testing.T) {
	opts, err := parseArgs([]string{
		"-l2go-login", "10.0.0.1:2106", "-l2go-server", "3", "-l2go-game", "10.0.0.1:7777",
		"-ref-login", "10.0.0.2:2106", "-ref-server", "4",
		"-ignore", "NpcInfo, *.Heading", "-ignore", "UserInfo.MaxLoad",
		"-json", "-full", "chat",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.l2go.LoginAddr != "10.0.0.1:2106" || opts.l2go.ServerID != 3 || opts.l2go.GameAddr != "10.0.0.1:7777" {
		t.Errorf("l2go target = %+v", opts.l2go)
	}
	if opts.ref.LoginAddr != "10.0.0.2:2106" || opts.ref.ServerID != 4 {
		t.Errorf("reference target = %+v", opts.ref)
	}
	if want := []string{"NpcInfo", "*.Heading", "UserInfo.MaxLoad"}; !slices.Equal(opts.ignore, want) {
		t.Errorf("ignore = %q, want %q", opts.ignore, want)
	}
	if !opts.json || !opts.full || opts.scenario != "chat" {
		t.Errorf("options = %+v", opts)
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := [][]string{
		{},
		{"a", "b"},
		{"-save", "x.json", "-golden", "y.json", "chat"},
		{"-side", "middle", "chat"},
		{"-l2go-server", "0", "chat"},
		{"-ref-server", "256", "chat"},
		{"-password", "", "chat"},
		{"-step-timeout", "0s", "chat"},
		{"-log-level", "loud", "chat"},
		{"-sample", "-golden", "x.json", "sample-sword"},
		{"-sample", "-save", "x.json", "sample-sword"},
		{"-swings", "0", "sample-sword"},
	}
	for _, args := range tests {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("parseArgs(%q) accepted it", args)
		}
	}
	if _, err := parseArgs([]string{"-h"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h gives %v, want flag.ErrHelp", err)
	}
	for _, args := range [][]string{{"-list"}, {"-golden", "x.json"}} {
		if _, err := parseArgs(args, io.Discard); err != nil {
			t.Errorf("parseArgs(%q): %v", args, err)
		}
	}
}

func TestRunList(t *testing.T) {
	var out bytes.Buffer
	if code := run(t.Context(), options{list: true}, &out, io.Discard, slog.New(slog.DiscardHandler)); code != exitSame {
		t.Errorf("exit code %d", code)
	}
	for _, sc := range scenario.All() {
		if !strings.Contains(out.String(), sc.Name) {
			t.Errorf("-list lacks %s:\n%s", sc.Name, out.String())
		}
	}
	for _, want := range []string{"[GM] GM setup", "[-sample] for -sample"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("-list lacks %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "ignores ") {
		t.Errorf("-list shows the ignores without -full:\n%s", out.String())
	}
	out.Reset()
	run(t.Context(), options{list: true, full: true}, &out, io.Discard, slog.New(slog.DiscardHandler))
	if !strings.Contains(out.String(), "ignores NpcInfo.X,NpcInfo.Y") {
		t.Errorf("-list -full lacks the ignores:\n%s", out.String())
	}
}

func TestRunSampleWithoutServers(t *testing.T) {
	// Nothing listens on port 1: both sides fail at the login, and the
	// sample report says so instead of passing.
	down := scenario.Target{LoginAddr: "127.0.0.1:1", ServerID: 1, Password: "x"}
	l2go, ref := down, down
	l2go.Label, ref.Label = "l2go", "reference"
	opts := options{scenario: "sample-sword", sample: true, swings: 5, stepTimeout: time.Second, l2go: l2go, ref: ref}
	var stdout, stderr bytes.Buffer
	if code := run(t.Context(), opts, &stdout, &stderr, slog.New(slog.DiscardHandler)); code != exitDiff {
		t.Errorf("exit code %d, want %d", code, exitDiff)
	}
	if !strings.Contains(stdout.String(), "the samples differ") || !strings.Contains(stderr.String(), "failed on l2go") {
		t.Errorf("stdout %q, stderr %q", stdout.String(), stderr.String())
	}
}

func TestRunUsageErrors(t *testing.T) {
	golden := filepath.Join(t.TempDir(), "chat.json")
	if err := recording.Save(golden, recording.Recording{Scenario: "chat", Server: "reference"}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		opts options
		want string
	}{
		{"unknown scenario", options{scenario: "nope"}, "unknown scenario"},
		{"golden of another scenario", options{scenario: "restart", golden: golden}, `recording of scenario "chat"`},
		{"missing golden file", options{scenario: "chat", golden: golden + ".missing"}, "load"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run(t.Context(), tt.opts, io.Discard, &stderr, slog.New(slog.DiscardHandler))
			if code != exitUsage || !strings.Contains(stderr.String(), tt.want) {
				t.Errorf("exit code %d, stderr %q; want %d and %q", code, stderr.String(), exitUsage, tt.want)
			}
		})
	}
}
