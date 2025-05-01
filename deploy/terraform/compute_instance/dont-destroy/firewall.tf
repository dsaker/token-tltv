
data "google_compute_network" "default" {
  name = "default"
}

# allow ssh to talkliketv vpc
resource "google_compute_firewall" "talkliketv_default_allow_ssh" {
  name    = "talkliketv-default-allow-ssh"
  network = data.google_compute_network.default.name

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags   = ["ssh-talkliketv"]
  source_ranges = [var.my_ip]
}