package openstack

import (
	"context"
	"sync"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
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
			secretToken(b),
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

func (b *backend) getClient(ctx context.Context, s logical.Storage) (*gophercloud.ProviderClient, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.client != nil {
		b.lock.Unlock()
		return b.client, nil
	}

	configOpts, err := b.getClientOpts(ctx, s)
	if err != nil {
		return nil, err
	}

	ao, err := clientconfig.AuthOptions(configOpts)
	if err != nil {
		return nil, err
	}

	if err = b.setClientOpts(ctx, s, ao); err != nil {
		return nil, err
	}

	client, err := openstack.AuthenticatedClient(*ao)
	if err != nil {
		return nil, err
	}
	b.client = client

	return client, nil
}

func (b *backend) getClientOpts(ctx context.Context, s logical.Storage) (*clientconfig.ClientOpts, error) {
	config, err := b.getConfig(ctx, s)
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = new(osConfig)
	}

	clientOpts := &clientconfig.ClientOpts{
		AuthInfo: &clientconfig.AuthInfo{
			AuthURL:     config.AuthURL,
			Username:    config.Username,
			Password:    config.Password,
			ProjectName: config.ProjectName,
			DomainName:  config.DomainName,
		},
		RegionName: config.Region,
	}

	return clientOpts, nil
}

func (b *backend) setClientOpts(ctx context.Context, s logical.Storage, ao *gophercloud.AuthOptions) error {
	config, err := b.getConfig(ctx, s)
	if err != nil {
		return err
	}

	if config == nil {
		config = new(osConfig)
	}

	clientOpts := &clientconfig.ClientOpts{
		AuthInfo: &clientconfig.AuthInfo{
			AuthURL:     firstAvailable(config.AuthURL, ao.IdentityEndpoint),
			Username:    firstAvailable(config.Username, ao.Username),
			Password:    firstAvailable(config.Password, ao.Password),
			ProjectName: firstAvailable(config.ProjectName, ao.TenantName),
			DomainName:  firstAvailable(config.DomainName, ao.DomainName),
		},
		RegionName: config.Region,
	}

	b.clientOpts = clientOpts

	return nil
}

func firstAvailable(opts ...string) string {
	for _, s := range opts {
		if s != "" {
			return s
		}
	}
	return ""
}
