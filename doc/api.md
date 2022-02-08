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

* `domain_name` `(string: <required>)` - Name of the domain of the root user.

* `username` `(string: <required>)` - OpenStack username of the root user

* `password` `(string: <required>)` - OpenStack password of the root user.

* `project_name` `(string: <optional>)` - Name of the OpenStack project the root user is used with.

### Sample Payload

```json
{
  "auth_url": "http://10.172.80.148:5000/v3/",
  "username": "admin",
  "password": "RcigTiYrJjVmEkrV71Cd",
  "domain_name": "Default"
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

## Read Root Credentials

This endpoint allows you to get existing root configuration. Note that it won't contain username, domain name or project
name, but only a generated token.

| Method | Path                     |
|:-------|:-------------------------|
| GET    | /openstack/config/:cloud |

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/config/example-cloud
```

### Sample Response

```json
{
  "data": {
    "auth_url": "http://10.172.80.148:5000/v3/",
    "token": "gAAAAABiA5N8qjR0UICb5S7-2lW-h-JRAzXKR2aRpliBHfb..."
  }
}
```

## Rotate Root Credentials

When you have configured Vault with static credentials, you can use this endpoint to have the Vault rotate the password
it used. Password change will be performed and new token will be returned.

Once this method is called, Vault will now be the only entity that knows the password used to access OpenStack instance.

| Method | Path                            |
|:-------|:--------------------------------|
| GET    | /openstack/config-rotate/:cloud |

### Sample Request

```shell
$ curl \
  --header "X-Vault-Token: ..." \
  --request POST \
  http://127.0.0.1:8200/v1/openstack/config-rotate/:cloud
```

### Sample Response

```json
{
  "data": {
    "auth_url": "http://10.172.80.148:5000/v3/",
    "token": "gAAAAABiA6Xfybumdwd84qvMDJKYOaauWxSvG9ItslSr5w0Mb..."
  }
}
```

## Configure Lease

This endpoint configures lease settings for the OpenStack secrets engine. It is optional, as there are default values
for `lease` and `lease_max`.

| Method   | Path                      |
|:---------|:--------------------------|
| `POST`   | `/openstack/lease/:cloud` |

### Parameters

- `lease` `(string: <required>)` – Specifies the lease value provided as a string duration with time suffix. "h" (hour)
  is the largest suffix.

- `lease_max` `(string: <required>)` – Specifies the maximum lease value provided as a string duration with time
  suffix. "h" (hour) is the largest suffix.

### Sample Payload

```json
{
  "lease": "30m",
  "lease_max": "12h"
}
```

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/openstack/lease/example-cloud
```

## Read Lease

This endpoint returns the current lease settings for the OpenStack secrets engine.

| Method    | Path                         |
|:----------|:-----------------------------|
| `GET`     | `/openstack/lease/:cloud`    |

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/lease/example-cloud
```

### Sample Response

```json
{
  "data": {
    "lease": "30m0s",
    "lease_max": "12h0m0s"
  }
}
```

## Create/Update Role

This endpoint creates or updates the role with the given `name`. If a role with the name does not exist, it will be
created. If the role exists, it will be updated with the new attributes.

| Method  | Path                       |
|:--------|:---------------------------|
| `POST`  | `/openstack/roles/:name`   |

### Parameters

- `name` `(string: <required>)` – Specifies the name of the role to create. This is part of the request URL.

- `user_groups` `(list: []`) - Specifies list of existing OpenStack groups this Vault role is allowed to assume.
  This is a comma-separated string or JSON array.

- `scope` `(string: <required>)` - Specifies role scope. Possible values are `domain`, `project`. 

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/openstack/roles/example-role
```

### Sample Payload

```json
{
  "scope": "project",
  "user_groups": [
    "default",
    "testing"
  ]
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
  "scope": "project",
  "user_groups": [
    "default",
    "testing"
  ]
}
```

## Generate Credentials

This endpoint generates a new service credentials based on the named role.

| Method   | Path                     |
|:---------|:-------------------------|
| `GET`    | `/openstack/issue/:name` |

### Parameters

- `name` (`string: <required>`) - Specifies the name of the role to create credentials against.

### Sample Request

```shell
$ curl \
    --header "X-Vault-Token: ..." \
    http://127.0.0.1:8200/v1/openstack/issue/example-role
```

### Sample Response

```json
{
  "data": {
    "auth_url": "http://10.172.80.148:5000/v3/",
    "token": "gAAAAABiA6Xfybumdwd84qvMDJKYOaauWxSvG9ItslSr5w0Mb..."
  }
}
```
