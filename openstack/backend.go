package openstack

import (
	"context"
	"sync"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const backendHelp = "OpenStack Token Backend"

const backendSecretType = "openstack_token"

type backend struct {
	*framework.Backend

	client     *gophercloud.ProviderClient
	clientOpts *clientconfig.ClientOpts

	lock sync.Mutex
}

func Factory(_ context.Context, _ *logical.BackendConfig) (logical.Backend, error) {
	b := new(backend)
	b.Backend = &framework.Backend{
		Help: backendHelp,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				infoPattern,
			},
			SealWrapStorage: []string{
				pathConfig,
			},
		},
		Paths: []*framework.Path{
			pathInfo,
			b.pathConfig(),
			b.pathToken(),
		},
		Secrets: []*framework.Secret{
			{
				Type: backendSecretType,
				Fields: map[string]*framework.FieldSchema{
					"token": {
						Type:        framework.TypeString,
						Description: "OpenStack token.",
					},
				},
			},
		},
		BackendType: logical.TypeLogical,
		Invalidate:  b.invalidate,
	}
	return b, nil
}

func (b *backend) reset() {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.client = nil
	b.clientOpts = nil
}

func (b *backend) invalidate(_ context.Context, key string) {
	switch key {
	case "config":
		b.reset()
	}
}
