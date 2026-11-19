package main

import (
	"context"
	"log/slog"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/loginserver"
	"github.com/IPeter/l2go/internal/storage/mongostore"
)

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
