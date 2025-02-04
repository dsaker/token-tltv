variable "project_id" {
  type = string
}

variable "region" {
  description = "The GCP region to deploy instances into"
  type = string
}

variable "zone" {
  description = "The GCP zone to deploy instances into"
  type        = string
}

variable "instance_name" {
  description = "The desired name to assign to the deployed instance"
  default     = "token-tltv"
}

variable "talkliketv_machine_type" {
  type = string
}

variable "module_sa_account_id" {
  description = "Name of the service account id for the bucket."
  type        = string
}

variable "repository_id" {
  type = string
  description = "the name of your repository"
  default = null
}

variable "image_name" {
  type = string
  description = "the name of image to run"
  default = null
}

variable "image_version" {
  type = string
  description = "the version of image to run"
  default = null
}

variable "sms_notification" {
  type = string
  description = "sms to be notified for cloud run job error"
  default = null
}

variable "email_notification" {
  type = string
  description = "email to be notified for cloud run job error"
  default = null
}

variable "firestore_tokens" {
  type        = string
  default     = null
}
