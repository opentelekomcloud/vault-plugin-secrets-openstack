OpenStack Cloud Secrets Engine plugin for HashiCorp Vault
=========================================================

A common but undesirable side effect of automation frameworks such as Terraform
or Ansible are passwords or other authentication credentials in plain text
inside of scripts and templates. They even often end up in repositories like
Git. On the one hand, secrets managed in this way quickly fall into the wrong
hands, and on the other hand, rotating them is also problematic and cumbersome.

HashiCorp Vault meets this challenge by storing and protecting these
credentials in a central, particularly secured place. Security administrators
then configure authorized clients for the respective access. It is an
identity-based secret and encryption management system. Sensitive data like API
encryption keys, passwords, certificates or anything else, what users want to
apply tight control access to, can be stored as secrets in Vault. There are
already connectors for many frameworks and environments. This project
introduces a plugin for managing OpenStack credentials.

The plugin addresses several typical use cases related to automation, user, and
trust management.

Infrastructure Automation
-------------------------

To get rid of plaintext credentials in IaC scripts and templates, devops
configure domain admin accounts in Vault. The IaC scripts fetch temporary
tokens for each cloud API operation instead of reading plaintext passwords from
configuration files. Once the script completes, the API token is revoked. This
scenario reduces the threat of leaking administrative account passwords in log
files dramatically. If a token leaks, it expires quickly. Even if attackers
have captured a token, they may not change any password associated with it.

Functional tests
----------------

Realistic test scenarios require short-living credentials that are valid only
for performing the test cases. For that, a domain administrative account is
configured in Vault. This account has privileges to create users. This is
necessary to verify the availability and correctness of cloud or application
APIs. For that, the software application test suite requests a temporary user
from Vault and performs the tests. Once they are completed, the user is
dropped.

Application with cloud API access
---------------------------------

Real-world application setups require several authorizations at once. Think of
an application server which reads from a database, writes to a log server, and
manages resources via a cloud API. For this use case, a regular cloud account
is configured in Vault. The HashiCorp Agent tool establishes the trust between
the application and Vault. The tool presents its Vault "ApplicationRole"
credentials to either other applications or to the infrastructure. Applications
can now obtain tokens to access the cloud API. They just read them from
preconfigured profiles stored in the Vault store. Password rotation of users
does no longer affect operations.

.. toctree::
   :hidden:

   installation
   configuration
   usage
   api.md
   examples/index
