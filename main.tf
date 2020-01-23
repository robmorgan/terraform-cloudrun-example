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
        "client.knative.dev/user-image" = local.image_name

        # uncomment the following line to connect to the cloud sql database instance
        #"run.googleapis.com/cloudsql-instances" = local.instance_connection_name
      }
    }

    spec {
      containers {
        image = local.image_name

        # uncomment the following env vars to provide the cloud run service
        # with the cloud sql database details.
        #env {
        #  name  = "INSTANCE_CONNECTION_NAME"
        #  value = local.instance_connection_name
        #}
        #
        #env {
        #  name  = "MYSQL_DATABASE"
        #  value = var.db_name
        #}
        #
        #env {
        #  name  = "MYSQL_USERNAME"
        #  value = var.db_username
        #}
        #
        #env {
        #  name  = "MYSQL_PASSWORD"
        #  value = var.db_password
        #}
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
# OPTIONALLY DEPLOY A DATABASE
# ---------------------------------------------------------------------------------------------------------------------

resource "google_sql_database_instance" "master" {
  count            = var.deploy_db ? 1 : 0
  name             = var.db_instance_name
  region           = var.location
  database_version = "MYSQL_5_7"

  settings {
    tier = "db-f1-micro"
  }
}

resource "google_sql_database" "default" {
  count = var.deploy_db ? 1 : 0

  name     = var.db_name
  project  = var.project
  instance = google_sql_database_instance.master[0].name

  depends_on = [google_sql_database_instance.master]
}

resource "google_sql_user" "default" {
  count = var.deploy_db ? 1 : 0

  project  = var.project
  name     = var.db_username
  instance = google_sql_database_instance.master[0].name

  host     = var.db_user_host
  password = var.db_password

  depends_on = [google_sql_database.default]
}

# ---------------------------------------------------------------------------------------------------------------------
# PREPARE LOCALS
# ---------------------------------------------------------------------------------------------------------------------

locals {
  image_name = var.image_name == "" ? "${var.gcr_region}.gcr.io/${var.project}/${var.service_name}" : var.image_name
  # uncomment the following line to connect to the cloud sql database instance
  #instance_connection_name = "${var.project}:${var.location}:${google_sql_database_instance.master[0].name}"
}
