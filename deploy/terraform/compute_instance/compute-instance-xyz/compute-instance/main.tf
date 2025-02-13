data "google_compute_address" "static_xyz" {
  name = var.static_ip_name
}

data "google_service_account" "tltv_sa_xyz" {
  account_id = var.sa_account_id
}

resource "google_compute_network" "vpc_network_xyz" {
  name                    = "talkliketv-vpc-network-xyz"
  auto_create_subnetworks = false
  mtu                     = 1460
}

resource "google_compute_subnetwork" "subnetwork_talkliketv_xyz" {
  name          = "talkliketv-subnet-xyz"
  ip_cidr_range = "10.0.1.0/24"
  region        = var.region
  network       = google_compute_network.vpc_network_xyz.id
}

data "google_compute_image" "debian" {
  family  = "debian-12"
  project = "debian-cloud"
}

# Create a single Compute Engine instance
resource "google_compute_instance" "talkliketv_xyz" {
  name                      = "talkliketv-vm-xyz"
  machine_type              = var.machine_type
  tags                      = ["ssh-talkliketv-xyz", "https-talkliketv-xyz"]
  allow_stopping_for_update = true
  zone = var.zone

  metadata = {
    ssh-keys = "${var.gce_ssh_user}:${file(var.gce_ssh_pub_key_file)}"
  }

  metadata_startup_script = "echo  PROJECT_ID=${var.project_id} >> /etc/environment"

  boot_disk {
    initialize_params {
      image = data.google_compute_image.debian.self_link
    }
  }

  network_interface {
    access_config {
      nat_ip = data.google_compute_address.static_xyz.address
    }
    network    =  google_compute_network.vpc_network_xyz.id
    subnetwork =  google_compute_subnetwork.subnetwork_talkliketv_xyz.id
  }

  scheduling {
    automatic_restart   = false
    provisioning_model  = "SPOT"
    preemptible = true
    instance_termination_action = "STOP"
  }

  service_account {
    email  = data.google_service_account.tltv_sa_xyz.email
    scopes = ["https://www.googleapis.com/auth/datastore", "https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/devstorage.read_only", "https://www.googleapis.com/auth/logging.write", "https://www.googleapis.com/auth/monitoring.write", "https://www.googleapis.com/auth/trace.append"]
  }

  connection {
    type     = "ssh"
    user     = var.gce_ssh_user
    host     = data.google_compute_address.static_xyz.address
    private_key = file(var.gce_ssh_private_key_file)
  }
}

# allow ssh to talkliketv vpc
resource "google_compute_firewall" "talkliketv_vpc_network_allow_ssh_xyz" {
  name    = "talkliketv-vpc-network-allow-ssh-xyz"
  network = google_compute_network.vpc_network_xyz.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags   = ["ssh-talkliketv-xyz"]
  source_ranges = [var.my_ip]
}

# allow https to talkliketv vpc
resource "google_compute_firewall" "talkliketv_vpc_network_allow_https_xyz" {
  name    = "talkliketv-vpc-network-allow-https-xyz"
  network = google_compute_network.vpc_network_xyz.name

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  target_tags   = ["https-talkliketv-xyz"]
  source_ranges = ["0.0.0.0/0"]
}
