package openstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathStaticCreds = "static-creds"

	staticCredsHelpSyn  = "Manage the Openstack static credentials with static roles."
	staticCredsHelpDesc = `
This path allows you to read OpenStack secret stored by predefined static roles.
`
)

func (b *backend) pathStaticCreds() *framework.Path {
	return &framework.Path{
		Pattern: fmt.Sprintf("%s/%s", pathStaticCreds, framework.GenericNameRegex("role")),
		Fields: map[string]*framework.FieldSchema{
			"role": {
				Type:        framework.TypeString,
				Description: "Name of the role.",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathStaticCredsRead,
			},
			logical.ListOperation: &framework.PathOperation{
				Callback: b.pathStaticCredsRead,
			},
		},
		HelpSynopsis:    staticCredsHelpSyn,
		HelpDescription: staticCredsHelpDesc,
	}
}

func roleToStaticMap(src *roleEntry) map[string]interface{} {
	return map[string]interface{}{
		"cloud":       src.Cloud,
		"root":        src.Root,
		"ttl":         src.TTL,
		"secret_type": string(src.SecretType),
		"secret":      src.Secret,
		"user_groups": src.UserGroups,
		"user_roles":  src.UserRoles,
	}
}

func (b *backend) pathStaticCredsRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("role").(string)
	entry, err := getRoleByName(ctx, roleName, r)
	if err != nil {
		return nil, err
	}

	data := roleToStaticMap(entry)
	return &logical.Response{
		Data: data,
	}, nil
}
