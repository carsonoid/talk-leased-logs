package main

import (
	"context"
	"os/exec"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"

	"github.com/carsonoid/talk-leased-logs/internal/lease"
)

type Capture struct {
	InitalLeaseDuration time.Duration `help:"The initial lease time." default:"5s"`
	Args                []string      `arg:"" optional:""`
}

func (cmd *Capture) Run(logClient *logging.Client, docRef *firestore.DocumentRef) error {
	ctx := context.Background()

	leaseManager := lease.NewManager(ctx, logClient.Logger("lease-"+cli.LeaseID), time.Now().Add(cmd.InitalLeaseDuration), docRef)

	execCmd := exec.Command(cmd.Args[0], cmd.Args[1:]...)
	execCmd.Stdout = leaseManager.StdoutWriter()
	execCmd.Stderr = leaseManager.StderrWriter()

	return execCmd.Run()
}
