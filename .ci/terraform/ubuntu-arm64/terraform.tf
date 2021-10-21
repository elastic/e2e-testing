terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = "us-east-2"
}

resource "random_id" "instance_id" {
  byte_length = 8
}

resource "aws_security_group" "instance" {
  name = "e2e-sg-${random_id.instance_id.hex}"
  description = "Security group for the aws instance"
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-arm64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["099720109477"] # Canonical
}

resource "aws_key_pair" "e2essh" {
  key_name = "e2essh"
  public_key = "${file(var.publickeypath)}"
}

resource "aws_instance" "default" {
  ami = data.aws_ami.ubuntu.id
  instance_type = "a1.large"
  associate_public_ip_address = true
  key_name = aws_key_pair.e2essh.key_name
  vpc_security_group_ids = [aws_security_group.instance.id]
  tags = {
    Name = "e2e-${random_id.instance_id.hex}"
  }

 provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "${var.user}"
      private_key = "${file(var.privatekeypath)}"
      host = aws_instance.default.public_ip
      agent = "false"
    }

    inline = [
      "sudo apt-get update",
      "sudo apt-get -qyf install rsync wget build-essential",
      "wget https://dl.google.com/go/go1.16.3.linux-amd64.tar.gz",
      "sudo tar -C /usr/local -xf go1.16.3.linux-amd64.tar.gz",
      "mkdir -p /home/${var.user}/e2e-testing",
    ]
  }

 provisioner "local-exec" {
   command = "cd ${var.workspace}/${var.base_dir} && rsync -avz --exclude='.git/' --include='.ci/' -e \"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${var.privatekeypath}\" ./ ci@${aws_instance.default.public_ip}:/home/${var.user}/e2e-testing"
  }

 provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "${var.user}"
      private_key = "${file(var.privatekeypath)}"
      host = aws_instance.default.public_ip
      agent = "false"
    }

   inline = [
     "touch /home/${var.user}/e2e-testing/.env || true",
      "echo \"export PATH=$PATH:/usr/local/go/bin\" | tee -a /home/ci/e2e-testing/.env",
      "echo \"export GOARCH=${var.goarch}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export PROVIDER=${var.provider_type}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export LOG_LEVEL=${var.log_level}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export OP_LOG_LEVEL=${var.log_level}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export KIBANA_URL=${var.kibana_url}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export KIBANA_PASSWORD=${var.kibana_password}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export ELASTICSEARCH_URL=${var.elasticsearch_url}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export ELASTICSEARCH_PASSWORD=${var.elasticsearch_password}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export FLEET_URL=${var.fleet_url}\" | tee -a /home/${var.user}/e2e-testing/.env",
      "echo \"export SKIP_PULL=${var.skip_pull}\" | tee -a /home/${var.user}/e2e-testing/.env",
    ]
 }
}
