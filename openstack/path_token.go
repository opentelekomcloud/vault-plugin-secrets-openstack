package openstack

import (
	"context"
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

func (b *backend) pathToken() *framework.Path {
	return &framework.Path{
		Pattern: pathToken,
		Fields: map[string]*framework.FieldSchema{
			"passcode": {
				Type: framework.TypeString,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathTokenRead,
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathTokenWrite,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathTokenUpdate,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathTokenDelete,
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
	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.Password,
		DomainName:       config.DomainName,
	}

	authClient, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewIdentityV3(authClient, gophercloud.EndpointOpts{
		Region: config.Region,
	})
	if err != nil {
		return nil, err
	}

	tokensOpts := &tokens.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.Password,
		DomainName:       config.DomainName,
	}

	tok, err := tokens.Create(client, tokensOpts).ExtractToken()
	if err != nil {
		return nil, err
	}

}

func (b *backend) pathTokenWrite(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {

}
func (b *backend) pathTokenUpdate(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {

}
func (b *backend) pathTokenDelete(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {

}
