package openstack

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

func Factory(context.Context, *logical.BackendConfig) (logical.Backend, error) {
	return nil, nil
}
