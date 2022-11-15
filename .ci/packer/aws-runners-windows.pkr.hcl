packer {
  required_plugins {
    amazon = {
      version = ">= 1.1.5"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "skip_create_ami" {
  type = bool
  default = false
}

locals {
  aws_region = "us-east-2"
  force_deregister = true
}

source "amazon-ebs" "windows2019" {
  ami_name        = "windows-2019-runner-1"
  instance_type   = "c5.2xlarge"
  region          = local.aws_region
  source_ami      = "ami-0587bd602f1da2f1d"
  ssh_username  = "ogc"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 60
    volume_type = "gp3"
  }

  tags            = {
    OS_Version  = "Windows"
    Release     = "2019"
    Arch        = "AMD64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

build {
  name = "e2e runners AMIs"
  sources = [
    "source.amazon-ebs.windows2019"
  ]

  provisioner "ansible" {
    user = build.User
    ansible_env_vars  =  ["PACKER_BUILD_NAME={{ build_name }}"]
    playbook_file     = ".ci/ansible/playbook.yml"
    extra_arguments   = ["--tags", "setup-ami", "--extra-vars", "nodeShellType=cmd"]
    galaxy_file       = ".ci/ansible/requirements.yml"
  }
}