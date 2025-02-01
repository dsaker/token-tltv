# Example of setting up a Cloud Run service with a static outbound IP
resource "google_compute_network" "tltv_network" {
  name = "tltv-cr-static-ip-network"
}

resource "google_compute_subnetwork" "tltv_subnetwork" {
  name          = "tltv-cr-static-ip"
  ip_cidr_range = "10.124.0.0/28"
  network       = google_compute_network.tltv_network.id
  region        = var.region
}

resource "google_vpc_access_connector" "tltv_cr_conn" {
  name          = "tltv-cr-conn"
  region        = var.region
  min_instances = 2
  max_instances = 3

  subnet {
    name = google_compute_subnetwork.tltv_subnetwork.name
  }

  # Wait for VPC API enablement
  # before creating this resource
  depends_on = [
    google_project_service.vpc
  ]
}

resource "google_compute_router" "tltv_cr_static_ip_router" {
  name    = "tltv-cr-static-ip-router"
  network = google_compute_network.tltv_network.name
  region  = google_compute_subnetwork.tltv_subnetwork.region
}

resource "google_compute_address" "tltv_cr_static_ip_addr" {
  name   = "tltv-cr-static-ip-addr"
  region = google_compute_subnetwork.tltv_subnetwork.region
}

resource "google_compute_router_nat" "tltv_cr_static_nat" {
  name   = "tltv-cr-static-nat"
  router = google_compute_router.tltv_cr_static_ip_router.name
  region = google_compute_subnetwork.tltv_subnetwork.region

  nat_ip_allocate_option = "MANUAL_ONLY"
  nat_ips                = [google_compute_address.tltv_cr_static_ip_addr.self_link]

  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"
  subnetwork {
    name                    = google_compute_subnetwork.tltv_subnetwork.id
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }
}
