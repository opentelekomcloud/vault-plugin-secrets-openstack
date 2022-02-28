# OpenStack Secrets Engine (API)

This is the API documentation for the Vault OpenStack secrets engine. For general information about the usage and
operation of the OpenStack secrets engine, please see the [Vault Openstack documentation](doc.md).

This documentation assumes the OpenStack secrets engine is enabled at the `/openstack` path in Vault. Since it is
possible to enable secrets engines at any location, please update your API calls accordingly.

## Configure Root Credentials

This endpoint configures the root credentials to communicate with OpenStack instance. If credentials already exist, this
will overwrite them.

| Method   | Path                       |
|:---------|:---------------------------|
| `POST`   | `/openstack/config/:cloud` |

### Parameters

* `auth_url` `(string: <required>)` - URL of identity service authentication endpoint.

* `user_domain_name` `(string: <required>)` - Name of the domain of the root user.

* `username` `(string: <required>)` - OpenStack username of the root user.

* `password` `(string: <required>)` - OpenStack password of the root user.

### Sample Payload

```json
{
  "auth_url": "https://example.com/v3/",
  "username": "admin",
  "password": "RcigTiYrJjVmEkrV71Cd",
  "user_domain_name": "Default"
}
```

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/openstack/config/example-cloud
```

## Read Root Configuration

This endpoint allows you to read non-secure values that have been configured in the `config/:cloud` endpoint.
In particular, the `password` parameter is never returned.

| Method | Path                       |
|:-------|:---------------------------|
| `GET`  | `/openstack/config/:cloud` |

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/config/example-cloud
```

### Sample Response

```json
{
  "auth_url": "https://example.com/v3/",
  "username": "admin",
  "user_domain_name": "Default"
}
```

## List Root Configurations

This endpoint allows you to list configurations values that have been configured in the `configs` endpoint.

| Method | Path                 |
|:-------|:---------------------|
| `LIST` | `/openstack/configs` |

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST
    http://127.0.0.1:8200/v1/openstack/configs
```

### Sample Response

```json
{
  "data": {
    "keys": ["sample-config-1", "sample-config-2"]  
  }
}
```

## Rotate Root Credentials

When you have configured Vault with static credentials, you can use this endpoint to have the Vault rotate the password
it used. Password change will be performed and new token will be returned.

Once this method is called, Vault will now be the only entity that knows the password used to access OpenStack instance.

| Method | Path                            |
|:-------|:--------------------------------|
| `GET`  | `/openstack/rotate-root/:cloud` |

### Sample Request

```shell
$ curl \
  --header "X-Vault-Token: ..." \
  --request POST \
  http://127.0.0.1:8200/v1/openstack/rotate-root/:cloud
```

## Create/Update Role

This endpoint creates or updates the role with the given `name`. If a role with the name does not exist, it will be
created. If the role exists, it will be updated with the new attributes.

| Method  | Path                    |
|:--------|:------------------------|
| `POST`  | `/openstack/role/:name` |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to create. This is part of the request URL.

- `cloud` `(string: <required>)` - Specifies root configuration of the created role.

- `root` `(bool: <optional>)` - Specifies whenever to use the root user as a role actor.
  If set to `true`, `secret_type` can't be set to `password`.
  If set to `true`, `user_groups` value is ignored.
  If set to `true`, `ttl` value is ignored.

- `ttl` `(string: "1h")` - Specifies TTL value for the dynamically created users as a
  string duration with time suffix.

- `secret_type` `(string: "token")` - Specifies what kind of secret will configuration contain.
  Valid choices are `token` and `password`.

- `user_groups` `(list: []`) - Specifies list of existing OpenStack groups this Vault role is allowed to assume.
  This is a comma-separated string or JSON array.

- `project_id` `(string: <optional>)` - Create a project-scoped role with given project ID. Mutually exclusive with
  `project_name`.

- `project_name` `(string: <optional>)` - Create a project-scoped role with given project name. Mutually exclusive with
  `project_id`.

When none of `project_name` or `project_id` is set, created role will have a project scope.

- `extensions` `(list: [])` - A list of strings representing a key/value pair to be used as extensions to the cloud
  configuration (e.g. `volume_api_version` or endpoint overrides). Format is a key and value
  separated by an `=` (e.g. `test_key=value`). Note: when using the CLI multiple tags
  can be specified in the role configuration by adding another `extensions` assignment
  in the same command.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/openstack/role/example-role
```

### Sample Payload

#### Creating a role with project scope

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "user_groups": [
    "default",
    "testing"
  ]
}
```

#### Creating a role using root user

```json
{
  "cloud": "example-cloud",
  "root": true,
  "project_name": "test"
}
```

#### Creating a role for password-based access

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "secret_type": "password",
  "user_groups": [
    "default",
    "testing"
  ]
}
```

#### Creating a role with endpoint override

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "user_groups": [
    "default",
    "testing"
  ],
  "extensions": [
    "volume_api_version=3",
    "object_store_endpoint_override=https://swift.example.com"
  ]
}
```

or 

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "user_groups": [
    "default",
    "testing"
  ],
  "extensions": {
    "volume_api_version": 3,
    "object_store_endpoint_override": "https://swift.example.com"
  }
}
```

## Read Role

This endpoint queries an existing role by the given name. If the role does not exist, a 404 is returned.

| Method   | Path                    |
|:---------|:------------------------|
| `GET`    | `/openstack/role/:name` |


### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to read. This is part of the request URL.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/role/example-role
```

### Sample Response

```json
{
  "cloud": "example-cloud",
  "root": false,
  "secret_type": "password",
  "project_name": "test",
  "user_groups": [
    "default",
    "testing"
  ],
  "ttl": "1h30m"
}
```

## List Roles

This endpoint queries an existing role by the given name. If the role does not exist, a 404 is returned.

| Method | Path               |
|:-------|:-------------------|
| `LIST` | `/openstack/roles` |


### Parameters

- `cloud` `(string: <optional>)` – Specifies the name of the role to read. This is part of the request URL.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST
    --data @payload.json
    http://127.0.0.1:8200/v1/openstack/roles
```

### Sample Payload

```json
{
  "cloud": "default-cloud"
}
```

### Sample Response

```json
{
  "data": {
    "keys": ["default-cloud-role-1", "default-cloud-role-2"]
  }
}
```

## Generate Credentials

This endpoint generates a new service credentials based on the named role.

| Method   | Path                     |
|:---------|:-------------------------|
| `GET`    | `/openstack/creds/:name` |

### Parameters

- `name` (`string: <required>`) - Specifies the name of the role to create credentials against.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/creds/example-role
```

### Sample Responses

#### Credentials for the token-type role

```json
{
  "data": {
    "auth_url": "https://example.com/v3/",
    "token": "gAAAAABiA6Xfybumdwd84qvMDJKYOaauWxSvG9ItslSr5w0Mb..."
  }
}
```

#### Credentials for the password-type role with project scope

```json
{
  "data": {
    "auth_url": "https://example.com/v3/",
    "username": "admin",
    "password": "RcigTiYrJjVmEkrV71Cd",
    "project_name": "test",
    "project_domain_name": "Default"
  }
}
```

#### Credentials for the password-type role with domain scope

```json
{
  "data": {
    "auth_url": "https://example.com/v3/",
    "username": "admin",
    "password": "RcigTiYrJjVmEkrV71Cd",
    "user_domain_name": "Default"
  }
}
```
