resource "google_service_account" "tltv_test_service_identity" {
  account_id = "tltv-test-service-account"
  project    = var.test_project_id
}

resource "google_project_iam_member" "test_cloud_translate_user" {
  project = var.test_project_id
  role    = "roles/cloudtranslate.user"
  member  = "serviceAccount:${google_service_account.tltv_test_service_identity.email}"
}

resource "google_project_iam_member" "test_speech_editor" {
  project = var.test_project_id
  role    = "roles/speech.editor"
  member  = "serviceAccount:${google_service_account.tltv_test_service_identity.email}"
}

resource "google_project_iam_member" "test_cloud_datastore_user" {
  project = var.test_project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.tltv_test_service_identity.email}"
}

output tltv_test_email {
  value = google_service_account.tltv_test_service_identity.email
}