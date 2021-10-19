provider "google" {
  project = "elastic-observability"
  region = "us-central1"
}

resource "random_id" "instance_id" {
  byte_length = 8
}

resource "google_compute_instance" "default" {
  name = "e2e-${random_id.instance_id.hex}"
  machine_type = "e2-standard-4"
  zone = "us-central1-c"
  boot_disk {
    initialize_params {
      image = "centos-cloud/centos-8"
    }
  }

  metadata = {
    ssh-keys = "${var.user}:${file(var.publickeypath)}"
  }

  network_interface {
    network = "default"
    access_config {
    }
  }

 provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "${var.user}"
      private_key = "${file(var.privatekeypath)}"
      host = google_compute_instance.default.network_interface.0.access_config.0.nat_ip
      agent = "false"
    }

    inline = [
      "sudo yum -y remove docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine || true",
      "sudo yum -y install yum-utils",
      "sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo",
      "sudo yum -y install docker-ce docker-ce-cli containerd.io rsync wget gcc make curl",
      "sudo systemctl start docker",
      "sudo curl -L \"https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)\" -o /usr/local/bin/docker-compose",
      "sudo chmod +x /usr/local/bin/docker-compose",
      "sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose",
      "mkdir -p /home/${var.user}/e2e-testing",
    ]
  }

 provisioner "local-exec" {
   command = "cd ${var.workspace}/${var.base_dir} && rsync -avz --exclude='.git/' --include='.ci/' -e \"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${var.privatekeypath}\" ./ ci@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}:/home/${var.user}/e2e-testing"
  }

 provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "${var.user}"
      private_key = "${file(var.privatekeypath)}"
      host = google_compute_instance.default.network_interface.0.access_config.0.nat_ip
      agent = "false"
    }

   inline = [
     "sudo docker-compose -f /home/${var.user}/e2e-testing/cli/config/compose/profiles/fleet/docker-compose.yml up -d",
    ]
 }
}
