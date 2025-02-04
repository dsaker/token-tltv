resource "google_artifact_registry_repository" "token_tltv" {
  repository_id = var.repository_id
  description   = "docker repository for token-tltv project"
  format        = "DOCKER"
  cleanup_policies {
    id     = "keep-tagged-release"
    action = "KEEP"
    condition {
      tag_state             = "TAGGED"
      tag_prefixes          = ["latest"]
      package_name_prefixes = ["token-tltv"]
    }
  }

}

# create locals data to store registry info for image name
locals {
  instance_name = format("%s-%s", var.instance_name, substr(md5(module.gce-tltv-container.container.image), 0, 8))
  l = google_artifact_registry_repository.token_tltv.location
  p = google_artifact_registry_repository.token_tltv.project
  r = google_artifact_registry_repository.token_tltv.repository_id
  image = "${local.l}-docker.pkg.dev/${local.p}/${local.r}/${var.image_name}:${var.image_version}"
}

resource "google_compute_address" "tltv_ipv4_address" {
  name = "tltv-ipv4-address"
}

module "gce-tltv-container" {
  source = "terraform-google-modules/container-vm/google"
  version = "~> 3.2"

  container = {
    image=local.image
    env = [
      {
        name = "PROJECT_ID"
        value = var.project_id
      },
      {
        name  = "FIRESTORE_TOKENS"
        value = var.firestore_tokens
      }
    ],
  }

  restart_policy = "Always"
}

resource "google_compute_instance" "tltv_gce_instance" {
  name         = local.instance_name
  machine_type               = "e2-micro"
  zone = var.zone
  tags = [tolist(google_compute_firewall.tcp_allow_8080.target_tags)[0]]
  project                 = var.project_id

  boot_disk {
    initialize_params {
      image = module.gce-tltv-container.source_image
    }
  }

  network_interface {
    subnetwork_project = "token-tltv"
    subnetwork         = "https://www.googleapis.com/compute/v1/projects/token-tltv/regions/us-central1/subnetworks/default"
    access_config {
      nat_ip                 = google_compute_address.tltv_ipv4_address.address
    }
  }

  metadata = {
    gce-container-declaration = module.gce-tltv-container.metadata_value
    google-logging-enabled    = "true"
    google-monitoring-enabled = "true"
  }

  labels = {
    container-vm = module.gce-tltv-container.vm_container_label
  }

  service_account {
    email  = "130230417309-compute@developer.gserviceaccount.com"
    scopes = ["https://www.googleapis.com/auth/datastore", "https://www.googleapis.com/auth/devstorage.read_only", "https://www.googleapis.com/auth/logging.write", "https://www.googleapis.com/auth/monitoring.write", "https://www.googleapis.com/auth/service.management.readonly", "https://www.googleapis.com/auth/servicecontrol", "https://www.googleapis.com/auth/trace.append"]
  }

  scheduling {
    preemptible       = "true"
    automatic_restart = "false"
  }
}

resource "google_compute_firewall" "tcp_allow_8080" {
  direction               = "INGRESS"
  name                    = "tcp-allow-8080"
  network                 = "https://www.googleapis.com/compute/v1/projects/token-tltv/regions/us-central1/subnetworks/default"
  priority                = 1000
  project                 = var.project_id
  source_ranges           = ["0.0.0.0/0"]
  target_tags             = ["tcp-allow-8080"]
  allow {
    ports    = ["8080"]
    protocol = "tcp"
  }
}

# resource "google_compute_network" "default" {
#   name = "default"
# }