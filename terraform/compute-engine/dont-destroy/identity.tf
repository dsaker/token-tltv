# service account to allow cloud run service to access necessary google api's
resource "google_service_account" "tltv_mig_service_identity" {
  account_id = "token-tltv-mig-service-account"
}

resource "google_project_iam_member" "tltv_cloud_translate_user" {
  project = var.project_id
  role    = "roles/cloudtranslate.user"
  member  = "serviceAccount:${google_service_account.tltv_mig_service_identity.email}"
}

resource "google_project_iam_member" "tltv_speech_editor" {
  project = var.project_id
  role    = "roles/speech.editor"
  member  = "serviceAccount:${google_service_account.tltv_mig_service_identity.email}"
}

resource "google_project_iam_member" "tltv_storage_object_user" {
  project = var.project_id
  role    = "roles/storage.objectUser"
  member  = "serviceAccount:${google_service_account.tltv_mig_service_identity.email}"
}

resource "google_project_iam_member" "tltv_cloud_datastore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.tltv_mig_service_identity.email}"
}

resource "google_project_iam_member" "tltv_artifactregistry_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.tltv_mig_service_identity.email}"
}

resource "google_project_iam_member" "tltv_logging_logWriter" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.tltv_mig_service_identity.email}"
}