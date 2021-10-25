variable "privatekeypath" {
    type = string
    default= "/ssh/e2essh"
}

variable "publickeypath" {
    type = string
    default= "/ssh/e2essh.pub"
}

variable "user" {
  type = string
  default = "ci"
}

variable "workspace" {
  type = string
  default = "/e2e-testing"
}
