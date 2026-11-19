package scripting

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watch starts e.Watch until the test ends.
func watch(t *testing.T, e *Engine) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.Watch(ctx, 20*time.Millisecond) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Watch = %v", err)
		}
	})
}

func TestWatchReloadsOnChange(t *testing.T) {
	logs := &logBuffer{}
	dir := writeScripts(t, map[string]string{"events/names.lua": rejectName("Old", ReasonIncorrectName)})
	e := newEngine(t, nil, Options{Dir: dir, Logger: logs.logger()})
	watch(t, e)
	waitFor(t, "the watcher to start", func() bool {
		_, ok := logs.find(t, "reloading scripts on change")
		return ok
	})

	writeScript(t, dir, "events/names.lua", rejectName("New", ReasonIncorrectName))
	waitFor(t, "the change to load", func() bool { return call(t, e, "New").Rejected })
	wantAccepted(t, call(t, e, "Old"))

	// A new folder is watched too.
	writeScript(t, dir, "quests/q1.lua", rejectName("Quest", ReasonNameExists))
	waitFor(t, "the new folder to load", func() bool { return call(t, e, "Quest").Rejected })
}

func TestWatchKeepsOldOnBrokenChange(t *testing.T) {
	logs := &logBuffer{}
	dir := writeScripts(t, map[string]string{"events/names.lua": rejectName("Old", ReasonIncorrectName)})
	e := newEngine(t, nil, Options{Dir: dir, Logger: logs.logger()})
	watch(t, e)
	waitFor(t, "the watcher to start", func() bool {
		_, ok := logs.find(t, "reloading scripts on change")
		return ok
	})

	writeScript(t, dir, "events/names.lua", "events.on_character_create(")
	waitFor(t, "the reload error", func() bool {
		rec, ok := logs.find(t, "script reload failed, keeping the old scripts")
		msg, _ := rec["err"].(string)
		return ok && strings.Contains(msg, "names.lua")
	})
	wantRejected(t, call(t, e, "Old"), ReasonIncorrectName)
}

func TestWatchMissingFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "scripts")
	writeScript(t, dir, "a.lua", "")
	e := newEngine(t, nil, Options{Dir: dir})
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := e.Watch(t.Context(), time.Millisecond); err == nil {
		t.Error("Watch succeeded on a missing folder")
	}
}

func TestRelevant(t *testing.T) {
	tests := []struct {
		ev   fsnotify.Event
		want bool
	}{
		{fsnotify.Event{Name: "scripts/a.lua", Op: fsnotify.Write}, true},
		{fsnotify.Event{Name: "scripts/a.lua", Op: fsnotify.Create}, true},
		{fsnotify.Event{Name: "scripts/a.lua", Op: fsnotify.Remove}, true},
		{fsnotify.Event{Name: "scripts/quests", Op: fsnotify.Create}, true},
		{fsnotify.Event{Name: "scripts/a.lua", Op: fsnotify.Chmod}, false},
		{fsnotify.Event{Name: "scripts/README.md", Op: fsnotify.Write}, false},
		{fsnotify.Event{Name: "scripts/.a.lua.swp", Op: fsnotify.Write}, false},
	}
	for _, tt := range tests {
		if got := relevant(tt.ev); got != tt.want {
			t.Errorf("relevant(%v) = %v, want %v", tt.ev, got, tt.want)
		}
	}
}
