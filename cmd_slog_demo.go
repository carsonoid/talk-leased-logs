package main

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"github.com/carsonoid/talk-leased-logs/internal/lease"
)

type SlogDemo struct {
	InitalLeaseDuration time.Duration `help:"The initial lease Duration." default:"5s"`
	DemoLogInterval     time.Duration `help:"The interval between logs." default:"1s"`
	DemoDuration        time.Duration `help:"The duration of the demo." default:"1m"`
}

func (cmd *SlogDemo) Run(logClient *logging.Client, docRef *firestore.DocumentRef) error {
	ctx := context.Background()

	leaseManager := lease.NewManager(ctx, logClient.Logger("lease-"+cli.LeaseID), time.Now().Add(cmd.InitalLeaseDuration), docRef)

	slog.SetDefault(leaseManager.SlogLogger())

	ctx, cancel := context.WithTimeout(ctx, cmd.DemoDuration)
	defer cancel()

	generateLogs(ctx, cmd.DemoLogInterval)

	return nil
}

func generateLogs(ctx context.Context, dur time.Duration) {
	t := time.NewTicker(dur)
	defer t.Stop()

	for {
		slog.Info("This is an info log.", slog.String("string", "value"))
		slog.Warn("This is a warning log.", slog.Int("int", 42))
		slog.Error("This is an error log.", slog.Float64("float", 3.14))

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
