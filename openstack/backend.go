package openstack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	backendSecretTypeToken = "openstack_token"
	backendSecretTypeUser  = "openstack_user"
	backendSecretTypeRoot  = "openstack_root"
	backendHelp            = "OpenStack Token Backend"
)

type sharedCloud struct {
	name string

	client *gophercloud.ServiceClient
	lock   sync.Mutex

	passwords *Passwords
}

type backend struct {
	*framework.Backend

	clouds map[string]*sharedCloud
}

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
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
			secretRoot(b),
		},
		BackendType: logical.TypeLogical,
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *backend) getSharedCloud(name string) *sharedCloud {
	passwords := &Passwords{PolicyGenerator: b.System()}
	if c, ok := b.clouds[name]; ok {
		if c.passwords == nil {
			c.passwords = passwords
		}
		return c
	}
	cloud := &sharedCloud{name: name, passwords: passwords}
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

	opts := gophercloud.AuthOptions{
		IdentityEndpoint: cloud.AuthURL,
		Username:         cloud.Username,
		Password:         cloud.Password,
		DomainName:       cloud.UserDomainName,
		Scope: &gophercloud.AuthScope{
			DomainName: cloud.UserDomainName,
		},
	}

	pClient, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return fmt.Errorf("error creating provider client: %w", err)
	}

	sClient, err := openstack.NewIdentityV3(pClient, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("error creating service client: %w", err)
	}

	c.client = sClient

	return nil
}

type OsCloud struct {
	Name             string        `json:"name"`
	AuthURL          string        `json:"auth_url"`
	UserDomainName   string        `json:"user_domain_name"`
	Username         string        `json:"username"`
	Password         string        `json:"password"`
	UsernameTemplate string        `json:"username_template"`
	PasswordPolicy   string        `json:"password_policy"`
	PasswordExpire   time.Duration `json:"password_expire"`
	ValidateCloud    bool          `json:"validate_cloud"`
}
