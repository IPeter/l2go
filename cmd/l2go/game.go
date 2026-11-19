package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage/mongostore"
)

func runGame(ctx context.Context, cfg config.Config, opts options, logger *slog.Logger) error {
	gs, ok := cfg.GameServerByID(opts.serverID)
	if !ok {
		return fmt.Errorf("no game server with id %d in the config", opts.serverID)
	}
	gsLogger := logger.With("server", gs.ID)

	gameData, err := data.Load(gs.DataDir)
	if err != nil {
		return err
	}
	logger.Info("game data loaded", "dir", gs.DataDir,
		"templates", len(gameData.Templates), "items", len(gameData.Items))

	engine, err := scripting.New(ctx, scripting.Options{
		Dir:     gs.Scripts.Dir,
		Timeout: time.Duration(gs.Scripts.EventTimeout),
		Logger:  gsLogger,
	})
	if err != nil {
		return err
	}

	db := gs.Database
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

	loginLink := link.NewClient(link.ClientConfig{
		Addr:       cfg.LoginServer.GameServerAddr(),
		ServerID:   gs.ID,
		Secret:     gs.Secret,
		MaxPlayers: int(gs.Options.MaxPlayers),
	}, gsLogger)

	srv, err := gameserver.New(cfg, opts.serverID, gameserver.Deps{
		Characters: store.Characters(),
		IDs:        store.IDs(),
		Data:       gameData,
		Login:      loginLink,
		Scripts:    engine,
	}, logger)
	if err != nil {
		return err
	}

	// The scripts engine and the login link outlive the server, so
	// connections that are closing still get answers from them. A link
	// error that retrying cannot fix (wrong secret, unknown server ID) stops
	// the server.
	srvCtx, stopServer := context.WithCancel(ctx)
	defer stopServer()
	bgCtx, stopBackground := context.WithCancel(context.WithoutCancel(ctx))
	var wg sync.WaitGroup
	wg.Go(func() { _ = engine.Run(bgCtx) })
	wg.Go(func() { reloadOnSIGHUP(bgCtx, engine, logger) })
	if opts.devReload || gs.Scripts.DevReload {
		wg.Go(func() {
			if err := engine.Watch(bgCtx, scripting.DefaultDebounce); err != nil {
				logger.Error("dev reload is off", "err", err)
			}
		})
	}
	var linkErr error
	wg.Go(func() {
		if err := loginLink.Run(bgCtx); err != nil {
			linkErr = fmt.Errorf("login server link: %w", err)
			stopServer()
		}
	})

	err = srv.Run(srvCtx)
	stopBackground()
	wg.Wait()
	return errors.Join(linkErr, err)
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
