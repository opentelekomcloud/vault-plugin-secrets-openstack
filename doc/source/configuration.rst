Configuration
=============

Currently it is possible to configure default policy used for the passwords
created by Vault plugin.

First it is required to create policy HCL or Json file:

.. code-block::

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

Then created file should be added to Vault:

.. code-block:: console

   $ vault write sys/policies/password/my-policy policy=@my-policy.hcl
