# Talk: Leased Logging in GCP

## Demo

The demo requires Go to be install locally and read/write access to Firestore in a GCP project and write access to GCP Cloud Logs.

If you do not have a project, you can create one using the "Project Setup" instructions in this document.
The sample CLI is set up to automatically detect the project id if you follow those instructions.

**If you do not create the sample project** then you will need to export a `PROJECT_ID` env variable or pass the `--project-d`
flag to all the sample commands.

Be sure to build the cli before running the exammple commands:

```bash
go build -o leased-logs .
```

### Capturing output from a command

You can use the `capture` command to have the cli execute a command. It then captures the stdin and stdout and
ships them to GCP Cloud Logs if the lease is currently active.

> It always prints all stdout and stderr to the screen, the only thing that changes based on lease state is whether or not
> logs are shipped to GCP Cloud Logs.

```bash
./leased-logs -l demo1 capture -- bash -c 'while :; do echo "It is currently $(date)"; sleep 1; done'
```

While that runs, it will print the output from the executed command and also include information about the intiial and active leases.

You can extend a lease using the `lease extend` command:

```bash
./leased-logs -l demo1 lease extend "extend lease for demo"
```

You should see the `capture` output print information about the lease being renewed, and then expiring after 5 seconds.

### Integrating with the `slog` package in Go

Another way to play with log leases is to run the `slog-demo` subcommand. This simply outputs sample logs every second.
But the [code](./cmd_slog_demo.go) shows how leased logs could easily be integrated with existing Go code that uses the `slog`
package to handle logs

```bash
./leased-logs -l demo2 slog-demo
```

Just like before, you should see logs regularly printed to screen, along with status messages about the lease state.
You can also use `lease extend` to extend this lease.

```bash
./leased-logs -l demo2 lease extend "extend for slog demo"
```

You can also expire a lease early by using `lease expire`. This will cause the lease to be deleted and immediately stop logs from being shipped

```bash
./leased-logs -l demo2 lease expire
```

## Project Setup

Using Firestore requires a project to be linked to a valid billing account. While firestore has a very
generious free tier, it still requires a linked account before the feature can be activated.

1. Change into the terraform directory
   ```bash
   cd terraform
   ```
2. Ensure you are logged in to the GCP account which you would like to own the project
   ```bash
   gcloud auth application-default login
   ```
3. Write a file to create a unique project id
   ```bash
   echo "project_id = \"talk-leased-logging-$(date +%s)\"" > terraform.tfvars
   ```
4. Apply the project state. Just leave all prompts empty and hit enter to bypass them
   ```bash
   terraform apply -target google_project.this
   ```
5. Log into the [Billing GCP Console](https://console.cloud.google.com/billing/projects) and link your project to your billing account
6. Check for the changed values in the project, again, skip the prompts
   ```bash
   terraform plan
   ```
7. Set the variables for the `org_id` and `billing_account` fields based off the diffs.
   ```bash
   echo 'org_id = "ORGIDVAL"' >> terraform.tfvars
   ```
   ```bash
   echo 'billing_account = "BILLINGACCOUNTVAL"' >> terraform.tfvars
   ```
   > This sort of progressive setting of variables isn't ideal for production usage. But it is the easiest thing to do for a demo project.


### Teardown

You can tear down the project by running `terraform destroy` or by simply deleting the project in the GCP console.
