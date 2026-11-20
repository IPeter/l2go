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
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/mongostore"
)

// runGame runs one game server: the world loop, the client listener and
// the link to the login server.
//
// Shutting down goes in this order: the listener and every connection
// close (each player in the world is reported to the game) and the script
// reloaders stop, then the world loop saves every player and stops, then
// the login link stops and the scripts engine closes.
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
	worldData, err := data.LoadWorld(gs.DataDir)
	if err != nil {
		return err
	}
	logger.Info("world data loaded", "dir", gs.DataDir, "npcs", len(worldData.Npcs),
		"spawns", len(worldData.Spawns), "zones", len(worldData.Zones))
	// The weight penalty (skill 4270) and the class categories of item
	// conditions.
	passives, err := data.LoadPassives(gs.DataDir)
	if err != nil {
		return err
	}
	categories, err := data.LoadCategories(gs.DataDir)
	if err != nil {
		return err
	}
	// Buy lists, multisell lists and merchant price configs.
	economy, err := data.LoadEconomy(gs.DataDir, data.NewEconomyRefs(gameData, worldData))
	if err != nil {
		return err
	}
	// Skills, skill trees and armor sets.
	skillData, err := data.LoadSkills(gs.DataDir)
	if err != nil {
		return err
	}
	// Doors, static objects (town maps, signboards) and fences.
	objects, err := data.LoadObjects(gs.DataDir)
	if err != nil {
		return err
	}
	// Raid boss spawns, grand boss seeds and boss zone parameters.
	bosses, err := data.LoadBosses(gs.DataDir)
	if err != nil {
		return err
	}
	if err := bosses.CheckWorld(worldData); err != nil {
		return err
	}

	// Geodata (ground heights, walls) from the L2J tiles in geodata.dir. An
	// empty setting, or a folder where no tile loads, runs without geodata
	// and pathfinding, as Mobius does without its files; Open logs what it
	// loaded and every tile it skipped.
	geoOpts, err := geoOptions(gs.Geodata)
	if err != nil {
		return err
	}
	geoEngine, err := geo.Open(gs.Geodata.Dir, geoOpts, logger)
	if err != nil {
		return err
	}
	if _, none := geoEngine.(geo.None); none && gs.Geodata.Dir == "" {
		logger.Info("no geodata (geodata.dir is empty): heights come from the clients and the spawns")
	}

	// The scripts' constant tables (ClassId, SystemMessageId, ...).
	constants, err := data.LoadConstants(gs.DataDir)
	if err != nil {
		return err
	}
	// Mobius's quest defaults until the quest config keys exist.
	quests := game.QuestSettings{}

	// The scripts run on the world loop. Their output (npc:say and the
	// like) and the quests' world (quest states, items, timers) reach the
	// players through the game's adapter.
	out := game.NewScriptOutput()
	engine, err := scripting.New(ctx, scripting.Options{
		Dir:       gs.Scripts.Dir,
		Timeout:   time.Duration(gs.Scripts.EventTimeout),
		Logger:    gsLogger,
		Output:    out,
		World:     out,
		Constants: game.ScriptConstants(constants, gs.Combat.PartyRange, quests),
		Rates:     scripting.Rates{QuestItemDropAmount: quests.ItemDropAmount},
	})
	if err != nil {
		return err
	}
	// Every return after the world loop starts waits for it to stop first,
	// so the engine closes only when the loop no longer calls it.
	defer engine.Close()

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

	// New items take their object IDs from a pool on the world loop, which
	// refills itself in the background (decision I3).
	idPool := storage.NewIDPool(store.IDs(), storage.IDPoolOptions{Logger: gsLogger})
	if err := idPool.Fill(ctx); err != nil {
		return err
	}
	defer idPool.Close()

	loginLink := link.NewClient(link.ClientConfig{
		Addr:       cfg.LoginServer.GameServerAddr(),
		ServerID:   gs.ID,
		Secret:     gs.Secret,
		MaxPlayers: int(gs.Options.MaxPlayers),
	}, gsLogger)

	world, err := game.New(game.Options{
		Config:          cfg,
		ServerID:        gs.ID,
		Data:            gameData,
		World:           worldData,
		Characters:      store.Characters(),
		Relations:       store.Relations(), // friend and block lists
		Clans:           store.Clans(),     // the clan registry, loaded before the loop starts
		IDs:             store.IDs(),
		IDPool:          idPool,
		Passives:        passives,
		Categories:      categories,
		Economy:         economy,
		Skills:          skillData,
		Scripts:         engine,
		ScriptOutput:    out,
		GlobalVariables: store.GlobalVariables(),
		Quests:          quests,
		Geo:             geoEngine,
		Objects:         objects, // doors, static objects, fences
		Bosses:          bosses,
		BossStore:       store.Bosses(), // raid and grand boss state, raid points
		Accounts:        loginLink,      // account bans of the admin panel
		Logger:          gsLogger,
	})
	if err != nil {
		return err
	}

	srv, err := gameserver.New(cfg, opts.serverID, gameserver.Deps{
		Characters: store.Characters(),
		IDs:        store.IDs(),
		Data:       gameData,
		Login:      loginLink,
		Game:       world,
	}, logger)
	if err != nil {
		return err
	}

	// srvCtx ends the listener: on a signal, when the world loop stops, or
	// on a link error that retrying cannot fix (wrong secret, unknown
	// server ID). The loop and the link have their own contexts, ended in
	// order below, so they outlive the connections.
	srvCtx, stopServer := context.WithCancel(ctx)
	defer stopServer()
	loopCtx, stopLoop := context.WithCancel(context.WithoutCancel(ctx))
	defer stopLoop()
	linkCtx, stopLink := context.WithCancel(context.WithoutCancel(ctx))
	defer stopLink()
	reloadCtx, stopReload := context.WithCancel(ctx)
	defer stopReload()

	var loopErr error
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		if err := world.Run(loopCtx); err != nil {
			loopErr = fmt.Errorf("world loop: %w", err)
		}
		stopServer() // without the loop there is no game
	}()

	var linkErr error
	var linkWG sync.WaitGroup
	linkWG.Go(func() {
		if err := loginLink.Run(linkCtx); err != nil {
			linkErr = fmt.Errorf("login server link: %w", err)
			stopServer()
		}
	})

	// Reloading prepares the scripts off the loop; SwapScripts puts them in
	// place on the loop.
	var reloadWG sync.WaitGroup
	reloadWG.Go(func() { reloadOnSIGHUP(reloadCtx, engine, world.SwapScripts, logger) })
	if opts.devReload || gs.Scripts.DevReload {
		reloadWG.Go(func() {
			if err := engine.Watch(reloadCtx, scripting.DefaultDebounce, world.SwapScripts); err != nil {
				logger.Error("dev reload is off", "err", err)
			}
		})
	}

	err = srv.Run(srvCtx)
	stopReload()
	reloadWG.Wait()
	stopLoop()
	<-loopDone
	stopLink()
	linkWG.Wait()
	return errors.Join(linkErr, loopErr, err)
}

// reloadOnSIGHUP prepares the scripts again on every SIGHUP until ctx is
// done (docker compose kill -s HUP game) and hands them to swap, which puts
// them in place on the world loop. A failed load keeps the old scripts.
func reloadOnSIGHUP(ctx context.Context, engine *scripting.Engine, swap func(*scripting.Prepared), logger *slog.Logger) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	for {
		select {
		case <-ctx.Done():
			return
		case <-hup:
			logger.Info("SIGHUP: reloading scripts")
			p, err := engine.Prepare(ctx)
			if err != nil {
				if ctx.Err() == nil {
					logger.Error("script reload failed, keeping the old scripts", "err", err)
				}
				continue
			}
			swap(p)
		}
	}
}
