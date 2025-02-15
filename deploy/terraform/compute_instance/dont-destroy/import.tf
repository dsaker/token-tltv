# terraform plan -generate-config-out=generated.tf
# #
# import {
#   // $ terraform import google_monitoring_alert_policy.default token-tltv-450304/talkliketv-vm-com-warning
#
#   id = "projects/token-tltv-450304/alertPolicies/16935055066143363649"
#   //id = "projects/{{project}}/global/instanceTemplates/{{name}}"
#   to = google_monitoring_alert_policy.default
# }