// Command refdiff runs a client scenario against L2Go and against the L2J
// Mobius reference server (deploy/reference), records both packet
// streams, normalizes them and reports the differences side by side or as
// JSON. It can also save a normalized recording as a golden file and later
// compare L2Go against it without running Mobius.
//
// Usage:
//
//	refdiff [flags] <scenario>
//	refdiff -save file [-side reference|l2go] [flags] <scenario>
//	refdiff -golden file [flags] [scenario]
//	refdiff -sample [-swings n] [flags] <scenario>
//	refdiff -list
//
// Each scenario brings its own ignore patterns for what is noise on any
// server (wandering NPCs, the positions of a fight); -ignore adds to them
// and -default-ignores=false drops them. Scenarios marked GM in -list set
// their characters up with GM commands and need GM rights for the
// generated characters on both servers.
//
// The sample mode runs a swinging scenario on both servers and compares
// the statistics of the swings instead of the packets (package sample).
//
// The exit code is 0 when the streams match, 1 when they differ or a
// scenario failed, and 2 for usage and file errors.
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
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IPeter/l2go/cmd/refdiff/recording"
	"github.com/IPeter/l2go/cmd/refdiff/sample"
	"github.com/IPeter/l2go/cmd/refdiff/scenario"
)

// Exit codes.
const (
	exitSame  = 0
	exitDiff  = 1
	exitUsage = 2
)

// Labels of the two sides.
const (
	sideL2Go      = "l2go"
	sideReference = "reference"
)

const usageText = `Usage:
  refdiff [flags] <scenario>                 compare L2Go with the reference server
  refdiff -save file [flags] <scenario>      record one side into a golden file
  refdiff -golden file [flags] [scenario]    compare L2Go with a golden file
  refdiff -sample [-swings n] <scenario>     compare swing statistics (sample-* scenarios)
  refdiff -list                              list the scenarios

Flags:
`

// options are the parsed command line.
type options struct {
	scenario    string
	list        bool
	l2go, ref   scenario.Target
	stepTimeout time.Duration
	ignore      []string
	json        bool
	full        bool
	save        string
	side        string
	golden      string
	logLevel    slog.Level

	// defaultIgnores keeps the scenario's own ignore patterns; sample and
	// swings are the sample mode.
	defaultIgnores bool
	sample         bool
	swings         int
}

