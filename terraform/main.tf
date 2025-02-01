resource "google_artifact_registry_repository" "token_tltv" {
  location      = var.region
  repository_id = var.repository_id
  description   = "docker repository for token-tltv project"
  format        = "DOCKER"

}

# create locals data to store registry info for image name
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

# cloud run service to run container
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

# service account to allow cloud run service to access necessary google api's
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

resource "google_firestore_database" "database" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
  deletion_policy = "ABANDON"
}