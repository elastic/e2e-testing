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
  tags          = {
    OS_Version = "Centos"
    Release    = "8"
    Arch       = "ARM64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "windows2019" {
  ami_name        = "windows-2019-runner-1"
  instance_type   = "c5.2xlarge"
  region          = local.aws_region
  source_ami      = "ami-0587bd602f1da2f1d"
  winrm_username  = "ogc"
  winrm_insecure  = true
  winrm_use_ssl   = true
  communicator    = "winrm"
  user_data_file  = ".ci/packer/scripts/winrm_bootstrap.txt"
  tags            = {
    OS_Version  = "Windows"
    Release     = "9"
    Arch        = "AMD64"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

build {
  name = "e2e runners AMIs"
  sources = [
#    "source.amazon-ebs.ubuntu",
#    "source.amazon-ebs.debian-10-amd64",
#    "source.amazon-ebs.debian-10-arm64",
#    "source.amazon-ebs.debian-11-amd64",
    "source.amazon-ebs.centos-8-amd64",
    "source.amazon-ebs.centos-8-arm64"
#    "source.amazon-ebs.windows2019"

  ]

  provisioner "ansible" {
    user = build.User
    ansible_env_vars  =  ["PACKER_BUILD_NAME={{ build_name }}"]
    playbook_file     = ".ci/ansible/playbook.yml"
    extra_arguments   = ["--tags", "setup-ami"]
    galaxy_file       = ".ci/ansible/requirements.yml"
  }
}