// Command bots is the load test: it runs N players against a login server
// and its game server (L2Go, or the Mobius reference for comparison).
//
// Each bot logs in with its own account (the prefix and its number, such
// as bot0007; both servers create unknown accounts), creates its character
// when the account has none (a Human Fighter Bot0007, or with -class
// mystic a Human Mystic Bot0007M), enters the world and then acts every few
// seconds (-step). A failed bot tries again after a backoff. Ctrl+C or the
// end of -duration logs every bot out.
//
// The mode (-mode) says what a bot does:
//
//   - wander (the default): it reports its position and either walks to a
//     random point around where it started, says something in general
//     chat, or targets and talks to the nearest NPC. With -stores, that
//     share of the bots (one in ten for 0.1) keep a private sell store of
//     Stems instead; with -buy, an action of the others is, with that
//     probability, a purchase at the nearest store. Both create their
//     Stems and adena with //create_item, so the characters need GM rights.
//   - hunt: it attacks the monsters within -hunt-range of where it
//     started, one after the other, casts its attack skill (-skill; Wind
//     Strike for the default Human Mystic) every -cast, holds -buffs buffs
//     (given with the GM command //buff, so the characters need GM rights),
//     and when it dies it restarts in the village and walks back. With 500
//     hunting bots, the server's "game world ticks" log line (every
//     minute) gives the tick percentiles of the load test (Plan 09, Plan
//     10).
//   - cruma: it hunts like hunt, on the first floor of Cruma Tower (Plan
//     13's load test with geodata): it gets -level (default 40) with
//     //set_level, teleports to one of a few spots there with //teleport,
//     and teleports back after a restart, so the characters need GM rights.
//
// Every -report interval and at the end, bots prints the bots in the
// world, the errors by kind, the game events (kill, death, cast, store, store refused, purchase), the
// packets per second in both directions and the latency percentiles of
// each request and its expected answer: move (MoveToLocation and the
// character's MoveToLocation back), say (Say2 and the CreatureSay echo),
// target (Action and MyTargetSelected), talk (the second Action and
// NpcHtmlMessage, including the walk to the NPC), attack (AttackRequest
// and the first Attack that hits), cast (RequestMagicSkillUse and
// MagicSkillLaunched), buff (the //buff commands and the icons), store (the
// setup window of a store to its message), shop (the second click on a
// store, the walk included, to its window), buy (RequestPrivateStoreBuy and
// the purchase message), restart
// (RequestRestartPoint and the UserInfo after the teleport), kill (from
// choosing a monster to its death), login and enter.
//
// Usage:
//
//	bots [-n 50] [-login 127.0.0.1:2106] [-server 1] [-game addr] [-prefix bot] [-rate 10] [-duration 5m] [-seed N] ...
//	bots -mode hunt [-class mystic|fighter] [-skill id] [-cast 5s] [-buffs 10] [-hunt-range 3000] ...
//	bots -mode cruma [-level 40] [hunt flags] ...
//	bots [-stores 0.1] [-buy 0.2] ...
//	bots -mode hunt -party 3 [-ping 1s] ...   (social.go: parties, trades, friends, the world loop ping)
//	bots [-trade 0.05] [-friend 0.05] [-ping 1s] ...
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
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

	// mode is modeWander or modeHunt, class classFighter or classMystic.
	mode  string
	class string
	// The hunt: the attack skill (0: none) cast every cast (0: never), the
	// number of huntBuffs held, and how far from the start point monsters
	// are hunted.
	skill     int32
	cast      time.Duration
	buffs     int
	huntRange float64
	// level is the level a bot of modeCruma gets with //set_level (0
	// keeps it); resolve sets crumaLevel when -level is not given.
	level int32

	// The private stores of wandering bots: the share of the bots that keep
	// a sell store, and the probability that an action of the others is a
	// purchase at the nearest store.
	stores float64
	buy    float64

	// social is the parties, trades, friend messages and the ping
	// (social.go).
	social socialConfig
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
		skill    int
		charLvl  = -1
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
	fs.StringVar(&cfg.mode, "mode", modeWander, "what the bots do: wander (walk, chat, talk to NPCs), hunt (fight monsters, cast, hold buffs, die and restart) or cruma (hunt in Cruma Tower; needs GM rights)")
	fs.StringVar(&cfg.class, "class", "", "character class: fighter or mystic (default: fighter to wander, mystic to hunt)")
	fs.IntVar(&skill, "skill", 0, "attack skill ID a hunter casts (default: Wind Strike for a mystic, none for a fighter)")
	fs.DurationVar(&cfg.cast, "cast", 5*time.Second, "time between two attack skill casts of a hunter (0: no casts)")
	fs.IntVar(&cfg.buffs, "buffs", 0, fmt.Sprintf("buffs a hunter holds, 0 to %d, given with //buff (needs GM rights)", len(huntBuffs)))
	fs.Float64Var(&cfg.huntRange, "hunt-range", 3000, "how far from its start point a hunter looks for monsters")
	fs.IntVar(&charLvl, "level", -1, fmt.Sprintf("level a cruma bot gets with //set_level, 0 to keep its own (default %d)", crumaLevel))
	fs.Float64Var(&cfg.stores, "stores", 0, "share of the wandering bots that keep a sell store (needs GM rights for //create_item)")
	fs.Float64Var(&cfg.buy, "buy", 0, "probability that an action of a wandering bot is a purchase at the nearest store (needs GM rights for the adena)")
	cfg.social.addFlags(fs)
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if skill < 0 || skill > math.MaxInt32 {
		return config{}, fmt.Errorf("-skill %d is not a skill ID", skill)
	}
	cfg.skill = int32(skill)
	if err := cfg.resolveLevel(charLvl); err != nil {
		return config{}, err
	}
	cfg.resolve()
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
	case c.say < 0 || c.talk < 0 || c.buy < 0 || c.say+c.talk+c.buy > 1:
		return errors.New("-say, -talk and -buy are probabilities with a sum of at most 1")
	case c.stores < 0 || c.stores > 1:
		return errors.New("-stores is a share between 0 and 1")
	case c.talkRange < 0:
		return errors.New("-talk-range must not be negative")
	case c.queue < 1:
		return errors.New("-queue must be at least 1")
	case c.mode != modeWander && c.mode != modeHunt && c.mode != modeCruma:
		return fmt.Errorf("-mode %q: use %s, %s or %s", c.mode, modeWander, modeHunt, modeCruma)
	case c.class != classFighter && c.class != classMystic:
		return fmt.Errorf("-class %q: use %s or %s", c.class, classFighter, classMystic)
	case c.cast < 0:
		return errors.New("-cast must not be negative")
	case c.buffs < 0 || c.buffs > len(huntBuffs):
		return fmt.Errorf("-buffs must be between 0 and %d", len(huntBuffs))
	case c.huntRange <= 0:
		return errors.New("-hunt-range must be positive")
	}
	return c.social.validate(c.mode)
}

