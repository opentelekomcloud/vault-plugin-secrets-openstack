package openstack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathToken = "token"

	tokenHelpSyn = `Create and return tokens a token using OpenStack secrets plugin.`
)

var (
	errUnableToCreateToken  = errors.New("unable to create token")
	errUnableToCreateClient = errors.New("unable to create OpenStack Identity client")
)

func (b *backend) pathToken() *framework.Path {
	return &framework.Path{
		Pattern: pathToken,
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathTokenRead,
			},
		},
		HelpSynopsis: tokenHelpSyn,
	}
}

func (b *backend) pathTokenRead(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	config, err := b.getConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return &logical.Response{}, errEmptyConfig
	}

	client, err := IdentityV3Client(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnableToCreateClient, err)
	}

	tokensOpts := &tokens.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.Password,
		DomainName:       config.DomainName,
	}

	token, err := tokens.Create(client, tokensOpts).ExtractToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnableToCreateToken, err)
	}

	resData := &logical.Response{
		Data: map[string]interface{}{
			"token":      token.ID,
			"expires_at": token.ExpiresAt,
		},
	}

	resData.Secret = &logical.Secret{
		InternalData: map[string]interface{}{
			"secret_type": backendSecretType,
		},
		LeaseOptions: logical.LeaseOptions{
			TTL: time.Until(token.ExpiresAt),
		},
	}

	return resData, nil
}

func IdentityV3Client(config *osConfig) (*gophercloud.ServiceClient, error) {
	options := gophercloud.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.Password,
		DomainName:       config.DomainName,
	}

	providerClient, err := openstack.AuthenticatedClient(options)
	if err != nil {
		return nil, err
	}

	return openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region: config.Region,
	})
}
