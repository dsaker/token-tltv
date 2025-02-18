variable "region" {
  description = "The GCP region to deploy instances into"
  type = string
}

variable "project_id" {
  type = string
}

variable "machine_type" {
  type = string
}

variable "gce_ssh_pub_key_file" {
  type = string
}

variable "gce_ssh_private_key_file" {
  type = string
}

variable "gce_ssh_user" {
  type = string
}

variable "sa_account_id" {
  type = string
}

variable "my_ip" {
    type = string
}

variable "static_ip_name" {
  type = string
}

variable "zone" {
    type = string
}

variable "email_notification_display_name" {
  type = string
}