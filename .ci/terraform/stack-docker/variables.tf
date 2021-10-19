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

variable "base_dir" {
  type = string
}
