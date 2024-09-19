package main

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/carsonoid/talk-leased-logs/internal/lease"
)

type LeaseExtendCmd struct {
	Duration time.Duration `help:"The duration of the lease." default:"5s"`
	User     string        `help:"The user extending the lease."`
	Reason   string        `help:"The reason for extending the lease." arg:""`
}

func (cmd *LeaseExtendCmd) Run(docRef *firestore.DocumentRef) error {
	ctx := context.Background()

	expireAt := time.Now().UTC().Add(cmd.Duration)

	_, err := docRef.Set(ctx, lease.Document{
		ExpireAt: expireAt,
		User:     cmd.User,
		Reason:   cmd.Reason,
	})
	if err != nil {
		return fmt.Errorf("Failed to set lease: %w", err)
	}

	fmt.Printf("Updated Lease %q\n", docRef.Path)
	fmt.Printf("  Expires: %s (in %s)\n", expireAt, cmd.Duration)
	if cmd.User != "" {
		fmt.Printf("  User: %q\n", cmd.User)
	}
	if cmd.Reason != "" {
		fmt.Printf("  Reason: %q\n", cmd.Reason)
	}

	return nil
}

type LeaseExpire struct {
}

func (cmd *LeaseExpire) Run(docRef *firestore.DocumentRef) error {
	ctx := context.Background()

	_, err := docRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete lease: %w", err)
	}

	fmt.Printf("Lease at %q deleted\n", docRef.Path)

	return nil
}

type LeaseCmd struct {
	Extend LeaseExtendCmd `cmd:"extend" help:"Extend a lease for a time."`
	Expire LeaseExpire    `cmd:"expire" help:"Expire a lease immediately."`
}
