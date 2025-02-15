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

resource "google_monitoring_alert_policy" "cpu_utilization_gt_80" {
  display_name = "cpu-utilization-gt-80"
  documentation {
    content = "The $${metric.display_name} of the $${resource.type} $${resource.label.instance_id} in $${resource.project} has exceeded 50% for over 1 minute."
  }
  combiner     = "OR"
  conditions {
    display_name = "Condition 1"
    condition_threshold {
      comparison = "COMPARISON_GT"
      duration = "0s"
      filter = "resource.type = \"gce_instance\" AND metric.type = \"compute.googleapis.com/instance/cpu/utilization\""
      threshold_value = "0.8"
      trigger {
        count = "1"
      }
    }
  }

  alert_strategy {
    notification_channel_strategy {
      renotify_interval = "1800s"
      notification_channel_names = [google_monitoring_notification_channel.email_notification.name]
    }
  }

  notification_channels = [google_monitoring_notification_channel.email_notification.name]

  user_labels = {
    severity = "warning"
  }
}

resource "google_monitoring_alert_policy" "talkliketv_vm_log_warning" {
  combiner              = "OR"
  display_name          = "talkliketv-vm-log-warning"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.name]
  project               = var.project_id
  severity              = "WARNING"
  user_labels           = {}
  alert_strategy {
    auto_close           = "604800s"
    notification_prompts = ["OPENED"]
    notification_rate_limit {
      period = "3600s"
    }
  }
  conditions {
    display_name = "Log match condition"
    condition_matched_log {
      filter = "severity=WARNING\nresource.type=\"gce_instance\""
      label_extractors = {
        warning_message = "EXTRACT(jsonPayload.message)"
        warning_method  = "EXTRACT(jsonPayload.method)"
        warning_status  = "EXTRACT(jsonPayload.status)"
        warning_uri     = "EXTRACT(jsonPayload.uri)"
      }
    }
  }
}

resource "google_monitoring_alert_policy" "talkliketv_vm_log_error" {
  combiner              = "OR"
  display_name          = "talkliketv-vm-log-error"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.name]
  project               = var.project_id
  severity              = "ERROR"
  user_labels           = {}
  alert_strategy {
    auto_close           = "604800s"
    notification_prompts = ["OPENED"]
    notification_rate_limit {
      period = "3600s"
    }
  }
  conditions {
    display_name = "Log match condition"
    condition_matched_log {
      filter = "severity=ERROR\nresource.type=\"gce_instance\""
      label_extractors = {
        warning_message = "EXTRACT(jsonPayload.message)"
        warning_method  = "EXTRACT(jsonPayload.method)"
        warning_status  = "EXTRACT(jsonPayload.status)"
        warning_uri     = "EXTRACT(jsonPayload.uri)"
      }
    }
  }
}

resource "google_monitoring_alert_policy" "vm_instance_cpu_utilization_gt_80" {
  combiner              = "OR"
  display_name          = "vm-instance-cpu-utilization-gt-80"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.name]
  project               = var.project_id
  alert_strategy {
    auto_close           = "604800s"
  }
  conditions {
    display_name = "VM Instance - High CPU utilization"
    condition_threshold {
      comparison              = "COMPARISON_GT"
      denominator_filter      = null
      duration                = "0s"
      evaluation_missing_data = null
      filter                  = "resource.type = \"gce_instance\" AND metric.type = \"compute.googleapis.com/instance/cpu/utilization\" "
      threshold_value         = 0.8
      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_MEAN"
      }
      trigger {
        count   = 1
        percent = 0
      }
    }
  }
  documentation {
    content   = "This alert fires when the CPU utilization on any VM instance rises above 80% for 5 minutes or more."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "vm_instance_disk_utilization_gt_80" {
  combiner              = "OR"
  display_name          = "vm-instance-disk-utilization-gt-80"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.name]
  project               = var.project_id
  alert_strategy {
    auto_close           = "604800s"
  }
  conditions {
    display_name = "VM Instance - High disk utilization"
    condition_threshold {
      comparison              = "COMPARISON_GT"
      duration                = "0s"
      filter                  = "resource.type = \"gce_instance\" AND metric.type = \"agent.googleapis.com/disk/percent_used\" AND metric.labels.state = \"used\""
      threshold_value         = 80
      aggregations {
        alignment_period     = "300s"
        per_series_aligner   = "ALIGN_MEAN"
      }
      trigger {
        count   = 1
        percent = 0
      }
    }
  }
  documentation {
    content   = "This alert fires when the disk utilization on any VM instance rises above 80% for 5 minutes or more."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "vm_instance_memory_utilization_gt_80" {
  combiner              = "OR"
  display_name          = "vm-instance-memory-utilization-gt-80"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.name]
  project               = var.project_id
  alert_strategy {
    auto_close           = "604800s"
  }
  conditions {
    display_name = "VM Instance - High memory utilization"
    condition_threshold {
      comparison              = "COMPARISON_GT"
      duration                = "0s"
      filter                  = "resource.type = \"gce_instance\" AND metric.type = \"agent.googleapis.com/memory/percent_used\" AND metric.labels.state = \"used\""
      threshold_value         = 80
      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
      }
      trigger {
        count   = 1
        percent = 0
      }
    }
  }
  documentation {
    content   = "This alert fires when the memory utilization on any VM instance rises above 80% for 60 seconds or more."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_alert_policy" "vm_instance_host_error_log_detected" {
  combiner              = "OR"
  display_name          = "vm-instance-host-error-log-detected"
  enabled               = true
  notification_channels = [google_monitoring_notification_channel.email_notification.name, google_monitoring_notification_channel.sms_notification.name]
  project               = var.project_id
  alert_strategy {
    auto_close           = "604800s"
    notification_rate_limit {
      period = "3600s"
    }
  }
  conditions {
    display_name = "VM Instance - Host Error Log Detected"
    condition_matched_log {
      filter           = "log_id(\"cloudaudit.googleapis.com/system_event\") AND operation.producer=\"compute.instances.hostError\""
    }
  }
  documentation {
    content   = "This alert fires when any host error is detected on any VM instance based on system_event logs, limited to notifying once per hour."
    mime_type = "text/markdown"
  }
}
