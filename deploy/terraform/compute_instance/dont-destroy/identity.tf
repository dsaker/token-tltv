# service account to allow compute-engine to access necessary google api's
resource "google_service_account" "tltv_ce_service_account" {
  account_id = var.sa_account_id
  project    = var.project_id
}

resource "google_project_iam_member" "ce_cloud_translate_user" {
  project = var.project_id
  role    = "roles/cloudtranslate.user"
  member  = "serviceAccount:${google_service_account.tltv_ce_service_account.email}"
}

resource "google_project_iam_member" "ce_speech_editor" {
  project = var.project_id
  role    = "roles/speech.editor"
  member  = "serviceAccount:${google_service_account.tltv_ce_service_account.email}"
}

resource "google_project_iam_member" "ce_storage_object_user" {
  project = var.project_id
  role    = "roles/storage.objectUser"
  member  = "serviceAccount:${google_service_account.tltv_ce_service_account.email}"
}

resource "google_project_iam_member" "ce_cloud_datastore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.tltv_ce_service_account.email}"
}

resource "google_project_iam_member" "ce_logging_logWriter" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.tltv_ce_service_account.email}"
}

resource "google_compute_address" "static_xyz" {
  name = var.static_ip_xyz
}

resource "google_compute_address" "static_com" {
  name = var.static_ip_com
}