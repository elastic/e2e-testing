variable "privatekeypath" {
    type = string
    default= "../e2essh"
}

variable "publickeypath" {
    type = string
    default= "../e2essh.pub"
}

variable "user" {
  type = string
  default = "ci"
}
