terraform {
  required_providers {
    vault = {
      source  = "hashicorp/vault"
      version = "3.6.0"
    }

    openstack = {
      source  = "terraform-provider-openstack/openstack"
      version = ">= 1.47.0"
    }
  }
}

provider "vault" {
  address = var.vault_public_addr
}

locals {
  auth = jsondecode(data.vault_generic_secret.token.data["auth"])
}

provider "openstack" {
  auth_url    = local.auth.auth_url
  token       = local.auth.token
  tenant_name = var.project_name
}