// resolve fills in the defaults that depend on other flags: the class (a
// fighter wanders, a mystic hunts) and a mystic's attack skill (Wind
// Strike).
func (c *config) resolve() {
	if c.class == "" {
		c.class = classFighter
		if c.mode == modeHunt || c.mode == modeCruma {
			c.class = classMystic
		}
	}
	if c.skill == 0 && c.class == classMystic {
		c.skill = skillWindStrike
	}
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
	fmt.Fprintf(out, "bots: %d %s bots (%s %s...) against %s, server %d, ramp-up %g/s, seed %d\n",
		cfg.n, cfg.mode, cfg.class, cfg.prefix, cfg.login, cfg.serverID, cfg.rate, cfg.seed)
	if cfg.mode == modeHunt {
		fmt.Fprintf(out, "bots: hunting within %g of the start, skill %d every %v, %d buffs\n",
			cfg.huntRange, cfg.skill, cfg.cast, cfg.buffs)
	}
	if cfg.mode == modeCruma {
		fmt.Fprintf(out, "bots: hunting in Cruma Tower within %g of %d spots, level %d, skill %d every %v, %d buffs\n",
			cfg.huntRange, len(crumaSpots), cfg.level, cfg.skill, cfg.cast, cfg.buffs)
	}
	if cfg.mode == modeWander && (cfg.stores > 0 || cfg.buy > 0) {
		fmt.Fprintf(out, "bots: %g of the bots keep a store, the others buy with probability %g\n", cfg.stores, cfg.buy)
	}
	cfg.social.describe(out)

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
