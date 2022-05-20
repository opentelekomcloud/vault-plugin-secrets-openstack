Configure OpenStack Secrets Engine With Terraform
=================================================

Prerequisite
------------

Before proceeding with the Vault configuration, you must have:

1. A virtual machine with an external address.

2. Vault server with enabled OpenStack secrets engine is up and running.

Configuration
-------------

After that it is possible to configure OpenStack plugin with the given
scripts.

``settings.tf`` file with the ``vault`` provider.

.. literalinclude:: terraform_vault_configure/settings.tf
   :language: guess

``vault_configure.tf`` file with a minimal required configuration.

.. literalinclude:: terraform_vault_configure/vault_configure.tf
   :language: guess

``variables.tf`` file with the ``vault`` address.

.. literalinclude:: terraform_vault_configure/variables.tf
   :language: guess

After that it is possible to run the ``terraform`` and proceed with the
configuration.

.. code-block:: console

   $ terraform apply

   ...
   Do you want to perform these actions?
     Terraform will perform the actions described above.
     Only 'yes' will be accepted to approve.

     Enter a value: yes

   vault_generic_secret.os_policy: Creating...
   vault_generic_secret.os_policy: Creation complete after 0s [id=sys/policies/password/os_policy]
   vault_generic_secret.os_root: Creating...
   vault_generic_secret.os_root: Creation complete after 0s [id=openstack/cloud/os_root]
   vault_generic_secret.tmp_user_token: Creating...
   vault_generic_secret.root_token: Creating...
   vault_generic_secret.root_token: Creation complete after 1s [id=openstack/role/root_token]
   vault_generic_secret.tmp_user_token: Creation complete after 1s [id=openstack/role/tmp_user_token]

   Apply complete! Resources: 4 added, 0 changed, 0 destroyed.

