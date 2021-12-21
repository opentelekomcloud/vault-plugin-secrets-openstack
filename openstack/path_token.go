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
	errUnableToCreateToken = errors.New("unable to create token")
)

func secretToken(_ *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretType,
		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "OpenStack token.",
			},
		},
	}

}

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
	provider, err := b.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: b.clientOpts.RegionName,
	})

	tokensOpts := &tokens.AuthOptions{
		IdentityEndpoint: b.clientOpts.AuthInfo.AuthURL,
		Username:         b.clientOpts.AuthInfo.Username,
		Password:         b.clientOpts.AuthInfo.Password,
		DomainName:       b.clientOpts.AuthInfo.DomainName,
		AllowReauth:      true,
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
