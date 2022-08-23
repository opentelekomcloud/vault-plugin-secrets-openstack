Usage
=====

Account configuration
---------------------

In order to start using the plugin to manage OpenStack credentials it is
required to register known cloud account into the plugin. Ideally this is some
form of an administrative user with the privilege to create other users. This
will give possibility to use dynamic roles for requesting a temporary user.

.. code-block:: console

   $ vault write /openstack/clouds/example-cloud auth_url=https://127.0.0.1/v3/ username=admin password=admin user_domain_name=mydomain username_template= vault{{random 8 | lowercase}} password_policy=my-policy
   Success! Data written to: openstack/cloud/example-cloud

Roles
-----

A role consists of a Vault managed OSC Service account along with a set of IAM
bindings defined for that service account. The name of the service account is
generated based on username_template field in the cloud path.

To configure a role that generates OSC Service Account token (preferred):

.. code-block:: console

   $ vault write /openstack/roles/role-tmp-user cloud=example-cloud project_name=myproject domain_name=mydomain user_groups=power-user root=false secret_type=token
   Success! Data written to: openstack/role/role-tmp-user

To configure a role that generates OSC Service Account password:

.. code-block:: console

   $ vault write /openstack/roles/role-tmp-user-pwd cloud=example-cloud project_name=myproject domain_name=mydomain user_groups=power-user root=false secret_type=password
   Success! Data written to: openstack/role/role-tmp-user-pwd

To configure a role that generates OSC root account token

.. code-block:: console

   $ vault write /openstack/roles/role-root-user cloud=example-cloud project_name=myproject domain_name=mydomain root=true
   Success! Data written to: openstack/role/role-root-user

After the secrets engine is configured and a user/machine has a Vault token
with the proper permission, it can generate credentials. Depending on how the
role was configured, you can generate OAuth2 tokens or service account keys.

Requesting Access Tokens
------------------------

To generate tokens, read from openstack/creds/.... The role must have been
created.

.. code-block:: console

   $ vault read /openstack/creds/role-tmp-user

   Key                Value
   ---                -----
   lease_id           openstack/creds/role-tmp-user/Humt41Qu8s1k5f4AZ8PUmDxE
   lease_duration     1h
   lease_renewable    false
   auth_url           https://127.0.0.1/v3/
   expires_at         2022-05-13 02:03:36 +0000 UTC
   token              gARAVABiXW-4r2Ofy4s4-oFlnbNgIrqHNkmIHPnE...

The token value then can be used as a HTTP Authorization token in requests to
OSC APIs:

.. code-block:: console

   $ curl -H "X-Auth-Token: gARAVABiXW-4r2Ofy4s4-oFlnbNgIrqHNkmIHPnE..." https://127.0.0.1/v3/endpoints


