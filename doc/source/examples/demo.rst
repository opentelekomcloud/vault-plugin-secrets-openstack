Demo
====

This article demonstrates how to install vault, add and configure plugin and
perform invoke Terraform to query image_id for the image by name.

- Install vault

- Modify vault config file adding ``plugin_dir = "/opt/vault/plugin"``

- Deploy the plugin

  .. code-block:: console

     wget https://github.com/opentelekomcloud/vault-plugin-secrets-openstack/releases/download/v1.0.2/vault-plugin-secrets-openstack_1.0.2_linux_amd64.tar.gz
     tar xvf vault-plugin-secrets-openstack_1.0.2_linux_arm64.tar.gz -C /opt/vault/plugins

- Register the plugin

  .. code-block::

     vault secrets enable -path="openstack" -plugin-name="vault-plugin-secrets-openstack" plugin

- Register password policy

  .. code-block::
     :caption: os_policy.hcl

     length = 20
     rule "charset" {
       charset = "abcdefghijklmnopqrstuvwxyz"
       min-chars = 1
     }
     rule "charset" {
       charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
       min-chars = 1
     }
     rule "charset" {
       charset = "0123456789"
       min-chars = 1
     }
     rule "charset" {
       charset = "!@#$%^&*"
       min-chars = 1
     }

  .. code-block:: console

     vault write sys/policies/password/os-policy policy=@os_policy.hcl


- Configure cloud root account

  .. code-block::

     vault write openstack/cloud/demo auth_url=https://<AUTH_URL> username=<USER> password=<PASSWORD> user_domain_name=<USER_DOMAIN_NAME> password_policy=os-policy

- Configure root token role

  .. code-block:: console

     vault write /openstack/role/root_token cloud=demo project_name=<PROJECT_NAME> domain_name=<DOMAIN_NAME> root=true

- Prepare Terraform configuration

  .. literalinclude:: terraform_vault_openstack/settings.tf
     :caption: settings.tf

  .. literalinclude:: terraform_vault_openstack/vault_os_creds.tf
     :caption: vault_os_cred.tf

  .. literalinclude:: terraform_vault_openstack/variables.tf
     :caption: variables.tf

  It is required to populate tenant_name (project_name) of the OpenStack
  provider when root token role is used.

- Apply Terraform plan

  .. code-block:: console

     terraform apply
