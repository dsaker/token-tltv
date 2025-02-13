data "google_compute_address" "static" {
  name = var.static_ip_name
}

data "google_service_account" "tltv_sa" {
  account_id = var.sa_account_id
}

resource "google_compute_network" "vpc_network" {
  name                    = "talkliketv-vpc-network"
  auto_create_subnetworks = false
  mtu                     = 1460
}

resource "google_compute_subnetwork" "subnetwork_talkliketv" {
  name          = "talkliketv-subnet"
  ip_cidr_range = "10.0.1.0/24"
  region        = var.region
  network       = google_compute_network.vpc_network.id
}

data "google_compute_image" "debian" {
  family  = "debian-12"
  project = "debian-cloud"
}

# Create a single Compute Engine instance
resource "google_compute_instance" "talkliketv" {
  name                      = "talkliketv-vm"
  machine_type              = var.machine_type
  tags                      = ["ssh-talkliketv", "https-talkliketv"]
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
      nat_ip = data.google_compute_address.static.address
    }
    network    =  google_compute_network.vpc_network.id
    subnetwork =  google_compute_subnetwork.subnetwork_talkliketv.id
  }

  scheduling {
    automatic_restart   = false
    provisioning_model  = "SPOT"
    preemptible = true
    instance_termination_action = "STOP"
  }

  service_account {
    email  = data.google_service_account.tltv_sa.email
    scopes = ["https://www.googleapis.com/auth/datastore", "https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/devstorage.read_only", "https://www.googleapis.com/auth/logging.write", "https://www.googleapis.com/auth/monitoring.write", "https://www.googleapis.com/auth/trace.append"]
  }

  connection {
    type     = "ssh"
    user     = var.gce_ssh_user
    host     = data.google_compute_address.static.address
    private_key = file(var.gce_ssh_private_key_file)
  }
}

# allow ssh to talkliketv vpc
resource "google_compute_firewall" "talkliketv_vpc_network_allow_ssh" {
  name    = "talkliketv-vpc-network-allow-ssh"
  network = google_compute_network.vpc_network.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags   = ["ssh-talkliketv"]
  source_ranges = [var.my_ip]
}

# allow https to talkliketv vpc
resource "google_compute_firewall" "talkliketv_vpc_network_allow_https" {
  name    = "talkliketv-vpc-network-allow-https"
  network = google_compute_network.vpc_network.name

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  target_tags   = ["https-talkliketv"]
  source_ranges = ["0.0.0.0/0"]
}
