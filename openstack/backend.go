package openstack

import (
	"context"
	"fmt"
	"sync"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	backendSecretTypeToken = "openstack_token"
	backendSecretTypeUser  = "openstack_user"
	backendHelp            = "OpenStack Token Backend"
)

type sharedCloud struct {
	name string

	client *gophercloud.ServiceClient
	lock   sync.Mutex
}

type backend struct {
	*framework.Backend

	clouds map[string]*sharedCloud
}

func Factory(_ context.Context, _ *logical.BackendConfig) (logical.Backend, error) {
	b := new(backend)
	b.Backend = &framework.Backend{
		Help: backendHelp,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				infoPattern,
			},
		},
		Paths: []*framework.Path{
			pathInfo,
			b.pathCloud(),
			b.pathClouds(),
			b.pathRole(),
			b.pathRoles(),
			b.pathRotateRoot(),
			b.pathCreds(),
		},
		Secrets: []*framework.Secret{
			secretToken(b),
			secretUser(b),
		},
		BackendType: logical.TypeLogical,
	}
	return b, nil
}

func (b *backend) getSharedCloud(name string) *sharedCloud {
	if c, ok := b.clouds[name]; ok {
		return c
	}
	cloud := &sharedCloud{name: name}
	if b.clouds == nil {
		b.clouds = make(map[string]*sharedCloud)
	}
	b.clouds[name] = cloud
	return cloud
}

// getClient returns initialized Keystone service client
func (c *sharedCloud) getClient(ctx context.Context, s logical.Storage) (*gophercloud.ServiceClient, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.client != nil {
		return c.client, nil
	}

	err := c.initClient(ctx, s)
	if err != nil {
		return nil, err
	}

	return c.client, nil
}

func (c *sharedCloud) initClient(ctx context.Context, s logical.Storage) error {
	cloud, err := c.getCloudConfig(ctx, s)
	if err != nil {
		return err
	}
	if cloud == nil { // this happened at least once during acceptance test
		return fmt.Errorf("no cloud found with name %s", c.name)
	}

	clientOpts := &clientconfig.ClientOpts{
		AuthInfo: &clientconfig.AuthInfo{
			AuthURL:        cloud.AuthURL,
			Username:       cloud.Username,
			Password:       cloud.Password,
			UserDomainName: cloud.UserDomainName,
		},
	}

	sClient, err := clientconfig.NewServiceClient("identity", clientOpts)
	if err != nil {
		return err
	}

	c.client = sClient

	return nil
}

type OsCloud struct {
	Name           string `json:"name"`
	AuthURL        string `json:"auth_url"`
	UserDomainName string `json:"user_domain_name"`
	Username       string `json:"username"`
	Password       string `json:"password"`
}
