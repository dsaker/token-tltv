# Enable Compute Engine API
resource "google_project_service" "compute_engine_api" {
  service            = "compute.googleapis.com"
  disable_on_destroy = false
}

# Enable Cloud Run API
resource "google_project_service" "cloudrun_api" {
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

# Enable Artifact Registry API
resource "google_project_service" "artifact_registry_api" {
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

# Enable Translate API
resource "google_project_service" "translate_api" {
  service            = "translate.googleapis.com"
  disable_on_destroy = false
}

# Enable TextToSpeech API
resource "google_project_service" "texttospeech_api" {
  service            = "texttospeech.googleapis.com"
  disable_on_destroy = false
}

# Enable Vpc Access
resource "google_project_service" "vpc" {
  service            = "vpcaccess.googleapis.com"
  disable_on_destroy = false
}