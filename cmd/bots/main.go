// Command bots is the load test: it runs N players against a login server
// and its game server (L2Go, or the Mobius reference for comparison).
//
// Each bot logs in with its own account (the prefix and its number, such
// as bot0007; both servers create unknown accounts), creates a Human
// Fighter when the account has none, enters the world and then, every few
// seconds, reports its position and either walks to a random point around
// where it started, says something in general chat, or targets and talks
// to the nearest NPC. A failed bot tries again after a backoff. Ctrl+C or
// the end of -duration logs every bot out.
//
// Every -report interval and at the end, bots prints the bots in the
// world, the errors by kind, the packets per second in both directions and
// the latency percentiles of each request and its expected answer: move
// (MoveToLocation and the character's MoveToLocation back), say (Say2 and
// the CreatureSay echo), target (Action and MyTargetSelected), talk (the
// second Action and NpcHtmlMessage, including the walk to the NPC), login
// and enter.
//
// Usage:
//
//	bots [-n 50] [-login 127.0.0.1:2106] [-server 1] [-game addr] [-prefix bot] [-rate 10] [-duration 5m] [-seed N] ...
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// config is the parsed command line.
type config struct {
	n         int
	login     string
	serverID  byte
	game      string
	prefix    string
	password  string
	rate      float64
	duration  time.Duration
	seed      int64
	radius    float64
	step      time.Duration
	say       float64
	talk      float64
	talkRange float64
	report    time.Duration
	timeout   time.Duration
	queue     int
	logLevel  slog.Level
}

// maxAccountLength is the longest account name the login packet holds.
const maxAccountLength = 14

