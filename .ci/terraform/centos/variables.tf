variable "privatekeypath" {
    type = string
    default= ".ci/terraform/e2essh"
}

variable "publickeypath" {
    type = string
    default= ".ci/terraform/e2essh.pub"
}

variable "user" {
  type = string
  default = "ci"
}

variable "workspace" {
  type = string
  default = "/tmp"
}

variable "goarch" {
  type = string
  default = "amd64"
}

variable "provider_type" {
  type = string
  default = "remote"
}

variable "log_level" {
  type = string
  default = "TRACE"
}

variable "op_log_level" {
  type = string
  default = "TRACE"
}

variable "kibana_url" {
  type = string
}
variable "kibana_password" {
  type = string
}
variable "elasticsearch_url" {
  type = string
}
variable "elasticsearch_password" {
  type = string
}
variable "fleet_url" {
  type = string
}
variable "skip_pull" {
  type = string
  default = "1"
}
variable "base_dir" {
  type = string
}

variable "tags"{
  type = string
}

variable "suite"{
  type = string
}
