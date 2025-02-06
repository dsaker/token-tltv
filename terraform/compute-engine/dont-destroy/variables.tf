variable "repository_id" {
  type = string
  description = "the name of your repository"
  default = null
}

variable "region" {
  description = "The GCP region to deploy instances into"
  type = string
}

variable "project_id" {
  type = string
}