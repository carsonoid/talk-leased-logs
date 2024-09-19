variable "project_id" {
  description = "The project ID to use for the project."

  type        = string
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Project ID must not be empty."
  }
}

variable "org_id" {
  description = "The organization ID to use for the project. Set after manually linking the project to a billing account."

  type        = string
}

variable "billing_account" {
  description = "The billing account ID to associate with the project. Set after manually linking the project to a billing account."

  type        = string
}

provider "google" {
  project = var.project_id
  region  = "us-west3"
}

resource "google_project" "this" {
  name                = var.project_id
  project_id          = var.project_id
  labels              = {}
  billing_account     = "${var.billing_account}"
  org_id              = "${var.org_id}"
}

# enable firestore
resource "google_project_service" "firestore" {
  service = "firestore.googleapis.com"

  timeouts {
    create = "30m"
    update = "40m"
  }

  disable_dependent_services = true
}

resource "google_firestore_database" "default" {
  project     = google_project.this.project_id
  name        = "(default)"
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"

  depends_on = [ google_project_service.firestore ]
}

# set up a job to remove expired leases
# actual deletes may take up to 24 hours past the expiration time
resource "google_firestore_field" "remove_expired_leases" {
  project  = google_project.this.project_id
  database = google_firestore_database.default.name
  collection = "leases"
  field      = "ExpireAt"
  ttl_config {}

  depends_on = [ google_project_service.firestore ]
}

# # Firestore access example
# resource "google_project_iam_member" "weave-ci-runner-user" {
#   role   = "roles/datastore.user"
#   member = "serviceAccount:ACCOUNT@PROJECT.iam.gserviceaccount.com"
# }
