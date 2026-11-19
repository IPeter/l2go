package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"
)

// Defaults of IDPoolOptions (decision I3).
const (
	DefaultIDPoolBlock      = 1000
	DefaultIDPoolLow        = 250
	DefaultIDPoolRetryDelay = time.Second
	DefaultIDPoolTimeout    = 10 * time.Second
)

// IDPoolOptions configures an IDPool. Zero fields take the defaults.
type IDPoolOptions struct {
	// Block is how many IDs one refill reserves (1,000).
	Block int
	// Low is the number of IDs left below which a refill starts (250).
	Low int
	// RetryDelay is how long after a failed refill the next one may start
	// (1 s).
	RetryDelay time.Duration
	// Timeout bounds one refill (10 s).
	Timeout time.Duration
	// Logger reports failed refills. The default is slog.Default().
	Logger *slog.Logger
}

// IDPool keeps a supply of object IDs reserved from an IDAllocator, so the
// world loop can give a new item an ID without waiting for the database
// (decision I3). TryNext never blocks: when fewer than Low IDs are left, a
// goroutine reserves another Block of them, and when none are left TryNext
// reports false and the action that needed the ID fails. Only IDs from
// FirstObjectID up to, not including, FirstRuntimeObjectID are handed out.
// IDs still in the pool when the server stops are never used, which costs
// nothing in a range of 1.3 billion.
//
// An IDPool is safe for concurrent use.
type IDPool struct {
	alloc IDAllocator
	opts  IDPoolOptions

	ctx    context.Context // refills run under it; Close cancels it
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu        sync.Mutex
	ids       []int32 // reserved and not yet handed out, in allocation order
	refilling bool
	failedAt  time.Time // when the last refill failed; zero after a success
	closed    bool
}

// errNoUsableIDs is the error of a refill whose IDs were all outside the
// persistent range.
var errNoUsableIDs = errors.New("storage: the allocator handed out no IDs below the runtime range")

// NewIDPool returns an empty pool that reserves IDs from alloc. Fill it
// before the first TryNext, and Close it when done.
func NewIDPool(alloc IDAllocator, opts IDPoolOptions) *IDPool {
	if opts.Block <= 0 {
		opts.Block = DefaultIDPoolBlock
	}
	if opts.Low <= 0 {
		opts.Low = DefaultIDPoolLow
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = DefaultIDPoolRetryDelay
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultIDPoolTimeout
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &IDPool{alloc: alloc, opts: opts, ctx: ctx, cancel: cancel}
}

// Fill reserves one block of IDs and waits for it, so the pool is ready
// before the world loop starts.
func (p *IDPool) Fill(ctx context.Context) error {
	ids, err := p.alloc.NextIDs(ctx, p.opts.Block)
	if err != nil {
		return fmt.Errorf("storage: fill the object ID pool: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.addLocked(ids) == 0 {
		return errNoUsableIDs
	}
	return nil
}

// TryNext returns a new object ID, or false when the pool is empty. It never
// blocks, and starts a refill when the pool runs low.
func (p *IDPool) TryNext() (int32, bool) {
	ids, ok := p.TryNextN(1)
	if !ok {
		return 0, false
	}
	return ids[0], true
}

// TryNextN returns n new object IDs, or false and none when fewer than n are
// left, so an action that needs several IDs gets all of them or none. Like
// TryNext it never blocks. n must be at least 1.
func (p *IDPool) TryNextN(n int) ([]int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var ids []int32
	ok := n >= 1 && n <= len(p.ids)
	if ok {
		ids = slices.Clone(p.ids[:n])
		p.ids = p.ids[n:]
	}
	p.refillLocked()
	return ids, ok
}

// Len returns how many IDs the pool holds.
func (p *IDPool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.ids)
}

// Close stops refilling and waits for a refill in progress. The IDs left in
// the pool can still be taken.
func (p *IDPool) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	p.cancel()
	p.wg.Wait()
}

// refillLocked starts a refill when the pool is low, unless one is running
// or the last one failed less than RetryDelay ago.
func (p *IDPool) refillLocked() {
	if p.closed || p.refilling || len(p.ids) >= p.opts.Low {
		return
	}
	if !p.failedAt.IsZero() && time.Since(p.failedAt) < p.opts.RetryDelay {
		return
	}
	p.refilling = true
	p.wg.Go(p.refill)
}

// refill reserves one block of IDs and adds it to the pool.
func (p *IDPool) refill() {
	ctx, cancel := context.WithTimeout(p.ctx, p.opts.Timeout)
	ids, err := p.alloc.NextIDs(ctx, p.opts.Block)
	cancel()

	p.mu.Lock()
	defer p.mu.Unlock()
	p.refilling = false
	if err == nil && p.addLocked(ids) == 0 {
		err = errNoUsableIDs
	}
	if err != nil {
		p.failedAt = time.Now()
		if p.ctx.Err() == nil {
			p.opts.Logger.Error("storage: cannot refill the object ID pool", "left", len(p.ids), "err", err)
		}
		return
	}
	p.failedAt = time.Time{}
}

// addLocked adds the IDs of the persistent range and returns how many it
// added. It reports the others.
func (p *IDPool) addLocked(ids []int32) int {
	n := 0
	for _, id := range ids {
		if id < FirstObjectID || id >= FirstRuntimeObjectID {
			continue
		}
		p.ids = append(p.ids, id)
		n++
	}
	if n < len(ids) {
		p.opts.Logger.Error("storage: object IDs outside the persistent range dropped",
			"dropped", len(ids)-n, "from", fmt.Sprintf("%#x", FirstObjectID), "below", fmt.Sprintf("%#x", FirstRuntimeObjectID))
	}
	return n
}
