package game

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// bossWriteTimeout bounds one write of the boss state.
const bossWriteTimeout = 10 * time.Second

// bossWriter writes the boss state in the background (decision A8, the
// W13 pattern): the loop hands it write jobs, which run one at a time in the
// order they came, on a goroutine of their own that ends when nothing is
// left. A failed write is logged, as Mobius logs it; the state stays on the
// loop and the next write of the same rows carries it. enqueue never
// blocks. Safe for concurrent use.
type bossWriter struct {
	log *slog.Logger

	mu      sync.Mutex
	queue   []bossJob
	running bool
	idle    []chan struct{} // closed when the queue runs dry
}

type bossJob struct {
	what string
	fn   func(ctx context.Context) error
}

func newBossWriter(log *slog.Logger) *bossWriter { return &bossWriter{log: log} }

// enqueue hands a write to the writer.
func (w *bossWriter) enqueue(what string, fn func(ctx context.Context) error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.queue = append(w.queue, bossJob{what: what, fn: fn})
	if !w.running {
		w.running = true
		go w.run()
	}
}

// run writes until the queue is empty.
func (w *bossWriter) run() {
	for {
		w.mu.Lock()
		if len(w.queue) == 0 {
			w.running = false
			for _, ch := range w.idle {
				close(ch)
			}
			w.idle = nil
			w.mu.Unlock()
			return
		}
		job := w.queue[0]
		w.queue[0] = bossJob{}
		w.queue = w.queue[1:]
		w.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), bossWriteTimeout)
		if err := job.fn(ctx); err != nil {
			w.log.Error("game: cannot save the boss state", "what", job.what, "err", err)
		}
		cancel()
	}
}

// wait returns once every write handed over so far is done, or with ctx's
// error.
func (w *bossWriter) wait(ctx context.Context) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	w.idle = append(w.idle, ch)
	w.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
