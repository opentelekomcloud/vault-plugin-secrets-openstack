Configure OpenStack Secrets Engine With Terraform
=================================================

This example demonstrates how to configure the plugin with the help of
HashiCorp Terraform. It does not describe how to get OpenStack credentials from
Vault with Terraform.

Prerequisites
-------------

This demo requires that there is a Vault server up and running with OpenStack
Secrets plugin deployed and enabled. Please see :ref:`installation` for details
on this can be achieved.

Preparation
-----------

``settings.tf`` file with the ``vault`` provider.

.. literalinclude:: terraform_vault_configure/settings.tf
   :caption: settings.tf

``vault_configure.tf`` file with a minimal required configuration.

.. literalinclude:: terraform_vault_configure/vault_configure.tf
   :caption: vault_configure.tf

``variables.tf`` file with input variables

.. literalinclude:: terraform_vault_configure/variables.tf
   :caption: variables.tf

Json configuration description (see :ref:`api` for further details about supported parameters)

.. literalinclude:: terraform_vault_configure/vault_data/os_policy.json
   :caption: os_policy.json

.. literalinclude:: terraform_vault_configure/vault_data/os_root.json
   :caption: os_root.json

.. literalinclude:: terraform_vault_configure/vault_data/root_token.json
   :caption: root_token.json

.. literalinclude:: terraform_vault_configure/vault_data/tmp_user_token.json
   :caption: tmp_user_token.json

Invocation
----------

Now you can run the ``terraform`` and proceed with the configuration.

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
