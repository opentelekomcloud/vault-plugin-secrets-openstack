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

provider "openstack" {
  auth_url    = data.vault_generic_secret.token.data["auth_url"]
  token       = data.vault_generic_secret.token.data["token"]
  tenant_name = var.project_name
}
