package openstack

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/hashicorp/go-multierror"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/common"
	"net/http"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
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

	client    *gophercloud.ServiceClient
	expiresAt time.Time
	lock      sync.Mutex

	passwords *Passwords
}

type backend struct {
	*framework.Backend
	clouds               map[string]*sharedCloud
	checkAutoRotateAfter time.Time
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
			b.pathStaticRoles(),
			b.pathStaticRole(),
			b.pathRotateRoot(),
			b.pathCreds(),
			b.pathRotateStaticCreds(),
			b.pathStaticCreds(),
		},
		Secrets: []*framework.Secret{
			secretToken(b),
			secretUser(b),
		},
		BackendType:  logical.TypeLogical,
		PeriodicFunc: b.periodicFunc,
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
		diff := time.Since(c.expiresAt)
		if diff.Seconds() <= -120 {
			return c.client, nil
		}
	}

	if err := c.initClient(ctx, s); err != nil {
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
		return fmt.Errorf("error creating provider client: %w", common.LogHttpError(err))
	}

	sClient, err := openstack.NewIdentityV3(pClient, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("error creating service client: %w", common.LogHttpError(err))
	}

	tokenResponse := tokens.Get(sClient, sClient.Token())
	token, err := tokenResponse.ExtractToken()
	if err != nil {
		return fmt.Errorf("error extracting token: %w", common.LogHttpError(err))
	}

	c.expiresAt = token.ExpiresAt
	c.client = sClient

	return nil
}

func (b *backend) periodicFunc(ctx context.Context, req *logical.Request) error {
	// Check for autorotation once an hour to avoid unnecessarily iterating
	// over all keys too frequently.
	if time.Now().Before(b.checkAutoRotateAfter) {
		return nil
	}
	b.Logger().Debug("periodic func", "rotate-root", "rotation cycle in progress")
	b.checkAutoRotateAfter = time.Now().Add(1 * time.Hour)

	return b.autoRotateKeys(ctx, req)
}

func (b *backend) autoRotateKeys(ctx context.Context, req *logical.Request) error {
	keys, err := req.Storage.List(ctx, "clouds/")
	if err != nil {
		return err
	}

	// Collect errors in a multierror to ensure a single failure doesn't prevent
	// all keys from being rotated.
	var errs *multierror.Error

	for _, key := range keys {
		cloudEntry := b.getSharedCloud(key)
		if cloudEntry == nil {
			continue
		}

		err = b.rotateIfRequired(ctx, req, cloudEntry)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	b.Logger().Debug("periodic func", "rotate-root", "rotation cycle complete")
	return errs.ErrorOrNil()
}

func (b *backend) rotateIfRequired(ctx context.Context, req *logical.Request, sCloud *sharedCloud) error {
	cloudConfig, err := sCloud.getCloudConfig(ctx, req.Storage)
	if err != nil {
		return err
	}
	if time.Now().After(cloudConfig.RootPasswordExpirationDate) {
		client, err := sCloud.getClient(ctx, req.Storage)
		if err != nil {
			return logical.CodedError(http.StatusConflict, common.LogHttpError(err).Error())
		}
		newPassword, err := sCloud.passwords.Generate(ctx)
		if err != nil {
			return err
		}

		// make sure we don't use this cloud until the password is changed
		sCloud.lock.Lock()
		defer sCloud.lock.Unlock()

		user, err := tokens.Get(client, client.Token()).ExtractUser()
		if err != nil {
			return logical.CodedError(http.StatusConflict, common.LogHttpError(err).Error())
		}
		err = users.ChangePassword(client, user.ID, users.ChangePasswordOpts{
			Password:         newPassword,
			OriginalPassword: cloudConfig.Password,
		}).ExtractErr()
		if err != nil {
			errorMessage := fmt.Sprintf("error changing root password: %s", common.LogHttpError(err).Error())
			return logical.CodedError(http.StatusConflict, errorMessage)
		}
		cloudConfig.Password = newPassword
		cloudConfig.RootPasswordExpirationDate = time.Now().Add(cloudConfig.RootPasswordTTL)

		if err := cloudConfig.save(ctx, req.Storage); err != nil {
			return err
		}
		b.Logger().Debug("password rotated", "cloud", cloudConfig.Name)
	}
	return nil
}