// parseArgs parses the arguments after the program name. A help request
// returns flag.ErrHelp.
func parseArgs(args []string, output io.Writer) (options, error) {
	var (
		opts                  options
		l2goServer, refServer int
		l2goLogin, refLogin   string
		l2goGame, refGame     string
		password, level       string
	)
	fs := flag.NewFlagSet("refdiff", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.Usage = func() {
		fmt.Fprint(output, usageText)
		fs.PrintDefaults()
	}
	fs.StringVar(&l2goLogin, "l2go-login", "127.0.0.1:2106", "L2Go login server address")
	fs.IntVar(&l2goServer, "l2go-server", 1, "L2Go game server ID")
	fs.StringVar(&l2goGame, "l2go-game", "", "L2Go game server address (default: from the server list)")
	fs.StringVar(&refLogin, "ref-login", "127.0.0.1:12106", "reference (Mobius) login server address")
	fs.IntVar(&refServer, "ref-server", 2, "reference game server ID")
	fs.StringVar(&refGame, "ref-game", "", "reference game server address (default: from the server list)")
	fs.StringVar(&password, "password", "refdiff1", "password of the generated accounts")
	fs.DurationVar(&opts.stepTimeout, "step-timeout", scenario.DefaultStepTimeout, "time limit of each scenario step")
	fs.Func("ignore", "comma-separated packets or Packet.Field globs to leave out, such as NpcInfo or *.Heading (repeatable)", func(s string) error {
		for p := range strings.SplitSeq(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				opts.ignore = append(opts.ignore, p)
			}
		}
		return nil
	})
	fs.BoolVar(&opts.defaultIgnores, "default-ignores", true, "apply the scenario's own ignore patterns (refdiff -list -full shows them)")
	fs.BoolVar(&opts.sample, "sample", false, "compare the statistics of the swings instead of the packets")
	fs.IntVar(&opts.swings, "swings", scenario.DefaultSwings, "swings per side in -sample mode")
	fs.BoolVar(&opts.json, "json", false, "print the comparison as JSON")
	fs.BoolVar(&opts.full, "full", false, "print every packet, not only the differences and their context")
	fs.StringVar(&opts.save, "save", "", "record the scenario on one side (-side) and save the normalized recording to this file")
	fs.StringVar(&opts.side, "side", sideReference, "the side -save records: reference or l2go")
	fs.StringVar(&opts.golden, "golden", "", "compare L2Go with this saved recording instead of the live reference server")
	fs.BoolVar(&opts.list, "list", false, "list the scenarios and exit")
	fs.StringVar(&level, "log-level", "warn", "log level: debug, info, warn or error")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if err := opts.logLevel.UnmarshalText([]byte(level)); err != nil {
		return options{}, fmt.Errorf("invalid -log-level %q", level)
	}
	if opts.list {
		return opts, nil
	}
	switch fs.NArg() {
	case 0:
		if opts.golden == "" {
			fs.Usage()
			return options{}, errors.New("missing scenario name (refdiff -list shows them)")
		}
	case 1:
		opts.scenario = fs.Arg(0)
	default:
		return options{}, fmt.Errorf("unexpected arguments: %v", fs.Args()[1:])
	}
	if opts.save != "" && opts.golden != "" {
		return options{}, errors.New("-save and -golden do not go together")
	}
	if opts.sample && (opts.save != "" || opts.golden != "") {
		return options{}, errors.New("-sample does not go with -save or -golden")
	}
	if opts.swings <= 0 {
		return options{}, errors.New("-swings must be positive")
	}
	if opts.side != sideReference && opts.side != sideL2Go {
		return options{}, fmt.Errorf("invalid -side %q, want %s or %s", opts.side, sideReference, sideL2Go)
	}
	if opts.stepTimeout <= 0 {
		return options{}, errors.New("-step-timeout must be positive")
	}
	if password == "" || len(password) > 16 {
		return options{}, errors.New("-password must have 1 to 16 characters")
	}
	l2goID, err := serverID("l2go-server", l2goServer)
	if err != nil {
		return options{}, err
	}
	refID, err := serverID("ref-server", refServer)
	if err != nil {
		return options{}, err
	}
	opts.l2go = scenario.Target{Label: sideL2Go, LoginAddr: l2goLogin, ServerID: l2goID, GameAddr: l2goGame, Password: password}
	opts.ref = scenario.Target{Label: sideReference, LoginAddr: refLogin, ServerID: refID, GameAddr: refGame, Password: password}
	return opts, nil
}

// serverID checks a game server ID flag.
func serverID(name string, v int) (byte, error) {
	if v < 1 || v > 255 {
		return 0, fmt.Errorf("-%s %d is not a game server ID (1-255)", name, v)
	}
	return byte(v), nil
}

func main() {
	opts, err := parseArgs(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "refdiff:", err)
		os.Exit(exitUsage)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: opts.logLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, opts, os.Stdout, os.Stderr, logger)
	stop()
	os.Exit(code)
}

