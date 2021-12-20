package openstack

import (
	"context"
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
		},
		HelpSynopsis: tokenHelpSyn,
	}
}

func (b *backend) pathTokenRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	config, err := b.getConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	client, err := IdentityV3Client(config)
	if err != nil {
		return nil, err
	}

	tokensOpts := &tokens.AuthOptions{
		IdentityEndpoint: config.AuthURL,
		Username:         config.Username,
		Password:         config.Password,
		DomainName:       config.DomainName,
	}

	if passcode, ok := d.GetOk("passcode"); ok {
		tokensOpts.Passcode = passcode.(string)
	}

	tok, err := tokens.Create(client, tokensOpts).ExtractToken()
	if err != nil {
		return nil, err
	}

	resData := &logical.Response{
		Data: map[string]interface{}{
			"token":      tok.ID,
			"expires_at": tok.ExpiresAt,
		},
	}

	resData.Secret = &logical.Secret{
		InternalData: map[string]interface{}{"secret_type": backendSecretType},
		LeaseOptions: logical.LeaseOptions{
			TTL: time.Until(tok.ExpiresAt),
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

	pc, err := openstack.AuthenticatedClient(options)
	if err != nil {
		return nil, err
	}

	return openstack.NewIdentityV3(pc, gophercloud.EndpointOpts{
		Region: config.Region,
	})
}
