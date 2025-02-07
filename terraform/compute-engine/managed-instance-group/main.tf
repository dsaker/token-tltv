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

data "google_artifact_registry_repository" "token_tltv" {
  location = var.region
  repository_id = var.repository_id
}

# create locals data to store registry info for image name
locals {
  instance_name = format("%s-%s", var.instance_name, substr(md5(module.gce-tltv-container.container.image), 0, 8))
  l = data.google_artifact_registry_repository.token_tltv.location
  p = data.google_artifact_registry_repository.token_tltv.project
  r = data.google_artifact_registry_repository.token_tltv.repository_id
  image = "${local.l}-docker.pkg.dev/${local.p}/${local.r}/${var.image_name}:${var.image_version}"
}

resource "google_compute_firewall" "health_check_8080" {
  name          = "health-check-8080"
  direction     = "INGRESS"
  network       = "https://www.googleapis.com/compute/v1/projects/${var.project_id}/global/networks/default"
  priority      = 1000
  source_ranges = ["130.211.0.0/22", "35.191.0.0/16"]
  target_tags   = ["allow-health-check"]
  allow {
    ports    = ["8080"]
    protocol = "tcp"
  }
}

resource "google_compute_health_check" "tltv_basic_check" {
  name               = "tltv-basic-check"
  check_interval_sec = 300
  healthy_threshold  = 2
  http_health_check {
    port               = 8080
    port_specification = "USE_FIXED_PORT"
    proxy_header       = "NONE"
    request_path       = "/"
  }
  timeout_sec         = 5
  unhealthy_threshold = 2
}

resource "google_compute_target_https_proxy" "tltv_https_proxy" {
  name     = "tltv-https-proxy"
  url_map  = google_compute_url_map.tltv_url_map.id

  ssl_certificates = [
    "projects/${var.project_id}/global/sslCertificates/tltv-ssl-cert"
  ]
}

resource "google_compute_global_forwarding_rule" "tltv_forwarding_rule" {
  name                  = "tltv-forwarding-rule"
  load_balancing_scheme = "EXTERNAL"
  port_range            = "443"
  target                = google_compute_target_https_proxy.tltv_https_proxy.id
  ip_address            = "projects/${var.project_id}/global/addresses/tltv-global-address"
}

resource "google_compute_autoscaler" "tltv_autoscaler" {
  name   = "tltv-autoscaler"
  zone   = var.zone
  target = google_compute_instance_group_manager.tltv_instance_group_manager.id

  autoscaling_policy {
    max_replicas    = 2
    min_replicas    = 0
    cooldown_period = 60

    cpu_utilization {
      target = 0.6
    }
  }
}

resource "google_compute_backend_service" "tltv_backend_service" {
  name                            = "tltv-backend-service"
  connection_draining_timeout_sec = 0
  health_checks                   = [google_compute_health_check.tltv_basic_check.id]
  load_balancing_scheme           = "EXTERNAL"
  port_name                       = "tcp-8080"
  protocol                        = "HTTP"
  session_affinity                = "NONE"
  timeout_sec                     = 30
  backend {
    group           = google_compute_instance_group_manager.tltv_instance_group_manager.instance_group
    balancing_mode  = "UTILIZATION"
    capacity_scaler = 1.0
  }
}

resource "google_compute_url_map" "tltv_url_map" {
  name            = "tltv-url-map"
  default_service = google_compute_backend_service.tltv_backend_service.id
}

resource "google_compute_instance_group_manager" "tltv_instance_group_manager" {
  name = "tltv-instance-group-manager"
  zone = var.zone
  named_port {
    name = "tcp-8080"
    port = 8080
  }
  version {
    instance_template = google_compute_instance_template.tltv_instance_template_a.id
    name              = "primary"
  }
  base_instance_name = "vm"
}