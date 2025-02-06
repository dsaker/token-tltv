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

resource "google_compute_managed_ssl_certificate" "tltv_ssl_cert" {
  name     = "tltv-ssl-cert"
  managed {
    domains = ["talkliketv.xyz"]
  }
}

resource "google_compute_global_address" "tltv_global_address" {
  name       = "tltv-global-address"
  ip_version = "IPV4"
}

output "google_compute_managed_ssl_certificate_id" {
  value = google_compute_managed_ssl_certificate.tltv_ssl_cert.id
}

output "google_compute_global_address_id" {
  value = google_compute_global_address.tltv_global_address.id
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

resource "google_project_iam_member" "tltv_cloud_datastore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.tltv_cloudrun_service_identity.email}"
}