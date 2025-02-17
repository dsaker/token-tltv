data "google_artifact_registry_repository" "token_tltv" {
  location      = var.region
  repository_id = var.repository_id
}

# create local data to store registry info for image name
locals {
  l = data.google_artifact_registry_repository.token_tltv.location
  p = data.google_artifact_registry_repository.token_tltv.project
  r = data.google_artifact_registry_repository.token_tltv.repository_id
  image = "${local.l}-docker.pkg.dev/${local.p}/${local.r}/${var.image_name}:${var.image_version}"
}

data "google_compute_subnetwork" "tltv_cr_subnetwork" {
  name   = "tltv-cr-subnetwork"
  region = var.region
}

data "google_service_account" "tltv_cloudrun_service_identity" {
  account_id = "token-tltv-cloudrun-sa"
}

# cloud run service to run container
resource "google_cloud_run_v2_service" "token-tltv" {
  name                 = "token-tltv-cloudrun-service"
  ingress              = "INGRESS_TRAFFIC_ALL"
  project              = var.project_id
  location             = data.google_compute_subnetwork.tltv_cr_subnetwork.region
  template {
    service_account = data.google_service_account.tltv_cloudrun_service_identity.email
    session_affinity                 = false
    timeout                          = "300s"
    containers {
      image       = local.image
      env {
        name  = "FIRESTORE_TOKENS"
        value = var.firestore_tokens
      }
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
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
    # vpc_access {
    #   connector = google_vpc_access_connector.tltv_cr_conn.id
    #   egress    = "ALL_TRAFFIC"
    # }
  }
  traffic {
    percent  = 100
    type     = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
  # Used in sample testing. These fields may change in 'terraform plan' output, which is expected and thus non-blocking.
  lifecycle {
    ignore_changes = [
      ingress#, template[0].vpc_access
    ]
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
