Get OpenStack credentials from Vault with Terraform
===================================================

This example demostrates how to get rid of passing OpenStack credentials to
Terraform from environment variables or configuration files. Insted credentials
are dynamically returned from Vault.

Prerequisites
-------------

This demo requires that there is Vault server up and running with OpenStack
Secrets plugin deployed and enabled. Further at least one account is configured
in Vault and corresponding role is created.

Preparation
-----------

In order to keep example simply it will only return image id for the named
passed as variable is going to be returned. As basic settings for Terraform
vault and OpenStack datasources are declared.

.. literalinclude:: terraform_vault_openstack/settings.tf
   :caption: settings.tf

Getting OpenStack authorization information from Vault together with a simple
datasource with output for retrieving image_id by its name can be implemented
in the following way:

.. literalinclude:: terraform_vault_openstack/vault_os_creds.tf
   :caption: vault_os_cred.tf

Input variables definition

.. literalinclude:: terraform_vault_openstack/variables.tf
   :caption: variables.tf

Invocation
----------

With these files it is possible to invoke ``terraform`` in order to start
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
