# Example of setting up a Cloud Run service with a static outbound IP
resource "google_compute_network" "tltv_network" {
  name = "tltv-cr-static-ip-network"
}

resource "google_compute_subnetwork" "tltv_subnetwork" {
  name          = "tltv-cr-subnetwork"
  ip_cidr_range = "10.124.0.0/28"
  network       = google_compute_network.tltv_network.id
  region        = var.region
}