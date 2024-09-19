package main

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"github.com/alecthomas/kong"
	"gopkg.in/ini.v1"
)

var cli struct {
	Debug     bool     `help:"Enable debug mode."`
	ProjectID string   `help:"The ID of the project to work with" env:"PROJECT_ID"`
	LeaseID   string   `help:"The ID of the lease to work with." required:"" env:"LEASE_ID" short:"l"`
	Lease     LeaseCmd `cmd:"" help:"Work with log leasing"`
	Capture   Capture  `cmd:"" help:"Capture logs"`
	SlogDemo  SlogDemo `cmd:"" help:"Run the slog demo"`
}

func main() {
	kctx := kong.Parse(&cli)

	if cli.ProjectID == "" {
		// try to get default project ID from terraform state file
		cli.ProjectID = getProjectIDFromTerraform()
	}

	// initialize Firestore and Logging clients with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// create a GCP cloud logging client using the project ID and default credentials
	logClient, err := logging.NewClient(ctx, cli.ProjectID)
	kctx.FatalIfErrorf(err, "Failed to create logging client")
	defer logClient.Close()

	// create a Firestore client using the project ID and default credentials
	fsClient, err := firestore.NewClient(ctx, cli.ProjectID)
	kctx.FatalIfErrorf(err, "Failed to create firestore client")

	// make a document reference to the lease document
	// this does not fetch the doc but can be used to interact with it later
	docRef := fsClient.Collection("leases").Doc(cli.LeaseID)

	// run sub-commands passing the firestore client, log client, and docRef for use
	err = kctx.Run(fsClient, logClient, docRef)
	kctx.FatalIfErrorf(err)
}

func getProjectIDFromTerraform() string {
	cfg, err := ini.Load("terraform/terraform.tfvars")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		return ""
	}

	projectID := cfg.Section("").Key("project_id").String()
	if projectID == "" {
		fmt.Println("project_id not found in terraform.tfvars")
		return ""
	}

	// fmt.Printf("✔️  Found project_id in terraform.tfvars: %s\n", projectID)

	return projectID
}
