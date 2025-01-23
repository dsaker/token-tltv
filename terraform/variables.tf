variable "project_id" {
  type        = string
  default     = null
}

variable "region" {
  type = string
  default = null
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