// parseFlags parses the arguments after the program name. A help request
// returns flag.ErrHelp.
func parseFlags(args []string, output io.Writer) (config, error) {
	var (
		cfg      config
		serverID int
		level    string
	)
	fs := flag.NewFlagSet("bots", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.IntVar(&cfg.n, "n", 50, "number of bots")
	fs.StringVar(&cfg.login, "login", "127.0.0.1:2106", "login server address")
	fs.IntVar(&serverID, "server", 1, "game server ID")
	fs.StringVar(&cfg.game, "game", "", "game server address (default: from the server list)")
	fs.StringVar(&cfg.prefix, "prefix", "bot", "account name prefix: lower-case letters and digits, starting with a letter")
	fs.StringVar(&cfg.password, "password", "botpass", "password of the bot accounts")
	fs.Float64Var(&cfg.rate, "rate", 10, "ramp-up: bots started per second")
	fs.DurationVar(&cfg.duration, "duration", 0, "how long to run (0: until Ctrl+C)")
	fs.Int64Var(&cfg.seed, "seed", 0, "random seed of the bots' behaviour (0: from the clock)")
	fs.Float64Var(&cfg.radius, "radius", 300, "random walk radius around the start point")
	fs.DurationVar(&cfg.step, "step", 2*time.Second, "mean time between two actions of a bot")
	fs.Float64Var(&cfg.say, "say", 0.1, "probability that an action is a chat message")
	fs.Float64Var(&cfg.talk, "talk", 0.05, "probability that an action is targeting and talking to the nearest NPC")
	fs.Float64Var(&cfg.talkRange, "talk-range", 1500, "how far away an NPC may be to talk to it")
	fs.DurationVar(&cfg.report, "report", 10*time.Second, "time between two reports")
	fs.DurationVar(&cfg.timeout, "timeout", 15*time.Second, "time limit of each request and its answer")
	fs.IntVar(&cfg.queue, "queue", 64, "packets each bot keeps unread at most (older ones are dropped)")
	fs.StringVar(&level, "log-level", "warn", "log level: debug, info, warn or error")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() > 0 {
		return config{}, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if err := cfg.logLevel.UnmarshalText([]byte(level)); err != nil {
		return config{}, fmt.Errorf("invalid -log-level %q", level)
	}
	if serverID < 1 || serverID > 255 {
		return config{}, fmt.Errorf("-server %d is not a game server ID (1-255)", serverID)
	}
	cfg.serverID = byte(serverID)
	if err := cfg.validate(); err != nil {
		return config{}, err
	}
	if cfg.seed == 0 {
		cfg.seed = time.Now().UnixNano()
	}
	return cfg, nil
}

// validate checks the values parseFlags cannot.
func (c config) validate() error {
	switch {
	case c.n < 1:
		return errors.New("-n must be at least 1")
	case !validPrefix(c.prefix):
		return fmt.Errorf("-prefix %q: use lower-case letters and digits, starting with a letter", c.prefix)
	case len(c.prefix)+nameDigits(c.n) > maxAccountLength:
		return fmt.Errorf("-prefix %q is too long for %d bots: accounts have at most %d characters", c.prefix, c.n, maxAccountLength)
	case c.password == "" || len(c.password) > 16:
		return errors.New("-password must have 1 to 16 characters")
	case c.rate <= 0:
		return errors.New("-rate must be positive")
	case c.duration < 0:
		return errors.New("-duration must not be negative")
	case c.radius < 0:
		return errors.New("-radius must not be negative")
	case c.step <= 0 || c.report <= 0 || c.timeout <= 0:
		return errors.New("-step, -report and -timeout must be positive")
	case c.say < 0 || c.talk < 0 || c.say+c.talk > 1:
		return errors.New("-say and -talk are probabilities with a sum of at most 1")
	case c.talkRange < 0:
		return errors.New("-talk-range must not be negative")
	case c.queue < 1:
		return errors.New("-queue must be at least 1")
	}
	return nil
}

// validPrefix reports whether p is a lower-case letter followed by
// lower-case letters and digits.
func validPrefix(p string) bool {
	if p == "" || p[0] < 'a' || p[0] > 'z' {
		return false
	}
	for i := 1; i < len(p); i++ {
		c := p[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func main() {
	cfg, err := parseFlags(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bots:", err)
		os.Exit(2)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.logLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// After the first Ctrl+C the bots log out; a second one kills the
	// program the usual way.
	go func() {
		<-ctx.Done()
		stop()
	}()
	if cfg.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.duration)
		defer cancel()
	}
	if err := run(ctx, cfg, os.Stdout, logger); err != nil {
		fmt.Fprintln(os.Stderr, "bots:", err)
		os.Exit(1)
	}
}

// run starts the bots at the ramp-up rate, reports every cfg.report and
// waits until every bot has stopped after ctx ends.
func run(ctx context.Context, cfg config, out io.Writer, logger *slog.Logger) error {
	start := time.Now()
	stats := NewStats(cfg.n, start, uint64(cfg.seed)) //nolint:gosec // a seed, any bits do
	fmt.Fprintf(out, "bots: %d bots (%s...) against %s, server %d, ramp-up %g/s, seed %d\n",
		cfg.n, cfg.prefix, cfg.login, cfg.serverID, cfg.rate, cfg.seed)

	reportDone := make(chan struct{})
	stopReports := make(chan struct{})
	go func() {
		defer close(reportDone)
		t := time.NewTicker(cfg.report)
		defer t.Stop()
		for {
			select {
			case now := <-t.C:
				if err := stats.Report(now, false).Write(out); err != nil {
					logger.Warn("report", "err", err)
				}
			case <-stopReports:
				return
			}
		}
	}()

	var wg sync.WaitGroup
	interval := time.Duration(float64(time.Second) / cfg.rate)
	ramp := time.NewTicker(max(interval, time.Microsecond))
	defer ramp.Stop()
start:
	for i := range cfg.n {
		if i > 0 {
			select {
			case <-ctx.Done():
				break start
			case <-ramp.C:
			}
		}
		b := newBot(i, cfg, stats, logger)
		wg.Go(func() { b.run(ctx) })
	}
	wg.Wait()
	close(stopReports)
	<-reportDone
	return stats.Report(time.Now(), true).Write(out)
}
