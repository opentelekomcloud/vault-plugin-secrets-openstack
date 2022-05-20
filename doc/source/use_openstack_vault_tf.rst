Get OSC credentials with Vault
==============================

When you have configured ``vault`` you should be able to get credentials
and pass them to ``openstack`` provider.

``settings.tf`` file with ``vault`` and ``openstack`` providers.

.. literalinclude:: terraform_vault_openstack/settings.tf
   :language: guess

``vault_os_creds.tf`` file where you get credentials from ``vault`` and use it.

.. literalinclude:: terraform_vault_openstack/vault_os_creds.tf
   :language: guess

``variables.tf`` file with input variables.

.. literalinclude:: terraform_vault_openstack/variables.tf
   :language: guess

With these files you can run ``terraform`` in order to start
infrastructure provisioning.

.. code-block:: console

   $ terraform apply

   ...
   Changes to Outputs:
   + image_id = "05d71131-e099-4ddd-f798-6acd4f9e3643"

   You can apply this plan to save these new output values to the Terraform state, without changing any real infrastructure.

   Apply complete! Resources: 0 added, 0 changed, 0 destroyed.

   Outputs:

   image_id = "05d71131-e099-4ddd-f798-6acd4f9e3643"

