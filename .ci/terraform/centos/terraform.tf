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
      "echo \"PATH=$PATH:/usr/local/go/bin\" | sudo tee -a /etc/profile",
      "mkdir -p /home/${var.user}/e2e-testing",
    ]
  }

 provisioner "local-exec" {
   command = "rsync -avz --exclude='.git/' -e \"ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${var.workspace}/e2essh\" ${var.workspace}/* ci@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}:/home/${var.user}/e2e-testing/"
  }
}
