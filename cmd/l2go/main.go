// Command l2go runs the L2Go login server or a game server.
//
// Usage:
//
//	l2go login [-config path] [-log-level level]
//	l2go game [-id N] [-config path] [-log-level level]
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
	"syscall"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver"
	"github.com/IPeter/l2go/internal/loginserver"
	"github.com/IPeter/l2go/internal/storage/mongostore"
)

const usage = `Usage:
  l2go login [-config path] [-log-level level]
  l2go game [-id N] [-config path] [-log-level level]

Run "l2go <command> -h" for the flags of a command.
`

// options are the parsed command line.
type options struct {
	command    string // "login" or "game"
	configPath string // empty means the default path
	logLevel   slog.Level
	serverID   int
}

// parseArgs parses the arguments after the program name. Help requests
// return flag.ErrHelp.
func parseArgs(args []string, output io.Writer) (options, error) {
	if len(args) == 0 {
		fmt.Fprint(output, usage)
		return options{}, errors.New("missing command")
	}
	opts := options{command: args[0]}
	switch opts.command {
	case "login", "game":
	case "-h", "-help", "--help", "help":
		fmt.Fprint(output, usage)
		return options{}, flag.ErrHelp
	default:
		fmt.Fprint(output, usage)
		return options{}, fmt.Errorf("unknown command %q", opts.command)
	}

	fs := flag.NewFlagSet("l2go "+opts.command, flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&opts.configPath, "config", "", "config file (default ~/.l2go/config/server.json)")
	level := fs.String("log-level", "info", "log level: debug, info, warn or error")
	if opts.command == "game" {
		fs.IntVar(&opts.serverID, "id", 1, "ID of the game server in the config file")
	}
	if err := fs.Parse(args[1:]); err != nil {
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
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts options, logger *slog.Logger) error {
	cfg, err := loadConfig(opts.configPath, logger)
	if err != nil {
		return err
	}
	switch opts.command {
	case "login":
		return runLogin(ctx, cfg, logger)
	default:
		return runGame(ctx, cfg, opts.serverID, logger)
	}
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

func runGame(ctx context.Context, cfg config.Config, id int, logger *slog.Logger) error {
	srv, err := gameserver.New(cfg, id, logger)
	if err != nil {
		return err
	}
	return srv.Run(ctx)
}
