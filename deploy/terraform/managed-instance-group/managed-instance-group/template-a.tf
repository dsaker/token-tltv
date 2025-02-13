resource "google_compute_instance_template" "tltv_instance_template_a" {
  name = "tltv-instance-template-a"
  disk {
    auto_delete  = true
    boot         = true
    device_name  = "persistent-disk-0"
    mode         = "READ_WRITE"
    source_image = module.gce-tltv-container.source_image
    type         = "PERSISTENT"
  }
  labels = {
    managed-by-cnrm = "true"
  }
  machine_type = var.machine_type
  metadata = {
    gce-container-declaration = module.gce-tltv-container.metadata_value
    google-logging-enabled    = "true"
    google-monitoring-enabled = "true"
  }
  network_interface {
    access_config {
      network_tier = "PREMIUM"
    }
    network    =  "https://www.googleapis.com/compute/v1/projects/${var.project_id}/global/networks/default"
    subnetwork =  "https://www.googleapis.com/compute/v1/projects/${var.project_id}/regions/${var.region}/subnetworks/default"
  }
  region = var.region
  scheduling {
    automatic_restart   = false
    provisioning_model  = "SPOT"
    preemptible = true
    instance_termination_action = "STOP"
  }
  service_account {
    email  = var.tltv_sa_email
    scopes = ["https://www.googleapis.com/auth/datastore", "https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/devstorage.read_only", "https://www.googleapis.com/auth/logging.write", "https://www.googleapis.com/auth/monitoring.write","https://www.googleapis.com/auth/service.management.readonly", "https://www.googleapis.com/auth/servicecontrol", "https://www.googleapis.com/auth/trace.append"]
  }
  tags = ["allow-health-check"]
}

