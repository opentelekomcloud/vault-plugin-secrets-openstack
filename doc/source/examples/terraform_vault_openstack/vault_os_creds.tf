data "vault_generic_secret" "token" {
  path = "openstack/creds/root_token"
}

data "openstack_images_image_v2" "this" {
  name        = var.image_name
  most_recent = true
}

output "image_id" {
  value = data.openstack_images_image_v2.this.id
}
