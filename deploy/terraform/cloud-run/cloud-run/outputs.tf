output "image_id" {
  value = local.image
}

output "cloud_run_url" {
  value = google_cloud_run_v2_service.token-tltv.urls
}