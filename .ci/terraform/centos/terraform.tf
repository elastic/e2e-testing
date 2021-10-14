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
      "sudo yum -y install rsync wget gcc",
      "wget https://dl.google.com/go/go1.16.3.linux-amd64.tar.gz",
      "sudo tar -C /usr/local -xf go1.16.3.linux-amd64.tar.gz",
      "echo \"export PATH=$PATH:/usr/local/go/bin\" | sudo tee -a /etc/profile",
      "mkdir -p /home/${var.user}/e2e-testing",
    ]
  }

 provisioner "local-exec" {
   command = "rsync -avz --exclude='.git/' -e \"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${var.privatekeypath}\" ${var.workspace}/src/github.com/elastic/e2e-testing/* ci@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}:/home/${var.user}/e2e-testing"
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
      "echo \"export GOARCH=${var.goarch}\" | sudo tee -a /etc/profile",
      "echo \"export PROVIDER=${var.provider_type}\" | sudo tee -a /etc/profile",
      "echo \"export LOG_LEVEL=${var.log_level}\" | sudo tee -a /etc/profile",
      "echo \"export OP_LOG_LEVEL=${var.log_level}\" | sudo tee -a /etc/profile",
      "echo \"export KIBANA_URL=${var.kibana_url}\" | sudo tee -a /etc/profile",
      "echo \"export KIBANA_PASSWORD=${var.kibana_password}\" | sudo tee -a /etc/profile",
      "echo \"export ELASTICSEARCH_URL=${var.elasticsearch_url}\" | sudo tee -a /etc/profile",
      "echo \"export ELASTICSEARCH_PASSWORD=${var.elasticsearch_password}\" | sudo tee -a /etc/profile",
      "echo \"export FLEET_URL=${var.fleet_url}\" | sudo tee -a /etc/profile",
      "echo \"export SKIP_PULL=${var.skip_pull}\" | sudo tee -a /etc/profile",
    ]
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
     "ls -lhR /home/ci",
     "cd /home/ci/e2e-testing/_suites/${var.suite} && sudo -E go test -timeout 60m -v --godog.tags=\"${var.tags}\""
    ]
  }
}
