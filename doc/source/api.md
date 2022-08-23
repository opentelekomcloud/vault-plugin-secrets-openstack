(api)=

# API description

## Configure Root Credentials

This endpoint configures the root credentials to communicate with OpenStack instance. If credentials already exist, this
will overwrite them.

| Method | Path                       |
|:-------|:---------------------------|
| `POST` | `/openstack/clouds/:cloud` |
| `PUT`  | `/openstack/clouds/:cloud` |

### Parameters

* `auth_url` `(string: <required>)` - URL of identity service authentication endpoint.

* `user_domain_name` `(string: <required>)` - Name of the domain of the root user.

* `username` `(string: <required>)` - OpenStack username of the root user.

* `password` `(string: <required>)` - OpenStack password of the root user.

* `username_template` `(string: "vault{{random 8 | lowercase}}")` - Template used for usernames
  of temporary users. For details on templating syntax please refer to
  [Username Templating](https://www.vaultproject.io/docs/concepts/username-templating). Additional
  fields available for the template are `.CloudName`, `.RoleName`.

* `password_policy` `(string: <optional>)` - Specifies a password policy name to use when creating dynamic credentials.
  Defaults to generating an alphanumeric password if not set. For details on password policies please refer
  to [Password Policies](https://www.vaultproject.io/docs/concepts/password-policies).

### Sample Payload

```json
{
  "auth_url": "https://example.com/v3/",
  "username": "admin",
  "password": "RcigTiYrJjVmEkrV71Cd",
  "user_domain_name": "Default",
  "username_template": "user-{{ .RoleName }}-{{ random 4 }}"
}
```

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/openstack/clouds/example-cloud
```

## Read Root Configuration

This endpoint allows you to read non-secure values that have been set in the `clouds/:cloud` endpoint.
In particular, the `password` parameter is never returned.

| Method | Path                       |
|:-------|:---------------------------|
| `GET`  | `/openstack/clouds/:cloud` |

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/clouds/example-cloud
```

### Sample Response

```json
{
  "auth_url": "https://example.com/v3/",
  "username": "admin",
  "user_domain_name": "Default",
  "username_template": "user-{{ .RoleName }}-{{ random 4 }}"
}
```

## List Clouds

This endpoint allows you to list clouds values that have been configured in the `clouds` endpoint.

| Method | Path                 |
|:-------|:---------------------|
| `LIST` | `/openstack/clouds`  |
| `GET`  | `/openstack/clouds`  |

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST
    http://127.0.0.1:8200/v1/openstack/clouds
```

### Sample Response

```json
{
  "data": {
    "keys": [
      "sample-cloud-1",
      "sample-cloud-2"
    ]
  }
}
```

## Rotate Root Credentials

When you have configured Vault with static credentials, you can use this endpoint to have the Vault rotate the password
it used. Password change will be performed and new token will be returned.

Once this method is called, Vault will now be the only entity that knows the password used to access OpenStack instance.

| Method | Path                            |
|:-------|:--------------------------------|
| `POST` | `/openstack/rotate-root/:cloud` |
| `PUT`  | `/openstack/rotate-root/:cloud` |

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

| Method | Path                     |
|:-------|:-------------------------|
| `POST` | `/openstack/roles/:name` |
| `PUT`  | `/openstack/roles/:name` |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to create. This is part of the request URL.

- `cloud` `(string: <required>)` - Specifies root configuration of the created role.

- `root` `(bool: <optional>)` - Specifies whenever to use the root user as a role actor.
  If set to `true`, `secret_type` can't be set to `password`.
  If set to `true`, `user_groups` value is ignored.
  if set to `true`, `user_roles` value is ignored.
  If set to `true`, `ttl` value is ignored.

- `ttl` `(string: "1h")` - Specifies TTL value for the dynamically created users as a
  string duration with time suffix.

- `secret_type` `(string: "token")` - Specifies what kind of secret will configuration contain.
  Valid choices are `token` and `password`.

- `user_groups` `(list: [])` - Specifies list of existing OpenStack groups this Vault role is allowed to assume.
  This is a comma-separated string or JSON array.

- `user_roles` `(list: [])` - Specifies list of existing OpenStack roles this Vault role is allowed to assume.
  This is a comma-separated string or JSON array.

- `project_id` `(string: <optional>)` - Create a project-scoped role with given project ID. Mutually exclusive with
  `project_name`.

- `project_name` `(string: <optional>)` - Create a project-scoped role with given project name. Mutually exclusive with
  `project_id`.

- `domain_id` `(string: <optional>)` - Create a domain-scoped role with given domain ID. Mutually exclusive with
  `domain_name`.

- `domain_name` `(string: <optional>)` - Create a domain-scoped role with given domain name. Mutually exclusive with
  `domain_id`.

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
    http://127.0.0.1:8200/v1/openstack/roles/example-role
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

| Method   | Path                     |
|:---------|:-------------------------|
| `GET`    | `/openstack/roles/:name` |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to read. This is part of the request URL.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/roles/example-role
```

### Sample Response

```json
{
  "cloud": "example-cloud",
  "root": false,
  "secret_type": "password",
  "project_name": "test",
  "domain_name": "test",
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
| `GET`  | `/openstack/roles` |

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
    "keys": [
      "default-cloud-role-1",
      "default-cloud-role-2"
    ]
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
    "auth": {
      "auth_url": "https://example.com/v3/",
      "token": "gAAAAABiA6Xfybumdwd84qvMDJKYOaauWxSvG9ItslSr5w0Mb...",
      "project_name": "test",
      "project_domain_id": "Default"
    },
    "auth_type": "token"
  }
}
```

#### Credentials for the password-type role with project scope

```json
{
  "data": {
    "auth": {
      "auth_url": "https://example.com/v3/",
      "username": "admin",
      "password": "RcigTiYrJjVmEkrV71Cd",
      "project_name": "test",
      "project_domain_id": "Default"
    },
    "auth_type": "password"
  }
}
```

#### Credentials for the password-type role with domain scope

```json
{
  "data": {
    "auth": {
      "auth_url": "https://example.com/v3/",
      "username": "admin",
      "password": "RcigTiYrJjVmEkrV71Cd",
      "user_domain_id": "Default"
    },
    "auth_type": "password"
  }
}
```

## Create/Update Static Role

This endpoint creates or updates the static role with the given `name`. If a role with the name does not exist, it will be
created. If the role exists, it will be updated with the new attributes.

| Method | Path                            |
|:-------|:--------------------------------|
| `POST` | `/openstack/static-roles/:name` |
| `PUT`  | `/openstack/static-roles/:name` |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the static role to create. This is part of the request URL.

- `cloud` `(string: <required>)` - Specifies root configuration of the created role.

- `username` `(string: <required>)` - Specifies username of user managed by the static role.

- `rotation_duration` `(string: "1h")` - Specifies password rotation time value for the static user as a
  string duration with time suffix.

- `secret_type` `(string: "token")` - Specifies what kind of secret will configuration contain.
  Valid choices are `token` and `password`.

- `project_id` `(string: <optional>)` - Create a project-scoped role with given project ID. Mutually exclusive with
  `project_name`.

- `project_name` `(string: <optional>)` - Create a project-scoped role with given project name. Mutually exclusive with
  `project_id`.

- `domain_id` `(string: <optional>)` - Create a domain-scoped role with given domain ID. Mutually exclusive with
  `domain_name`.

- `domain_name` `(string: <optional>)` - Create a domain-scoped role with given domain name. Mutually exclusive with
  `domain_id`.

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
    http://127.0.0.1:8200/v1/openstack/static-roles/example-role
```

### Sample Payload

#### Creating a static role with project scope

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "username": "test-user"
}
```

#### Creating a static role for password-based access

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "secret_type": "password",
  "username": "test-user"
}
```

#### Creating a static role with endpoint override

```json
{
  "cloud": "example-cloud",
  "project_name": "test",
  "username": "test-user",
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
  "username": "test-user",
  "extensions": {
    "volume_api_version": 3,
    "object_store_endpoint_override": "https://swift.example.com"
  }
}
```

## List Static Roles

This endpoint queries an existing static role by the given name. If the role does not exist, a 404 is returned.

| Method | Path                      |
|:-------|:--------------------------|
| `LIST` | `/openstack/static-roles` |
| `GET`  | `/openstack/static-roles` |

### Parameters

- `cloud` `(string: <optional>)` – Specifies the name of the role to read. This is part of the request URL.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request LIST
    --data @payload.json
    http://127.0.0.1:8200/v1/openstack/static-roles
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
    "keys": [
      "default-cloud-role-1",
      "default-cloud-role-2"
    ]
  }
}
```

## Read Static Role Credentials

This endpoint returns user credentials based on the named static role.

| Method   | Path                            |
|:---------|:--------------------------------|
| `GET`    | `/openstack/static-creds/:name` |

### Parameters

- `name` (`string: <required>`) - Specifies the name of the role to return credentials against.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/static-creds/example-role
```

### Sample Responses

#### Credentials for the token-type static role

```json
{
  "data": {
    "auth": {
      "auth_url": "https://example.com/v3/",
      "token": "gAAAAABiA6Xfybumdwd84qvMDJKYOaauWxSvG9ItslSr5w0Mb...",
      "project_name": "test",
      "project_domain_id": "Default"
    },
    "auth_type": "token"
  }
}
```

#### Credentials for the password-type static role with project scope

```json
{
  "data": {
    "auth": {
      "auth_url": "https://example.com/v3/",
      "username": "admin",
      "password": "RcigTiYrJjVmEkrV71Cd",
      "project_name": "test",
      "project_domain_id": "Default"
    },
    "auth_type": "password"
  }
}
```

#### Credentials for the password-type static role with domain scope

```json
{
  "data": {
    "auth": {
      "auth_url": "https://example.com/v3/",
      "username": "admin",
      "password": "RcigTiYrJjVmEkrV71Cd",
      "user_domain_id": "Default"
    },
    "auth_type": "password"
  }
}
```

## Rotate Static Role Credentials

When you have configured Vault with static role, you can use this endpoint to have the Vault rotate the password
for the static user. Password change will be performed.

Once this method is called, password for static user related to static role will be updated.

| Method | Path                           |
|:-------|:-------------------------------|
| `POST` | `/openstack/rotate-role/:name` |
| `PUT`  | `/openstack/rotate-role/:name` |

### Sample Request

```shell
$ curl \
  --header "X-Vault-Token: ..." \
  --request POST \
  http://127.0.0.1:8200/v1/openstack/rotate-role/:name
```
