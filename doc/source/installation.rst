.. _installation:

Installation
============

All Vault auth methods and secrets engines are considered plugins. In order to
add plugin to Vault it is required to place compiled binary to the configured
location. After that the plugin must be registered and properly configured.

Compiled binary for the plugin can be either downloaded from official builds in
the repository or it can be built from sources.

.. code-block:: console

   $ wget https://github.com/opentelekomcloud/vault-plugin-secrets-openstack/releases/download/v1.0.1/vault-plugin-secrets-openstack_1.1.2_linux_arm64.tar.gz
   $ tar xvf vault-plugin-secrets-openstack_1.1.2_linux_amd64.tar.gz -C /etc/vault/plugins

Once the plugin is unpacked into the location expected by Vault the server
should be restarted.

.. code-block:: console

   $ service vault restart

After that it is possible to register the plugin and proceed with the
configuration.

.. code-block:: console

   $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/vault-plugin-secrets-openstack" | cut -d' ' -f1)
   $ vault plugin register -sha256="${SHA256}" secret vault-plugin-secrets-openstack

Now we need to enable the plugin.

.. code-block:: console

   $ vault secrets enable \
      -path="openstack" \
      -plugin-name="vault-plugin-secrets-openstack" plugin

   Success! Enabled the vault-plugin-secrets-openstack secrets engine at: openstack/