// run does what opts say and returns the exit code.
func run(ctx context.Context, opts options, stdout, stderr io.Writer, logger *slog.Logger) int {
	if opts.list {
		listScenarios(stdout, opts.full)
		return exitSame
	}
	fail := func(format string, args ...any) int {
		fmt.Fprintf(stderr, "refdiff: "+format+"\n", args...)
		return exitUsage
	}

	var golden recording.Recording
	if opts.golden != "" {
		g, err := recording.Load(opts.golden)
		if err != nil {
			return fail("%v", err)
		}
		if opts.scenario == "" {
			opts.scenario = g.Scenario
		}
		if g.Scenario != opts.scenario {
			return fail("%s is a recording of scenario %q, not %q", opts.golden, g.Scenario, opts.scenario)
		}
		golden = g
	}
	sc, ok := scenario.Find(opts.scenario)
	if !ok {
		return fail("unknown scenario %q (refdiff -list shows them)", opts.scenario)
	}
	sopts := scenario.Options{StepTimeout: opts.stepTimeout, Logger: logger, Swings: opts.swings}

	if opts.sample {
		return runSample(ctx, opts, sc, sopts, stdout, stderr)
	}

	if opts.save != "" {
		t := opts.ref
		if opts.side == sideL2Go {
			t = opts.l2go
		}
		capt := scenario.Run(ctx, sc, t, sopts)
		rec := recording.Normalize(capt)
		if err := recording.Save(opts.save, rec); err != nil {
			return fail("%v", err)
		}
		fmt.Fprintf(stdout, "saved %s: scenario %s on %s, %d streams, %d server packets\n",
			opts.save, sc.Name, t.Label, len(rec.Streams), rec.Packets())
		if capt.Err != nil {
			fmt.Fprintf(stderr, "refdiff: the scenario failed on %s, the recording is partial: %v\n", t.Label, capt.Err)
			return exitDiff
		}
		return exitSame
	}

	var left, right recording.Recording
	if opts.golden != "" {
		left = recording.Normalize(scenario.Run(ctx, sc, opts.l2go, sopts))
		right = golden
		right.Source = fmt.Sprintf("golden file %s, recorded with %s", opts.golden, golden.Source)
	} else {
		var wg sync.WaitGroup
		wg.Go(func() { left = recording.Normalize(scenario.Run(ctx, sc, opts.l2go, sopts)) })
		wg.Go(func() { right = recording.Normalize(scenario.Run(ctx, sc, opts.ref, sopts)) })
		wg.Wait()
	}

	res := recording.Diff(left, right, recording.Options{Ignore: ignorePatterns(sc, opts)})
	var err error
	if opts.json {
		err = recording.WriteJSON(stdout, res)
	} else {
		err = recording.WriteText(stdout, res, recording.TextOptions{Full: opts.full})
	}
	if err != nil {
		return fail("write the report: %v", err)
	}
	if !res.Identical() {
		return exitDiff
	}
	return exitSame
}

// ignorePatterns returns the scenario's own ignore patterns (unless
// -default-ignores=false) followed by the -ignore ones.
func ignorePatterns(sc scenario.Scenario, opts options) []string {
	if !opts.defaultIgnores {
		return slices.Clone(opts.ignore)
	}
	return slices.Concat(sc.Ignore, opts.ignore)
}

// listScenarios prints the scenarios; full adds their ignore patterns.
func listScenarios(w io.Writer, full bool) {
	for _, sc := range scenario.All() {
		var tags []string
		if sc.GM {
			tags = append(tags, "GM")
		}
		if sc.Sample {
			tags = append(tags, "-sample")
		}
		desc := sc.Description
		if len(tags) > 0 {
			desc = "[" + strings.Join(tags, ", ") + "] " + desc
		}
		fmt.Fprintf(w, "%-22s %s\n", sc.Name, desc)
		if full && len(sc.Ignore) > 0 {
			fmt.Fprintf(w, "%-22s ignores %s\n", "", strings.Join(sc.Ignore, ","))
		}
	}
}

// runSample runs sc on both servers and compares the swings of its first
// role.
func runSample(ctx context.Context, opts options, sc scenario.Scenario, sopts scenario.Options, stdout, stderr io.Writer) int {
	if !sc.Sample {
		fmt.Fprintf(stderr, "refdiff: %s is not a swinging scenario; -sample still compares its attacks\n", sc.Name)
	}
	var left, right *scenario.Capture
	var wg sync.WaitGroup
	wg.Go(func() { left = scenario.Run(ctx, sc, opts.l2go, sopts) })
	wg.Go(func() { right = scenario.Run(ctx, sc, opts.ref, sopts) })
	wg.Wait()

	code := exitSame
	role := sc.Roles[0]
	var stats [2]sample.Stats
	for i, c := range []*scenario.Capture{left, right} {
		if c.Err != nil {
			fmt.Fprintf(stderr, "refdiff: the scenario failed on %s: %v\n", c.Target.Label, c.Err)
			code = exitDiff
		}
		st, err := sample.Collect(c, role)
		if err != nil {
			fmt.Fprintf(stderr, "refdiff: %v\n", err)
			code = exitDiff
		}
		stats[i] = st
	}
	rep := sample.Compare(stats[0], stats[1], opts.swings)
	if err := sample.WriteText(stdout, sc.Name, rep); err != nil {
		fmt.Fprintf(stderr, "refdiff: write the report: %v\n", err)
		return exitUsage
	}
	if !rep.OK() {
		code = exitDiff
	}
	return code
}
