data "google_compute_address" "static_xyz" {
  name = var.static_ip_com
}

data "google_service_account" "tltv_sa_xyz" {
  account_id = var.sa_account_id
}

data "google_compute_image" "debian" {
  family  = "debian-12"
  project = "debian-cloud"
}

data "google_compute_network" "tltv_network" {
  name = var.tltv_network
}

data "google_compute_subnetwork" "tltv_subnetwork" {
  name          = var.tltv_subnetwork
}
# Create a single Compute Engine instance
resource "google_compute_instance" "talkliketv_com" {
  name                      = "talkliketv-vm-com"
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
    network    =  data.google_compute_network.tltv_network.id
    subnetwork =  data.google_compute_subnetwork.tltv_subnetwork.id
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
