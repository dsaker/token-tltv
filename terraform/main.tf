resource "google_artifact_registry_repository" "token_tltv" {
  location      = var.region
  repository_id = var.repository_id
  description   = "docker repository for token-tltv project"
  format        = "DOCKER"

}

# create local data to store registry info for image name
locals {
  l = data.google_artifact_registry_repository.token_tltv.location
  p = data.google_artifact_registry_repository.token_tltv.project
  r = data.google_artifact_registry_repository.token_tltv.repository_id
  image = "${local.l}-docker.pkg.dev/${local.p}/${local.r}/${var.image_name}:${var.image_version}"
}

data "google_artifact_registry_repository" "token_tltv" {
  location      = var.region
  repository_id = var.repository_id
}

# cloud run job to run container
resource "google_cloud_run_v2_service" "token-tltv" {
  name     = "token-tltv-cloudrun-service"
  location = var.region
  deletion_protection = false

  template {
    containers {
      image = local.image
      resources {
        limits = {
          cpu    = "1000m",
          memory = "1024Mi"
        }
      }
    }
  }
}

# service account to run cloud run job
resource "google_service_account" "sa-name" {
  account_id = "cloud-run-invoker"
}

resource "google_project_iam_member" "cloud-run-builder" {
  project = var.project_id
  role    = "roles/run.builder"
  member  = "serviceAccount:${google_service_account.sa-name.email}"
}

resource "google_project_iam_member" "cloud-run-invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.sa-name.email}"
}

# alerting policy to alert when errors occur in the token-tltv project
resource "google_monitoring_alert_policy" "cloud_run_service_alert" {
  combiner              = "OR"
  display_name          = "cloud_run_service_error"
  enabled               = true
  # add notification channels
  notification_channels = [google_monitoring_notification_channel.sms_notification.id, google_monitoring_notification_channel.email_notification.id]
  project               = var.project_id
  severity              = "ERROR"
  user_labels           = {}
  alert_strategy {
    auto_close           = "604800s"
    notification_prompts = ["OPENED"]
    notification_rate_limit {
      period = "36000s" # 10 hours
    }
  }
  conditions {
    display_name = "Log match condition"
    condition_matched_log {
      filter           = "resource.type=\"cloud_run_job\"\nseverity>=ERROR"
      label_extractors = {}
    }
  }
}

resource "google_monitoring_notification_channel" "sms_notification" {
  description  = null
  display_name = "Phone SMS Notification"
  enabled      = true
  force_delete = false
  labels = {
    number = var.sms_notification
  }
  project     = var.project_id
  type        = "sms"

user_labels = {}
}

resource "google_monitoring_notification_channel" "email_notification" {
  description  = null
  display_name = "Email Notification"
  enabled      = true
  force_delete = false
  labels = {
    email_address = var.email_notification
  }
  project     = var.project_id
  type        = "email"
  user_labels = {}
}