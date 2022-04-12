package openstack

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	rotateHelpSyn  = "Rotate root cloud password."
	rotateHelpDesc = `
Rotate the cloud's root user credentials.

Once this method is called, Vault will now be the only entity that knows the password used to access OpenStack instance.
`
)

var (
	pathRotateRoot = fmt.Sprintf("rotate-root/%s", framework.GenericNameRegex("cloud"))
)

func (b *backend) pathRotateRoot() *framework.Path {
	return &framework.Path{
		Pattern: pathRotateRoot,
		Fields: map[string]*framework.FieldSchema{
			"cloud": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "Specifies name of the cloud which credentials will be rotated.",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.rotateRootCredentials,
			},
		},
		HelpSynopsis:    rotateHelpSyn,
		HelpDescription: rotateHelpDesc,
	}
}

func (b *backend) rotateRootCredentials(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	cloudName := d.Get("cloud").(string)

	sharedCloud := b.getSharedCloud(cloudName)
	client, err := sharedCloud.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	user, err := tokens.Get(client, client.Token()).ExtractUser()
	if err != nil {
		return nil, err
	}

	cloudConfig, err := sharedCloud.getCloudConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	newPassword, err := sharedCloud.passwords.Generate(ctx)
	if err != nil {
		return nil, err
	}

	// make sure we don't use this cloud until the password is changed
	sharedCloud.lock.Lock()
	defer sharedCloud.lock.Unlock()

	err = users.ChangePassword(client, user.ID, users.ChangePasswordOpts{
		Password:         newPassword,
		OriginalPassword: cloudConfig.Password,
	}).ExtractErr()
	if err != nil {
		return nil, err
	}
	cloudConfig.Password = newPassword

	if err := cloudConfig.save(ctx, req.Storage); err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}
