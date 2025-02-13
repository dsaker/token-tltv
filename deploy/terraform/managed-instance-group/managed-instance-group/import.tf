# terraform plan -generate-config-out=generated.tf
#
# import {
#   id = "projects/${var.project_id}/global/instanceTemplates/tltv-instance-template-20250206-163513"
#   //id = "projects/{{project}}/global/instanceTemplates/{{name}}"
#   to = google_compute_instance_template.default
# }