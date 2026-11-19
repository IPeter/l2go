package netutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// Accept backoff limits after a failed Accept (for example, out of file
// descriptors).
const (
	minAcceptDelay = 5 * time.Millisecond
	maxAcceptDelay = time.Second
)

// AcceptLoop accepts connections from ln and runs handle for each one in a
// goroutine tracked by wg. handle owns the connection and must close it.
//
// It returns nil once ctx is done; the caller closes ln to stop a blocked
// Accept. It returns an error if ln is closed while ctx is still live.
// Other Accept errors are logged and retried with backoff.
func AcceptLoop(ctx context.Context, ln net.Listener, wg *sync.WaitGroup, logger *slog.Logger, handle func(net.Conn)) error {
	var delay time.Duration
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, net.ErrClosed) {
				return fmt.Errorf("accept on %s: %w", ln.Addr(), err)
			}
			delay = min(max(2*delay, minAcceptDelay), maxAcceptDelay)
			logger.Warn("accept failed, retrying", "addr", ln.Addr(), "err", err, "delay", delay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(delay):
			}
			continue
		}
		delay = 0
		wg.Go(func() { handle(conn) })
	}
}
