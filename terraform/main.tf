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
  name                 = "token-tltv-cloudrun-service"
  ingress              = "INGRESS_TRAFFIC_ALL"
  project              = var.project_id
  location             = var.region

  template {
    service_account = google_service_account.tltv_cloudrun_service_identity.email
    session_affinity                 = false
    timeout                          = "300s"
    containers {
      image       = local.image
      resources {
        cpu_idle = true
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
        startup_cpu_boost = true
      }
      startup_probe {
        failure_threshold     = 1
        initial_delay_seconds = 0
        period_seconds        = 240
        timeout_seconds       = 240
        tcp_socket {
          port = 8080
        }
      }
    }
    scaling {
      max_instance_count = 2
      min_instance_count = 0
    }
  }
  traffic {
    percent  = 100
    type     = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# allow unauthenticated access to access service url (users from the internet)
resource "google_cloud_run_service_iam_binding" "all-users" {
  location = var.region
  service  = google_cloud_run_v2_service.token-tltv.name
  role     = "roles/run.invoker"
  members = [
    "allUsers"
  ]
}

# alerting policy to alert when errors occur in the token-tltv-cloudrun-service
resource "google_monitoring_alert_policy" "cloud_run_service_error" {
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
      period = "900s" # 15 minutes
    }
  }
  conditions {
    display_name = "Log match condition"
    condition_matched_log {
      filter           = "severity=ERROR\nresource.labels.service_name=\"token-tltv-cloudrun-service\"\nresource.type=\"cloud_run_revision\""
    }
  }
}

# alerting policy to alert when warnings occur in the token-tltv-cloudrun-service
resource "google_monitoring_alert_policy" "cloud_run_service_warning" {
  combiner              = "OR"
  display_name          = "cloud_run_service_warning"
  enabled               = true
  # add notification channels
  notification_channels = [google_monitoring_notification_channel.email_notification.id]
  project               = var.project_id
  severity              = "WARNING"
  user_labels           = {}
  alert_strategy {
    auto_close           = "604800s"
    notification_prompts = ["OPENED"]
    notification_rate_limit {
      period = "900s" # 15 minutes
    }
  }
  conditions {
    display_name = "Log match condition"
    condition_matched_log {
      filter           = "severity=WARNING\nresource.labels.service_name=\"token-tltv-cloudrun-service\"\nresource.type=\"cloud_run_revision\""
    }
  }
}

# metric that measures errors from container logs instead of service logs
resource "google_logging_metric" "cloudrun_service_json_payload_error" {
  description      = "level in json payload response is error"
  disabled         = false
  filter           = "resource.type = \"cloud_run_revision\"\nresource.labels.service_name = \"token-tltv-cloudrun-service\"\njsonPayload.level = \"ERROR\"\n"
  name             = "cloud-run-service-json-payload-error"
  project          = "token-tltv"
  metric_descriptor {
    metric_kind  = "DELTA"
    unit         = jsonencode(1)
    value_type   = "INT64"
  }
}

# alerting policy to alert when warnings occur in the logs of the container instead of the logs of the service
# this alert policy also uses the metric created above
resource "google_monitoring_alert_policy" "cloud_run_service_json_payload_error" {
  combiner              = "OR"
  display_name          = "cloud_run_service_json_payload_error"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.id]
  project               = "token-tltv"
  conditions {
    display_name = "token-tltv-cloud-run-service-json-payload-error"
    condition_threshold {
      comparison              = "COMPARISON_GT"
      filter                  = "resource.type = \"cloud_run_revision\" AND metric.type = \"logging.googleapis.com/user/${google_logging_metric.cloudrun_service_json_payload_error.name}\""
      threshold_value         = 1
      duration                = "0s"
      aggregations {
        alignment_period     = "300s"
      }
      trigger {
        count   = 1
        percent = 0
      }
    }
  }
}

# service account to run cloud run job
resource "google_service_account" "tltv_cloudrun_service_identity" {
  account_id = "token-tltv-service-account"
}

resource "google_project_iam_member" "tltv_cloud_translate_user" {
  project = var.project_id
  role    = "roles/cloudtranslate.user"
  member  = "serviceAccount:${google_service_account.tltv_cloudrun_service_identity.email}"
}

resource "google_project_iam_member" "tltv_speech_editor" {
  project = var.project_id
  role    = "roles/speech.editor"
  member  = "serviceAccount:${google_service_account.tltv_cloudrun_service_identity.email}"
}

resource "google_project_iam_member" "tltv_storage_object_user" {
  project = var.project_id
  role    = "roles/storage.objectUser"
  member  = "serviceAccount:${google_service_account.tltv_cloudrun_service_identity.email}"
}

#
resource "google_monitoring_notification_channel" "sms_notification" {
  display_name = "Phone SMS Notification"
  enabled      = true
  force_delete = false
  labels = {
    number = var.sms_notification
  }
  project     = var.project_id
  type        = "sms"
}

resource "google_monitoring_notification_channel" "email_notification" {
  display_name = "Email Notification"
  enabled      = true
  force_delete = false
  labels = {
    email_address = var.email_notification
  }
  project     = var.project_id
  type        = "email"
}