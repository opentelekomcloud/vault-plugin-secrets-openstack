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
	errUnableToRevokeToken = errors.New("unable to revoke token")

	errUnableToCreateClient = errors.New("unable to create IdentityV3 client")
)

func secretToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretType,
		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "OpenStack token.",
			},
		},
		Revoke: b.tokenRevoke,
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
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnableToCreateClient, err)
	}

	tokensOpts := &tokens.AuthOptions{
		Username:   b.clientOpts.AuthInfo.Username,
		Password:   b.clientOpts.AuthInfo.Password,
		DomainName: b.clientOpts.AuthInfo.DomainName,
		Scope: tokens.Scope{
			DomainName:  b.clientOpts.AuthInfo.DomainName,
			ProjectName: b.clientOpts.AuthInfo.ProjectName,
		},
	}

	token, err := tokens.Create(client, tokensOpts).ExtractToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnableToCreateToken, err)
	}

	resData := &logical.Response{
		Data: map[string]interface{}{
			"token":      token.ID,
			"expires_at": token.ExpiresAt.String(),
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

func (b *backend) tokenRevoke(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	tokenRaw, ok := r.Secret.InternalData["token"]
	if !ok {
		return nil, errors.New("internal data 'token' not found")
	}

	token := tokenRaw.(string)

	provider, err := b.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: b.clientOpts.RegionName,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errUnableToCreateClient, err)
	}

	if err := tokens.Revoke(client, token).Err; err != nil {
		return nil, fmt.Errorf("%w: %v", errUnableToRevokeToken, err)
	}

	return &logical.Response{}, nil
}
