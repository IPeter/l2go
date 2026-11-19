package scripting

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// The deprecated asynchronous API keeps working until every caller runs on
// the world loop.
func TestDeprecatedRunCallReload(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.lua": rejectName("Old", ReasonNameExists)})
	e, err := New(context.Background(), Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.Run(ctx) }()

	res, err := e.Call(context.Background(), createEvent("Old"))
	if err != nil {
		t.Fatal(err)
	}
	wantRejected(t, res, ReasonNameExists)

	writeScript(t, dir, "a.lua", rejectName("New", ReasonNameExists))
	if err := e.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if res, _ := e.Call(context.Background(), createEvent("New")); !res.Rejected {
		t.Errorf("after Reload: %+v", res)
	}

	canceled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	if _, err := e.Call(canceled, createEvent("New")); !errors.Is(err, context.Canceled) {
		t.Errorf("Call with a canceled context = %v, want context.Canceled", err)
	}

	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run = %v", err)
	}
	if _, err := e.Call(context.Background(), createEvent("New")); !errors.Is(err, ErrStopped) {
		t.Errorf("Call after Run returned = %v, want ErrStopped", err)
	}
	if err := e.Reload(context.Background()); !errors.Is(err, ErrStopped) {
		t.Errorf("Reload after Run returned = %v, want ErrStopped", err)
	}
}

// Watch with SwapLocked runs next to the deprecated Call. Run with -race.
func TestDeprecatedWatchSwapLocked(t *testing.T) {
	logs := &logBuffer{}
	dir := writeScripts(t, map[string]string{"events/names.lua": rejectName("Old", ReasonIncorrectName)})
	e := newEngine(t, nil, Options{Dir: dir, Logger: logs.logger()})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- e.Watch(ctx, 20*time.Millisecond, e.SwapLocked) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Watch = %v", err)
		}
	}()
	waitFor(t, "the watcher to start", func() bool {
		_, ok := logs.find(t, "reloading scripts on change")
		return ok
	})

	writeScript(t, dir, "events/names.lua", rejectName("New", ReasonIncorrectName))
	waitFor(t, "the change to load", func() bool {
		res, err := e.Call(context.Background(), createEvent("New"))
		return err == nil && res.Rejected
	})
}

// The deprecated Call and Reload may run on many goroutines at once. Run
// with -race.
func TestDeprecatedCallConcurrent(t *testing.T) {
	dir := writeScripts(t, map[string]string{"a.lua": `
events.on_character_create(function(ev)
  if ev.name:sub(1, 3) == "bad" then
    return events.reject("incorrect_name")
  end
end)`})
	e := newEngine(t, nil, Options{Dir: dir})

	const goroutines, calls = 8, 25
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range calls {
				name := fmt.Sprintf("good-%d-%d", g, i)
				if i%2 == 0 {
					name = fmt.Sprintf("bad-%d-%d", g, i)
				}
				res, err := e.Call(context.Background(), createEvent(name))
				if err != nil {
					t.Errorf("Call(%s): %v", name, err)
					return
				}
				if want := i%2 == 0; res.Rejected != want || res.Err != nil {
					t.Errorf("Call(%s) = %+v, want rejected=%v", name, res, want)
				}
			}
		})
	}
	wg.Go(func() {
		for range 3 {
			if err := e.Reload(context.Background()); err != nil {
				t.Errorf("Reload: %v", err)
			}
		}
	})
	wg.Wait()
}
