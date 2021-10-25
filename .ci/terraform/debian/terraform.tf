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
      image = "debian-cloud/debian-9"
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
      "sudo apt-get update",
      "sudo apt-get -qyf install rsync wget gcc make",
      "wget https://dl.google.com/go/go1.16.3.linux-amd64.tar.gz",
      "sudo tar -C /usr/local -xf go1.16.3.linux-amd64.tar.gz",
      "mkdir -p /home/${var.user}/e2e-testing",
    ]
  }

 provisioner "local-exec" {
   command = "cd ${var.workspace} && rsync -avz --exclude='.git/' --include='.ci/' -e \"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${var.privatekeypath}\" ./ ci@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}:/home/${var.user}/e2e-testing"
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
