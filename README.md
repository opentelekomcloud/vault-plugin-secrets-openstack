# Vault Plugin: OpenStack Secrets Backend [![Build Status](https://zuul.otc-service.com/api/tenant/eco/badge?project=opentelekomcloud/vault-plugin-secrets-openstack&pipeline=gate)](https://zuul.otc-service.com/t/eco/builds?project=opentelekomcloud%2Fvault-plugin-secrets-openstack&pipeline=gate)

This is a standalone backend plugin for use with [Hashicorp Vault](https://www.github.com/hashicorp/vault).
This plugin generates revocable, time-limited Tokens and Users for OpenStack.

## Quick Links
- [OpenStack Secrets API
  Docs](https://docs-beta.otc.t-systems.com/vault-plugin-secrets-openstack/api.html)
- [Vault Website](https://www.vaultproject.io)
- [Vault GitHub Project](https://www.github.com/hashicorp/vault)

## Getting Started

This is a [Vault plugin](https://www.vaultproject.io/docs/internals/plugins.html)
and is meant to work with Vault. This guide assumes you have already installed Vault
and have a basic understanding of how Vault works.

Otherwise, first read this guide on how to [get started with Vault](https://www.vaultproject.io/intro/getting-started/install.html).

To learn specifically about how plugins work, see documentation on [Vault plugins](https://www.vaultproject.io/docs/internals/plugins.html).

## Setup

The setup guide assumes some familiarity with Vault and Vault's plugin ecosystem. 
You must have a Vault server already running, unsealed, and authenticated.

1. Download and decompress the latest plugin binary from the Releases tab on
   GitHub. Alternatively you can compile the plugin from source.

1. Move the compiled plugin into Vault's configured [`plugin_directory`](https://www.vaultproject.io/docs/configuration/index.html#plugin_directory):

   ```sh
   $ mv vault-plugin-secrets-openstack /etc/vault/plugins/vault-plugin-secrets-openstack
   ```

1. Calculate the SHA256 of the plugin and register it in Vault's plugin catalog.
   If you are downloading the pre-compiled binary, it is highly recommended that
   you use the published checksums to verify integrity.

   ```sh
   $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/vault-plugin-secrets-openstack" | cut -d' ' -f1)
   
   $ vault write sys/plugins/catalog/vault-plugin-secrets-openstack \
       sha_256="${SHA256}" \
       command="vault-plugin-secrets-openstack"
       
   Success! Data written to: sys/plugins/catalog/vault-plugin-secrets-openstack
   ```

1. Mount secrets engine:

   ```sh
   $ vault secrets enable \
       -path="openstack" \
       -plugin-name="vault-plugin-secrets-openstack" plugin
       
   Success! Enabled the vault-plugin-secrets-openstack secrets engine at: openstack/
   ```

## Usage Guideline.

1. Firstly you have to define an admin credentials in a cloud.

   ```sh
   $ vault write /openstack/cloud/example-cloud auth_url=https://127.0.0.1/v3/ username=admin password=admin user_domain_name=mydomain
   Success! Data written to: openstack/cloud/example-cloud
   ```

   > **_NOTE:_** In the `cloud` you can specify additional parameters such as [`username_template`](https://www.vaultproject.io/docs/concepts/username-templating) 
   > and [`password_policy`](https://www.vaultproject.io/docs/concepts/password-policies) in order to follow conventions and security policies defined in your OpenStack cloud.

1. After that you need to create a role for the cloud.

   ```sh
   $ vault write /openstack/role/example-role cloud=example-cloud project_name=myproject domain_name=mydomain user_roles="member" root=false
   Success! Data written to: openstack/role/example-role
   ```

1. Now you can easily create a temporary user/token pair.

   ```sh
   $ vault read /openstack/creds/example-role

   Key                Value
   ---                -----
   lease_id           openstack/creds/example-role/v7I5mxix3iCpkdGYDelNmKdo
   lease_duration     24h
   lease_renewable    false
   auth               map[auth_url:https://127.0.0.1/v3/ project_domain_name:mydomain project_name:myproject token:MIIF-QYJKoZIhvcNAQcCoIIF7jCCBeoCAQExDTALBglghkgBZQMEAgEwggOCBgkqhkiG9w0BBwGgggNzBIIDb3sidG
   9rZW4iOnsiZXhwaXJlc19hdCI6IjIwMjItMDUtMjRUMTc6MTU6NDQuNTI4MDAwWiIsIm1ldGhvZHMiOlsicGFzc3dvcmQiXSwiY2F0YWxvZyI6W10sInJvbGVzIjpbeyJuYW1lIjoic2VydmVyX2FkbSIsImlkIjoiMCJ9LHsibmFtZSI6InRlX2FkbWluIiwiaWQiOiIwIn0seyJuYW1lIjoib3BfZ2F0ZWRfY2
   NlX3N3aXRjaCIsImlkIjoiMCJ9LHsibmFtZSIwiaWQiOiIwIn1dLCJwcm9qZWN0Ijp7ImRvbWFpbiI6eyJ4ZG9tYWluX3R5cGUiOiJUU0kiLCJuYW1lIjoiT1RDMDAwMDAwMDAwMDEwMDAwMDA0NDciLCJpZCI6ImZhZjcyZTg1MWMxZTQ4MzE5Mjk2ZjE4MTgxOGYxNzZhIiwieGRvbWFpbl9pZCI6IjAwMD
   AwMDAwMDAxMDAwMDAwNDQ3In0sIm5hbWUiOiJlCwYDVQQHDARCb25uMRwwGgYDVQQKDBNEZXV0c2NoZSBUZWxla29tIEFHMTcwNQYDVQQLDC5ULVN5c3RlbXMgUFUgUHVibGljIENsb3VkIC0gT3BlbiBUZWxla29tIENsb3VkMRMwEQYDVQQDDApjYS5pYW0ucGtpAg
   kAi0NlULiz2iIwCwYJYIZIAWUDBAIBMA0GCSqGSIb3DQEBAQUABIIBgBtmvHm0ChAl0kppzoaVJgYSovI0ujP6Srk7sDrDVO7auks+dQ-D1yb4Xh398mn1VNs0i4e98Bp-AQnwb3N-uZcUnbNHAUg2uRUoWUBmVs8oXszkGJYRK6Kf+BgGE5GqS4KgE2e8ob96xPGRQ5WdKMjYzb9nAz37sUsvy-jz70anyhKoWW
   B55kYVDyCn4JYh7-ivKrfaHdNY9GSebskw0zqAeMr2Lr4DZyZcr6mor88p1MnT+kz028qcxkrHBKiHQ0wt8YWUfAFD7RV-b97ZkBBSeEMs3ai1zMbYWXtIzztAWZLn52l+vstKiae8hXoE4HIpkMOD+fTyj2z6gN7X6xfvfBIZC2mSOQ2PAiTls2JTrcDVEo3eCI1vgvKrFdY52yDGmb3lHBITPPya0ztvjFrdNv59Gdx8-rxf1fr1htXRhyaMOBiaPDGVkUtjwqC9Zjm3fve5Z5A6UQurBm8xDjtRWjnFB5kAQ708adpCUHJAdxand3Mf-Md4UysLVSf4Ug==]
   auth_type          token

   ```

### Developing

If you wish to work on this plugin, you'll first need [Go](https://www.golang.org) installed on your machine (version 1.17+ is *required*).

For local dev first make sure Go is properly installed, including  setting up a [GOPATH](https://golang.org/doc/code.html#GOPATH).
Next, clone this repository into `$GOPATH/src/github.com/opentelekomcloud/vault-plugin-secrets-openstack`.

To compile a development version of this plugin, run `make` and `make install` commands.
This will put the plugin binary in the `bin` and `$GOPATH/bin` folders accordingly.

```sh
$ make
$ make install
```

Put the plugin binary into a location of your choice. This directory should be specified as the [`plugin_directory`](https://www.vaultproject.io/docs/configuration/index.html#plugin_directory) 
in the Vault config used to start the server.

```
...
plugin_directory = "path/to/plugin/directory"
...
```

Start a Vault server with this config file:

```sh
$ vault server -dev -dev-root-token-id=root -config=path/to/config.json
```

Once the server is started, register the plugin in the Vault server's [`plugin catalog`](https://www.vaultproject.io/docs/internals/plugins.html#plugin-catalog):

```sh
$ vault write sys/plugins/catalog/vault-plugin-secrets-openstack \
        sha256=<expected SHA256 Hex value of the plugin binary> \
        command="vault-plugin-secrets-openstack"

Success! Data written to: sys/plugins/catalog/vault-plugin-secrets-openstack
```

Note you should generate a new sha256 checksum if you have made changes
to the plugin. Example using openssl:

```sh
openssl dgst -sha256 $GOPATH/vault-plugin-secrets-openstack
...
SHA256(.../go/bin/vault-plugin-secrets-openstack)=896c13c0f2305daed381912a128322e02bc28a57d0c862a78cbc2ea66e8c6fa1
```

Enable the secrets' plugin backend using the secrets enable plugin command:

```sh
$ vault secrets enable -path=openstack -plugin-name=vault-plugin-secrets-openstack plugin
...
Successfully enabled the plugin-secrets-openstack secrets engine at: openstack/!
```

#### Tests

If you are developing this plugin and want to verify it is still
functioning (and you haven't broken anything else), we recommend
running the tests.

To run the tests, invoke `make test`:

```sh
$ make test
```

#### Acceptance Tests

Acceptance tests requires admin privileges in an OpenStack cloud.

```sh
$ export OS_CLIENT_CONFIG_FILE=<clouds.yaml path>
$ export OS_CLOUD=<cloud name>
$ make functional
```
