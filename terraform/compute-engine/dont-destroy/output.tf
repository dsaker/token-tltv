output "google_compute_managed_ssl_certificate_id" {
  value = google_compute_managed_ssl_certificate.tltv_ssl_cert.id
}

output "google_compute_global_address_id" {
  value = google_compute_global_address.tltv_global_address.id
}

output "google_service_account_email" {
  value = google_service_account.tltv_mig_service_identity.email
}