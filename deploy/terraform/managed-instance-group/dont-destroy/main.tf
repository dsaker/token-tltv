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

resource "google_firestore_database" "database" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
  deletion_policy = "ABANDON"
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
