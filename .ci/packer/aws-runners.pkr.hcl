packer {
  required_plugins {
    amazon = {
      version = ">= 1.1.5"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

variable "skip_create_ami" {
  type    = bool
  default = false
}

variable "source_set" {
  type    = string
  default = "linux"
}

variable "org_arn" {
  type    = string
  default = ""
}

variable "ami_suffix" {
  type    = string
  default = "test_suffix"
}

variable "galaxy_command" {
  type    = string
  default = "ansible-galaxy"
}

variable "command" {
  type    = string
  default = "ansible-playbook"
}

locals {
  aws_region       = "us-east-2"
  force_deregister = true
  source_sets = {
    "linux" = [
      "source.amazon-ebs.ubuntu",
      "source.amazon-ebs.debian-10-amd64",
      "source.amazon-ebs.debian-10-arm64",
      "source.amazon-ebs.debian-11-amd64",
      "source.amazon-ebs.centos-8-amd64",
      "source.amazon-ebs.centos-8-arm64",
      "source.amazon-ebs.oracle-linux-8",
      "source.amazon-ebs.sles15"
    ],
    "test"    = ["source.amazon-ebs.ubuntu"],
    "windows" = ["source.amazon-ebs.windows2019"],
    "all" = [
      "source.amazon-ebs.ubuntu",
      "source.amazon-ebs.debian-10-amd64",
      "source.amazon-ebs.debian-10-arm64",
      "source.amazon-ebs.debian-11-amd64",
      "source.amazon-ebs.centos-8-amd64",
      "source.amazon-ebs.centos-8-arm64",
      "source.amazon-ebs.oracle-linux-8",
      "source.amazon-ebs.sles15",
      "source.amazon-ebs.windows2019"
    ]
  }
  common_tags = {
    Division = "engineering"
    Org      = "obs"
    Team     = "observability-robots"
    Project  = "e2e-testing",
    Branch   = var.ami_suffix
  }
}

source "amazon-ebs" "ubuntu" {
  ami_name      = "ubuntu-2204-e2e-runner-${var.ami_suffix}"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0aeb7c931a5a61206"
  ssh_username  = "ubuntu"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Ubuntu"
      Release    = "22.04"
      Arch       = "AMD64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "debian-10-amd64" {
  ami_name      = "debian-10-amd64-runner-${var.ami_suffix}"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0d90bed76900e679a"
  ssh_username  = "admin"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Debian"
      Release    = "10"
      Arch       = "AMD64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "debian-10-arm64" {
  ami_name      = "debian-10-arm64-runner-${var.ami_suffix}"
  instance_type = "a1.large"
  region        = local.aws_region
  source_ami    = "ami-06dac44ad759182bd"
  ssh_username  = "admin"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Debian"
      Release    = "10"
      Arch       = "ARM64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "debian-11-amd64" {
  ami_name      = "debian-11-amd64-runner-${var.ami_suffix}"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0c7c4e3c6b4941f0f"
  ssh_username  = "admin"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Debian"
      Release    = "11"
      Arch       = "AMD64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "centos-8-amd64" {
  ami_name      = "centos-8-amd64-runner-${var.ami_suffix}"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-045b0a05944af45c1"
  ssh_username  = "centos"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Centos"
      Release    = "8"
      Arch       = "AMD64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "centos-8-arm64" {
  ami_name      = "centos-8-arm64-runner-${var.ami_suffix}"
  instance_type = "a1.large"
  region        = local.aws_region
  source_ami    = "ami-01cdc9e8306344fe0"
  ssh_username  = "centos"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Centos"
      Release    = "8"
      Arch       = "ARM64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

source "amazon-ebs" "oracle-linux-8" {
  ami_name      = "oracle-linux-8-x86-64-runner-${var.ami_suffix}"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-00371eeb8fd8e0e16"
  ssh_username  = "ec2-user"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "Oracle Linux"
      Release    = "8"
      Arch       = "x86-64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister

}

source "amazon-ebs" "sles15" {
  ami_name      = "sles15-runner-${var.ami_suffix}"
  instance_type = "t3.xlarge"
  region        = local.aws_region
  source_ami    = "ami-0f7cb53c916a75006"
  ssh_username  = "ec2-user"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 15
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = "${merge(
    local.common_tags,
    {
      OS_Version = "SUSE Linux Enterprise Server 15 SP3"
      Release    = "8"
      Arch       = "ARM64"
    }
  )}"
  ami_org_arns     = [var.org_arn]
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

build {
  name = "linux"

  sources = local.source_sets[var.source_set]

  provisioner "ansible" {
    user             = build.User
    ansible_env_vars = ["PACKER_BUILD_NAME={{ build_name }}"]
    playbook_file    = "ansible/playbook.yml"
    extra_arguments  = ["--tags", "setup-ami"]
    galaxy_file      = "ansible/requirements.yml"
    galaxy_command   = var.galaxy_command
    command          = var.command
  }
}

# Windows
source "amazon-ebs" "windows2019" {
  ami_name      = "windows-2019-runner-${var.ami_suffix}"
  instance_type = "c5.2xlarge"
  region        = local.aws_region
  source_ami    = "ami-0587bd602f1da2f1d"
  ssh_username  = "ogc"
  communicator  = "ssh"
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 60
    volume_type           = "gp3"
    delete_on_termination = true
  }
  tags = {
    OS_Version = "Windows"
    Release    = "2019"
    Arch       = "x86_64"
    Branch     = var.ami_suffix
    Project    = "e2e"
  }
  skip_create_ami  = var.skip_create_ami
  force_deregister = local.force_deregister
}

build {
  name = "windows"
  sources = [
    "source.amazon-ebs.windows2019"
  ]

  provisioner "ansible" {
    user             = build.User
    ansible_env_vars = ["PACKER_BUILD_NAME={{ build_name }}"]
    playbook_file    = "ansible/playbook.yml"
    extra_arguments  = ["--tags", "setup-ami", "--extra-vars", "nodeShellType=cmd"]
    galaxy_file      = "ansible/requirements.yml"
    galaxy_command   = var.galaxy_command
    command          = var.command
  }
}