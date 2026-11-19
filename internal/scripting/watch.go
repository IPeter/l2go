package scripting

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultDebounce is how long Watch waits after the last change before it
// reloads. Editors often write a file in several steps.
const DefaultDebounce = 200 * time.Millisecond

// Watch prepares the scripts again whenever a file in the scripts folder
// changes, until ctx is done, and hands each successful Prepare to swap.
// swap runs on the watcher's goroutine; it must get Engine.Swap called on
// the world loop, for example:
//
//	engine.Watch(ctx, scripting.DefaultDebounce, func(p *scripting.Prepared) {
//		loop.Post(func() { engine.Swap(p) })
//	})
//
// Changes closer together than debounce cause one reload. A failed load is
// logged and the old scripts stay. Watch returns an error only if watching
// cannot start.
func (e *Engine) Watch(ctx context.Context, debounce time.Duration, swap func(*Prepared)) error {
	if swap == nil {
		return errors.New("scripting: watch: no swap function")
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("scripting: watch: %w", err)
	}
	defer w.Close()
	if err := watchTree(w, e.opts.Dir); err != nil {
		return fmt.Errorf("scripting: watch: %w", err)
	}
	e.opts.Logger.Info("reloading scripts on change", "dir", e.opts.Dir)

	timer := time.NewTimer(debounce)
	timer.Stop()
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if ev.Has(fsnotify.Create) {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					if err := watchTree(w, ev.Name); err != nil {
						e.opts.Logger.Warn("cannot watch new script folder", "dir", ev.Name, "err", err)
					}
				}
			}
			if relevant(ev) {
				timer.Reset(debounce)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			e.opts.Logger.Warn("script watcher error", "err", err)
		case <-timer.C:
			p, err := e.Prepare(ctx)
			if err != nil {
				if ctx.Err() == nil {
					e.opts.Logger.Error("script reload failed, keeping the old scripts", "err", err)
				}
				continue
			}
			e.opts.Logger.Debug("scripts prepared after a change", "dir", e.opts.Dir, "files", p.Files())
			swap(p)
		}
	}
}

// relevant reports whether a change can affect the loaded scripts: any
// change except chmod to a .lua file or to something without an
// extension (a folder).
func relevant(ev fsnotify.Event) bool {
	if ev.Op == fsnotify.Chmod {
		return false
	}
	base := filepath.Base(ev.Name)
	if strings.HasPrefix(base, ".") {
		return false
	}
	ext := filepath.Ext(base)
	return ext == ".lua" || ext == ""
}

// watchTree adds dir and every folder below it, except hidden ones.
func watchTree(w *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != dir && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		return w.Add(path)
	})
}
