resource "vault_generic_secret" "os_policy" {
  path = "sys/policies/password/os_policy"

  data_json = file("${path.cwd}/vault_data/os_policy.json")
}

resource "vault_generic_secret" "os_root" {
  path = "openstack/cloud/os_root"

  data_json = file("${path.cwd}/vault_data/os_root.json")

  depends_on = [
    vault_generic_secret.os_policy
  ]
}

resource "vault_generic_secret" "tmp_user_token" {
  path = "openstack/role/tmp_user_token"

  data_json = file("${path.cwd}/vault_data/tmp_user_token.json")

  depends_on = [
    vault_generic_secret.os_root
  ]
}

resource "vault_generic_secret" "root_token" {
  path = "openstack/role/root_token"

  data_json = file("${path.cwd}/vault_data/root_token.json")

  depends_on = [
    vault_generic_secret.os_root
  ]
}
