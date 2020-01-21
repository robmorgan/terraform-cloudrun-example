terraform {
  # The modules used in this example have been updated with 0.12 syntax, additionally we depend on a bug fixed in
  # version 0.12.7.
  required_version = ">= 0.12.7"

  required_providers {
    google = ">= 3.4"
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# DEPLOY A GOOGLE CLOUD SOURCE REPOSITORY
# ---------------------------------------------------------------------------------------------------------------------

resource "google_sourcerepo_repository" "repo" {
  name = var.repository_name
}

# ---------------------------------------------------------------------------------------------------------------------
# DEPLOY A CLOUD RUN SERVICE
# ---------------------------------------------------------------------------------------------------------------------

resource "google_cloud_run_service" "service" {
  name     = var.service_name
  location = var.location

  template {
    metadata {
      annotations = {
        "client.knative.dev/user-image" = var.image_name
      }
    }

    spec {
      containers {
        image = local.image_name
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# EXPOSE THE SERVICE PUBLICALLY
# We give all users the ability to invoke the service.
# ---------------------------------------------------------------------------------------------------------------------

resource "google_cloud_run_service_iam_member" "allUsers" {
  service  = google_cloud_run_service.service.name
  location = google_cloud_run_service.service.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# ---------------------------------------------------------------------------------------------------------------------
# CREATE A CLOUD BUILD TRIGGER
# ---------------------------------------------------------------------------------------------------------------------

resource "google_cloudbuild_trigger" "cloud_build_trigger" {
  description = "Cloud Source Repository Trigger ${var.repository_name} (${var.branch_name})"

  trigger_template {
    branch_name = var.branch_name
    repo_name   = var.repository_name
  }

  # These substitutions have been defined in the sample app's cloudbuild.yaml file.
  # See: https://github.com/robmorgan/sample-docker-app/blob/master/cloudbuild.yaml#L43
  substitutions = {
    _LOCATION     = var.location
    _GCR_REGION   = var.gcr_region
    _SERVICE_NAME = var.service_name
  }

  # The filename argument instructs Cloud Build to look for a file in the root of the repository.
  # Either a filename or build template (below) must be provided.
  filename = "cloudbuild.yaml"

  depends_on = [google_sourcerepo_repository.repo]
}

# ---------------------------------------------------------------------------------------------------------------------
# CONFIGURE THE GCR REGISTRY TO STORE THE CLOUD BUILD ARTIFACTS
# ---------------------------------------------------------------------------------------------------------------------

module "gcr_registry" {
  source = "github.com/gruntwork-io/terraform-google-ci.git//modules/gcr-registry?ref=upgrade-google-providers"

  project         = var.project
  registry_region = var.gcr_region
}

# ---------------------------------------------------------------------------------------------------------------------
# PREPARE LOCALS
# ---------------------------------------------------------------------------------------------------------------------

locals {
  image_name = var.image_name == "" ? "${var.gcr_region}.gcr.io/${var.project}/${var.service_name}" : var.image_name
}
