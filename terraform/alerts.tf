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
  project          = var.project_id
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
