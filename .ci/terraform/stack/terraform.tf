terraform {
  required_providers {
    ec = {
      source  = "elastic/ec"
      version = "0.2.1"
    }
  }
}

provider "ec" {
   endpoint = "https://staging.found.no/"
}

resource "ec_deployment" "end-to-end" {
  name                   = "terraform-demo"
  region                 = "gcp-us-central1"
  version                = "7.15.1"
  deployment_template_id = "gcp-io-optimized-v2"

  elasticsearch {}

  kibana {}

  apm {}
}
