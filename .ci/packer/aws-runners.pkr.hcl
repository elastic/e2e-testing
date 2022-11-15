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

source "amazon-ebs" "ubuntu" {
  ami_name      = "ubuntu-2204-e2e-runner-1"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0aeb7c931a5a61206"
  ssh_username  = "ubuntu"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags = {
    OS_Version = "Ubuntu"
    Release    = "22.04"
    Arch       = "AMD64"
  }
  skip_create_ami = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "debian-10-amd64" {
  ami_name      = "debian-10-amd64-runner-1"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0d90bed76900e679a"
  ssh_username  = "admin"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags = {
    OS_Version = "Debian"
    Release    = "10"
    Arch       = "AMD64"
  }
  skip_create_ami = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "debian-10-arm64" {
  ami_name      = "debian-10-arm64-runner-1"
  instance_type = "a1.large"
  region        = local.aws_region
  source_ami    = "ami-06dac44ad759182bd"
  ssh_username  = "admin"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags = {
    OS_Version = "Debian"
    Release    = "10"
    Arch       = "ARM64"
  }
  skip_create_ami = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "debian-11-amd64" {
  ami_name      = "debian-11-amd64-runner-1"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0c7c4e3c6b4941f0f"
  ssh_username  = "admin"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags          = {
    OS_Version = "Debian"
    Release    = "11"
    Arch       = "AMD64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "centos-8-amd64" {
  ami_name      = "centos-8-amd64-runner-1"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-045b0a05944af45c1"
  ssh_username  = "centos"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags          = {
    OS_Version = "Centos"
    Release    = "8"
    Arch       = "AMD64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "centos-8-arm64" {
  ami_name      = "centos-8-arm64-runner-1"
  instance_type = "a1.large"
  region        = local.aws_region
  source_ami    = "ami-01cdc9e8306344fe0"
  ssh_username  = "centos"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags          = {
    OS_Version = "Centos"
    Release    = "8"
    Arch       = "ARM64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "oracle-linux-8" {
  ami_name      = "oracle-linux-8-x86-64-runner-1"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-00371eeb8fd8e0e16"
  ssh_username  = "ec2-user"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags          = {
    OS_Version = "Oracle Linux"
    Release    = "8"
    Arch       = "x86-64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister

}

source "amazon-ebs" "sles15" {
  ami_name      = "sles15-runner-1"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0f7cb53c916a75006"
  ssh_username  = "ec2-user"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name = "/dev/sda1"
    volume_size = 15
    volume_type = "gp3"
  }
  tags          = {
    OS_Version = "SUSE Linux Enterprise Server 15 SP3"
    Release    = "8"
    Arch       = "ARM64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

build {
  name = "e2e runners AMIs"
  sources = [
    "source.amazon-ebs.ubuntu",
    "source.amazon-ebs.debian-10-amd64",
    "source.amazon-ebs.debian-10-arm64",
    "source.amazon-ebs.debian-11-amd64",
    "source.amazon-ebs.centos-8-amd64",
    "source.amazon-ebs.centos-8-arm64",
    "source.amazon-ebs.oracle-linux-8",
    "source.amazon-ebs.sles15"
  ]

  provisioner "ansible" {
    user = build.User
    ansible_env_vars  =  ["PACKER_BUILD_NAME={{ build_name }}"]
    playbook_file     = ".ci/ansible/playbook.yml"
    extra_arguments   = ["--tags", "setup-ami"]
    galaxy_file       = ".ci/ansible/requirements.yml"
  }
}