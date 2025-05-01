resource "google_compute_address" "static_xyz" {
  name = "xyz-ip-address"
}

data "google_service_account" "tltv_sa_xyz" {
  account_id = var.sa_account_id
}

data "google_compute_network" "tltv_network" {
  name = "default"
}

data "google_compute_subnetwork" "tltv_subnetwork" {
  name          = "default"
}

data "google_compute_image" "debian" {
  family  = "debian-12"
  project = "debian-cloud"
}

# Create a single Compute Engine instance
resource "google_compute_instance" "talkliketv_xyz" {
  name                      = "talkliketv-vm-xyz"
  machine_type              = var.machine_type
  tags                      = ["ssh-talkliketv", "https-server"]
  allow_stopping_for_update = true
  zone = var.zone

  metadata = {
    ssh-keys = "${var.gce_ssh_user}:${file(var.gce_ssh_pub_key_file)}"
  }

  metadata_startup_script = "echo  PROJECT_ID=${var.project_id} >> /etc/environment"

  boot_disk {
    initialize_params {
      image = data.google_compute_image.debian.self_link
    }
  }

  network_interface {
    access_config {
      nat_ip = google_compute_address.static_xyz.address
    }
    network    =  data.google_compute_network.tltv_network.id
    subnetwork =  data.google_compute_subnetwork.tltv_subnetwork.id
  }

  scheduling {
    automatic_restart   = true
    preemptible = false
  }


  service_account {
    email  = data.google_service_account.tltv_sa_xyz.email
    scopes = ["https://www.googleapis.com/auth/datastore", "https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/devstorage.read_only", "https://www.googleapis.com/auth/logging.write", "https://www.googleapis.com/auth/monitoring.write", "https://www.googleapis.com/auth/trace.append"]
  }

  connection {
    type     = "ssh"
    user     = var.gce_ssh_user
    host     = google_compute_address.static_xyz.address
    private_key = file(var.gce_ssh_private_key_file)
  }
}

// setup uptime check for talkliketv.xyz
resource "google_monitoring_uptime_check_config" "talkliketv_xyz_uptime_check" {
  display_name = "talkliketv-xyz-uptime-check"

  http_check {
    path = "/"
    port = "443"
    request_method = "GET"
    use_ssl = true
    validate_ssl = true
    accepted_response_status_codes {
      status_class = "STATUS_CLASS_2XX"
    }
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = "talkliketv.xyz"
    }
  }

  timeout = "10s"
  period  = "300s"
}

data "google_monitoring_notification_channel" "email_notification" {
  display_name = var.email_notification_display_name
}

resource "google_monitoring_alert_policy" "talkliketv_xyz_uptime_failure" {
  combiner              = "OR"
  display_name          = "talkliketv-xyz-uptime-failure"
  enabled               = true
  notification_channels = [data.google_monitoring_notification_channel.email_notification.name]
  project               = "token-tltv-450304"
  severity = "ERROR"
  conditions {
    display_name = "Failure of uptime check_id ${google_monitoring_uptime_check_config.talkliketv_xyz_uptime_check.display_name}"
    condition_threshold {
      comparison              = "COMPARISON_GT"
      duration                = "60s"
      filter                  = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND metric.label.check_id=\"${google_monitoring_uptime_check_config.talkliketv_xyz_uptime_check.uptime_check_id}\" AND resource.type=\"uptime_url\""
      threshold_value         = 1
      aggregations {
        alignment_period     = "1200s"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.label.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }
      trigger {
        count   = 1
        percent = 0
      }
    }
  }
}
