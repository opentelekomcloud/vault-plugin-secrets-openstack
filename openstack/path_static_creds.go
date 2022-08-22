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
	pathStaticCreds       = "static-creds"
	pathStaticCredsRotate = "rotate-role"

	staticCredsHelpSyn  = "Manage the Openstack static credentials with static roles."
	staticCredsHelpDesc = `
This path allows you to read OpenStack secret stored by predefined static roles.
`

	rotateStaticHelpSyn  = "Rotate static role password."
	rotateStaticHelpDesc = `
Rotate the static role user credentials.

Once this method is called, static role will now be the only entity that knows the static user password.
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
		},
		HelpSynopsis:    staticCredsHelpSyn,
		HelpDescription: staticCredsHelpDesc,
	}
}

func (b *backend) pathRotateStaticCreds() *framework.Path {
	return &framework.Path{
		Pattern: fmt.Sprintf("%s/%s", pathStaticCredsRotate, framework.GenericNameRegex("role")),
		Fields: map[string]*framework.FieldSchema{
			"role": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "Specifies name of the static role which credentials will be rotated.",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.rotateStaticCreds,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.rotateStaticCreds,
			},
		},
		HelpSynopsis:    rotateStaticHelpSyn,
		HelpDescription: rotateStaticHelpDesc,
	}
}

func (b *backend) pathStaticCredsRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("role").(string)
	role, err := getStaticRoleByName(ctx, roleName, r)
	if err != nil {
		return nil, err
	}

	sharedCloud := b.getSharedCloud(role.Cloud)
	cloudConfig, err := sharedCloud.getCloudConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	client, err := sharedCloud.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	user, err := users.Get(client, role.UserID).Extract()
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	switch r := role.SecretType; r {
	case SecretToken:
		tokenOpts := &tokens.AuthOptions{
			Username: user.Name,
			Password: role.Secret,
			DomainID: user.DomainID,
			Scope:    getScopeFromStaticRole(role),
		}

		token, err := createToken(client, tokenOpts)
		if err != nil {
			return nil, err
		}

		authResponse := &authStaticResponseData{
			AuthURL:  cloudConfig.AuthURL,
			Username: role.Username,
			Token:    token.ID,
			DomainID: user.DomainID,
		}

		data = map[string]interface{}{
			"auth": formStaticAuthResponse(
				role,
				authResponse,
			),
			"auth_type": "token",
		}

	case SecretPassword:
		authResponse := &authStaticResponseData{
			AuthURL:  cloudConfig.AuthURL,
			Username: role.Username,
			Password: role.Secret,
			DomainID: user.DomainID,
		}
		data = map[string]interface{}{
			"auth": formStaticAuthResponse(
				role,
				authResponse,
			),
			"auth_type": "password",
		}

	default:
		return nil, fmt.Errorf("invalid secret type: %s", r)
	}

	for extensionKey, extensionValue := range role.Extensions {
		data[extensionKey] = extensionValue
	}

	return &logical.Response{Data: data}, nil
}

func (b *backend) rotateStaticCreds(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("role").(string)
	role, err := getStaticRoleByName(ctx, roleName, r)
	if err != nil {
		return nil, err
	}

	sharedCloud := b.getSharedCloud(role.Cloud)
	if err != nil {
		return nil, err
	}

	client, err := sharedCloud.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	newPassword, err := Passwords{}.Generate(ctx)
	if err != nil {
		return nil, err
	}

	err = users.ChangePassword(client, role.UserID, users.ChangePasswordOpts{
		Password:         newPassword,
		OriginalPassword: role.Secret,
	}).ExtractErr()
	if err != nil {
		return nil, err
	}

	role.Secret = newPassword

	if err := saveStaticRole(ctx, role, r); err != nil {
		return nil, err
	}

	return nil, nil
}

func getScopeFromStaticRole(role *roleStaticEntry) tokens.Scope {
	var scope tokens.Scope
	switch {
	case role.ProjectID != "":
		scope = tokens.Scope{
			ProjectID: role.ProjectID,
		}
	case role.ProjectName != "":
		scope = tokens.Scope{
			ProjectName: role.ProjectName,
			DomainName:  role.DomainName,
			DomainID:    role.DomainID,
		}
	case role.DomainID != "":
		scope = tokens.Scope{
			DomainID: role.DomainID,
		}
	case role.DomainName != "":
		scope = tokens.Scope{
			DomainName: role.DomainName,
		}
	default:
		scope = tokens.Scope{}
	}
	return scope
}

type authStaticResponseData struct {
	AuthURL    string
	Username   string
	Password   string
	Token      string
	DomainID   string
	DomainName string
}

func formStaticAuthResponse(role *roleStaticEntry, authResponse *authStaticResponseData) map[string]interface{} {
	var auth map[string]interface{}

	switch {
	case role.ProjectID != "":
		auth = map[string]interface{}{
			"project_id": role.ProjectID,
		}
	case role.ProjectName != "":
		auth = map[string]interface{}{
			"project_name":      role.ProjectName,
			"project_domain_id": authResponse.DomainID,
		}
	default:

		auth = map[string]interface{}{
			"user_domain_id": authResponse.DomainID,
		}
	}

	if authResponse.Token != "" {
		auth["token"] = authResponse.Token
	} else {
		auth["username"] = authResponse.Username
		auth["password"] = authResponse.Password
	}

	auth["auth_url"] = authResponse.AuthURL

	return auth
}

func (b *backend) rotateUserPassword(ctx context.Context, req *logical.Request, cloud *sharedCloud, user string, password string) (string, error) {
	var userId string
	client, err := cloud.getClient(ctx, req.Storage)
	if err != nil {
		return userId, err
	}
	opts := users.ListOpts{Name: user}
	allPages, err := users.List(client, opts).AllPages()
	if err != nil {
		return userId, fmt.Errorf("provided user doesn't exist")
	}

	allUsers, err := users.ExtractUsers(allPages)
	if err != nil {
		return userId, fmt.Errorf("page can't be extracted for given username: %s (%s)", user, err)
	}

	if len(allUsers) > 1 {
		return userId, fmt.Errorf("given username is not unique")
	} else if len(allUsers) == 0 {
		return userId, fmt.Errorf("user `%s` doesn't exist", user)
	}

	userId = allUsers[0].ID

	_, err = users.Update(client, userId, users.UpdateOpts{Password: password}).Extract()
	if err != nil {
		return userId, fmt.Errorf("error rotating user password for user `%s`: %s", user, err)
	}
	return userId, nil
}
