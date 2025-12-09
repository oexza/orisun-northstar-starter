package orisun

import (
	"context"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/oexza/Orisun/config"
	"github.com/oexza/Orisun/logging"
	"github.com/oexza/Orisun/postgres"
)
import "github.com/oexza/Orisun/orisun"

func SetupOrisun(ctx context.Context, js jetstream.JetStream) error {
	// Load default configuration
	cfg := config.InitializeConfig()

	logger := logging.InitializeDefaultLogger(cfg.Logging)

	// Initialize database connection
	cfg.Postgres.User = "postgres"
	cfg.Postgres.Password = "password@1"
	saveEvents, getEvents, lockProvider, _, eventPublishing := postgres.InitializePostgresDatabase(ctx, cfg.Postgres, cfg.Admin, js, logger)

	// Initialize EventStore
	_ = orisun.InitializeEventStore(
		ctx,
		cfg,
		saveEvents,
		getEvents,
		lockProvider,
		js,
		logger,
	)

	// Start polling events from the event store and publish them to NATS jetstream
	orisun.StartEventPolling(ctx, cfg, lockProvider, getEvents, js, eventPublishing, logger)

	_, err := orisun.NewOrisunServer(
		ctx,
		saveEvents,
		getEvents,
		lockProvider,
		js,
		cfg.GetBoundaryNames(),
		logger,
	)
	if err != nil {
		return err
	}
	return nil
}
