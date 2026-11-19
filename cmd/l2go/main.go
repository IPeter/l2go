// Command l2go runs the L2Go login server or a game server, or checks the
// Lua scripts.
//
// Usage:
//
//	l2go login [-config path] [-log-level level]
//	l2go game [-id N] [-config path] [-log-level level] [-dev-reload]
//	l2go scripts check [-dir path] [-log-level level]
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

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver"
	"github.com/IPeter/l2go/internal/loginserver"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage/mongostore"
)

const usage = `Usage:
  l2go login [-config path] [-log-level level]
  l2go game [-id N] [-config path] [-log-level level] [-dev-reload]
  l2go scripts check [-dir path] [-log-level level]

Run "l2go <command> -h" for the flags of a command.
`

// options are the parsed command line.
type options struct {
	command    string // "login", "game" or "scripts" (scripts check)
	configPath string // empty means the default path
	logLevel   slog.Level
	serverID   int
	devReload  bool   // game: reload scripts when a file changes
	scriptsDir string // scripts check: the folder to check
}

func isHelp(arg string) bool {
	switch arg {
	case "-h", "-help", "--help", "help":
		return true
	}
	return false
}

// parseArgs parses the arguments after the program name. Help requests
// return flag.ErrHelp.
func parseArgs(args []string, output io.Writer) (options, error) {
	if len(args) == 0 {
		fmt.Fprint(output, usage)
		return options{}, errors.New("missing command")
	}
	opts := options{command: args[0]}
	name, rest := "l2go "+opts.command, args[1:]
	switch {
	case opts.command == "login", opts.command == "game":
	case opts.command == "scripts":
		if len(rest) > 0 && isHelp(rest[0]) {
			fmt.Fprint(output, usage)
			return options{}, flag.ErrHelp
		}
		if len(rest) == 0 || rest[0] != "check" {
			fmt.Fprint(output, usage)
			return options{}, errors.New(`unknown or missing scripts command (want "scripts check")`)
		}
		name, rest = "l2go scripts check", rest[1:]
	case isHelp(opts.command):
		fmt.Fprint(output, usage)
		return options{}, flag.ErrHelp
	default:
		fmt.Fprint(output, usage)
		return options{}, fmt.Errorf("unknown command %q", opts.command)
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(output)
	level := fs.String("log-level", "info", "log level: debug, info, warn or error")
	switch opts.command {
	case "scripts":
		fs.StringVar(&opts.scriptsDir, "dir", scripting.DefaultDir, "scripts folder to check")
	case "game":
		fs.IntVar(&opts.serverID, "id", 1, "ID of the game server in the config file")
		fs.BoolVar(&opts.devReload, "dev-reload", false, "reload the Lua scripts when a file changes (also scripts.devReload in the config)")
		fallthrough
	default:
		fs.StringVar(&opts.configPath, "config", "", "config file (default ~/.l2go/config/server.json)")
	}
	if err := fs.Parse(rest); err != nil {
		return options{}, err
	}
	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if err := opts.logLevel.UnmarshalText([]byte(*level)); err != nil {
		return options{}, fmt.Errorf("invalid -log-level %q", *level)
	}
	return opts, nil
}

func main() {
	opts, err := parseArgs(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "l2go:", err)
		os.Exit(2)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: opts.logLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err = run(ctx, opts, logger)
	stop()
	if err != nil {
		if !errors.Is(err, errReported) {
			logger.Error("fatal", "err", err)
		}
		os.Exit(1)
	}
}

// errReported is returned when the problem has already been printed.
var errReported = errors.New("already reported")

func run(ctx context.Context, opts options, logger *slog.Logger) error {
	if opts.command == "scripts" {
		return checkScripts(ctx, opts.scriptsDir, os.Stdout, os.Stderr, logger)
	}
	cfg, err := loadConfig(opts.configPath, logger)
	if err != nil {
		return err
	}
	switch opts.command {
	case "login":
		return runLogin(ctx, cfg, logger)
	default:
		return runGame(ctx, cfg, opts, logger)
	}
}

// checkScripts loads every script in dir in the sandbox, without starting
// a server. Problems go to stderr, one per line as file:line: message, and
// make it return errReported.
func checkScripts(ctx context.Context, dir string, stdout, stderr io.Writer, logger *slog.Logger) error {
	n, err := scripting.Check(ctx, scripting.Options{Dir: dir, Logger: logger})
	if le, ok := errors.AsType[*scripting.LoadError](err); ok {
		for _, e := range le.Errs {
			fmt.Fprintln(stderr, e)
		}
		fmt.Fprintf(stderr, "%s: %d problem(s)\n", dir, len(le.Errs))
		return errReported
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: %d Lua files OK\n", dir, n)
	return nil
}

func loadConfig(path string, logger *slog.Logger) (config.Config, error) {
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return config.Config{}, err
		}
		path = p
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, err
	}
	if cfg.Path == "" {
		logger.Warn("config file not found, using built-in defaults", "path", path)
	} else {
		logger.Info("config loaded", "path", cfg.Path)
	}
	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func runLogin(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	db := cfg.LoginServer.Database
	store, err := mongostore.Open(ctx, db.MongoURI(), db.Name)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			logger.Warn("closing the database", "err", err)
		}
	}()
	logger.Info("connected to the database", "name", db.Name)

	srv, err := loginserver.New(cfg, store.Accounts(), logger)
	if err != nil {
		return err
	}
	return srv.Run(ctx)
}

func runGame(ctx context.Context, cfg config.Config, opts options, logger *slog.Logger) error {
	gs, ok := cfg.GameServerByID(opts.serverID)
	if !ok {
		return fmt.Errorf("no game server with id %d in the config", opts.serverID)
	}
	engine, err := scripting.New(ctx, scripting.Options{
		Dir:     gs.Scripts.Dir,
		Timeout: time.Duration(gs.Scripts.EventTimeout),
		Logger:  logger.With("server", gs.ID),
	})
	if err != nil {
		return err
	}
	srv, err := gameserver.New(cfg, opts.serverID, engine, logger)
	if err != nil {
		return err
	}

	// The engine outlives the server, so connections that are closing still
	// get answers from the scripts.
	engineCtx, stopEngine := context.WithCancel(context.WithoutCancel(ctx))
	var wg sync.WaitGroup
	wg.Go(func() { _ = engine.Run(engineCtx) })
	wg.Go(func() { reloadOnSIGHUP(engineCtx, engine, logger) })
	if opts.devReload || gs.Scripts.DevReload {
		wg.Go(func() {
			if err := engine.Watch(engineCtx, scripting.DefaultDebounce); err != nil {
				logger.Error("dev reload is off", "err", err)
			}
		})
	}

	err = srv.Run(ctx)
	stopEngine()
	wg.Wait()
	return err
}

// reloadOnSIGHUP reloads the scripts on every SIGHUP until ctx is done
// (docker compose kill -s HUP game). A failed reload keeps the old scripts.
func reloadOnSIGHUP(ctx context.Context, engine *scripting.Engine, logger *slog.Logger) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			logger.Info("SIGHUP: reloading scripts")
			if err := engine.Reload(ctx); err != nil && ctx.Err() == nil {
				logger.Error("script reload failed, keeping the old scripts", "err", err)
			}
		}
	}
}
