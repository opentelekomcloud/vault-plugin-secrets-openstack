variable "vault_public_addr" {
  description = "Address of the Vault Server"
  default     = "http://127.0.0.1:8200"
}

variable "project_name" {
  description = "OpenStack project name"
  default     = ""
}

variable "image_name" {
  description = "Image name in OSC"
}
