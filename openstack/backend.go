package openstack

import (
	"context"
	"os"
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

	if b.client != nil {
		b.lock.Unlock()
		return b.client, nil
	}

	b.lock.Unlock()
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.client != nil {
		return b.client, nil
	}

	config, err := b.getConfig(ctx, s)
	if err != nil {
		return nil, err
	}

	if b.clientOpts == nil {
		if config == nil {
			config = new(osConfig)
		}

		clientOpts := b.genClientOpts(config)
		b.clientOpts = clientOpts
	}

	if config == nil {
		return nil, errEmptyConfig
	}

	ao, err := clientconfig.AuthOptions(b.clientOpts)
	if err != nil {
		return nil, err
	}

	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	err = openstack.Authenticate(provider, *ao)
	if err != nil {
		return nil, err
	}

	client, err := openstack.AuthenticatedClient(*ao)
	if err != nil {
		return nil, err
	}
	b.client = client

	return client, nil
}

func (b *backend) genClientOpts(config *osConfig) *clientconfig.ClientOpts {
	firstAvailable := func(opts ...string) string {
		for _, s := range opts {
			if s != "" {
				return s
			}
		}
		return ""
	}

	clientOpts := new(clientconfig.ClientOpts)
	clientOpts.AuthInfo.AuthURL = firstAvailable(os.Getenv("OS_AUTH_URL"), config.AuthURL)
	clientOpts.AuthInfo.Password = firstAvailable(os.Getenv("OS_PASSWORD"), config.Password)
	clientOpts.AuthInfo.DomainName = firstAvailable(os.Getenv("OS_DOMAIN_NAME"), config.DomainName)
	clientOpts.AuthInfo.ProjectName = firstAvailable(os.Getenv("OS_PROJECT_NAME"), config.ProjectName)
	clientOpts.AuthInfo.Username = firstAvailable(os.Getenv("OS_USERNAME"), config.Username)
	clientOpts.RegionName = firstAvailable(os.Getenv("OS_REGION_NAME"), config.Region)

	return clientOpts
}
