package openstack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	nameDefaultSet = `0123456789abcdefghijklmnopqrstuvwxyz`

	pathCreds = "creds"

	credsHelpSyn  = "Manage the OpenStack credentials with roles."
	credsHelpDesc = `
This path allows you to create OpenStack token or temporary user using predefined roles.
`
)

func secretToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretTypeToken,
		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "OpenStack Token.",
			},
			"role": {
				Type:        framework.TypeString,
				Description: "Used role.",
			},
		},
		Revoke: b.tokenRevoke,
	}
}

func secretUser(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretTypeUser,
		Fields: map[string]*framework.FieldSchema{
			"user_id": {
				Type:        framework.TypeString,
				Description: "User ID of temporary account.",
			},
			"role": {
				Type:        framework.TypeString,
				Description: "Used role.",
			},
		},
		Revoke: b.userDelete,
	}
}

func (b *backend) pathCreds() *framework.Path {
	return &framework.Path{
		Pattern: fmt.Sprintf("%s/%s", pathCreds, framework.GenericNameRegex("role")),
		Fields: map[string]*framework.FieldSchema{
			"role": {
				Type:        framework.TypeString,
				Description: "Name of the role.",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathCredsRead,
			},
		},
		HelpSynopsis:    credsHelpSyn,
		HelpDescription: credsHelpDesc,
	}
}

func (b *backend) pathCredsRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("role").(string)
	role, err := getRoleByName(ctx, roleName, r.Storage)
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

	var data map[string]interface{}
	secret := new(logical.Secret)
	if role.Root {
		if role.SecretType == "token" {
			token, err := createToken(
				client,
				cloudConfig.Name,
				cloudConfig.Password,
				role.ProjectName,
				cloudConfig.UserDomainName,
			)
			if err != nil {
				return nil, err
			}
			data = map[string]interface{}{
				"role":       roleName,
				"auth_url":   cloudConfig.AuthURL,
				"token":      token.ID,
				"expires_at": token.ExpiresAt.String(),
			}
			secret = &logical.Secret{
				LeaseOptions: logical.LeaseOptions{
					TTL:       time.Until(token.ExpiresAt),
					IssueTime: time.Now(),
				},
				InternalData: map[string]interface{}{
					"secret_type": backendSecretTypeToken,
				},
			}
		} else {
			data = map[string]interface{}{
				"role":             roleName,
				"auth_url":         cloudConfig.AuthURL,
				"user_domain_name": cloudConfig.UserDomainName,
				"username":         cloudConfig.Username,
				"password":         cloudConfig.Password,
			}
		}
	} else {
		password := randomString(pwdDefaultSet, 6)
		user, err := createUser(client, password)
		if err != nil {
			return nil, err
		}
		if role.SecretType == "token" {
			token, err := createToken(
				client,
				user.Name,
				password,
				role.ProjectName,
				cloudConfig.UserDomainName,
			)
			if err != nil {
				return nil, err
			}
			data = map[string]interface{}{
				"role":       roleName,
				"auth_url":   cloudConfig.AuthURL,
				"token":      token.ID,
				"expires_at": token.ExpiresAt.String(),
			}
			secret = &logical.Secret{
				LeaseOptions: logical.LeaseOptions{
					TTL:       time.Until(token.ExpiresAt),
					IssueTime: time.Now(),
				},
				InternalData: map[string]interface{}{
					"secret_type": backendSecretTypeToken,
				},
			}
		} else {
			data = map[string]interface{}{
				"role":               roleName,
				"auth_url":           cloudConfig.AuthURL,
				"username":           user.Name,
				"password":           password,
				"domain_id":          user.DomainID,
				"default_project_id": user.DefaultProjectID,
			}
			secret = &logical.Secret{
				LeaseOptions: logical.LeaseOptions{
					TTL:       time.Until(time.Now().Add(time.Hour)),
					IssueTime: time.Now(),
				},
				InternalData: map[string]interface{}{
					"secret_type": backendSecretTypeUser,
					"user_id":     user.ID,
				},
			}
		}
	}

	return &logical.Response{
		Data:   data,
		Secret: secret,
	}, nil
}

func (b *backend) tokenRevoke(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	tokenRaw, ok := d.GetOk("token")
	if !ok {
		return nil, errors.New("data 'token' not found")
	}

	token := tokenRaw.(string)

	roleName := d.Get("role").(string)
	role, err := getRoleByName(ctx, roleName, r.Storage)
	if err != nil {
		return nil, err
	}

	sharedCloud := b.getSharedCloud(role.Cloud)
	client, err := sharedCloud.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	err = tokens.Revoke(client, token).Err
	if err != nil {
		return nil, fmt.Errorf("unable to revoke token: %w", err)
	}

	return &logical.Response{}, nil
}

func (b *backend) userDelete(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	userIDRaw, ok := r.Secret.InternalData["user_id"]
	if !ok {
		return nil, errors.New("internal data 'user_id' not found")
	}

	userID := userIDRaw.(string)

	roleName := d.Get("role").(string)
	role, err := getRoleByName(ctx, roleName, r.Storage)
	if err != nil {
		return nil, err
	}

	sharedCloud := b.getSharedCloud(role.Cloud)
	client, err := sharedCloud.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	err = users.Delete(client, userID).ExtractErr()
	if err != nil {
		return nil, fmt.Errorf("unable to delete user: %w", err)
	}

	return &logical.Response{}, nil
}

func createUser(client *gophercloud.ServiceClient, password string) (*users.User, error) {
	username := randomString(nameDefaultSet, 6)
	createOpts := users.CreateOpts{
		Name:        username,
		Description: "Vault's temporary user",
		Password:    password,
	}
	user, err := users.Create(client, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating a user: %w", err)
	}

	return user, nil
}

func createToken(client *gophercloud.ServiceClient, username, password, roleProjectName, userDomainName string) (*tokens.Token, error) {
	tokenOpts := &tokens.AuthOptions{
		Username:   username,
		Password:   password,
		DomainName: userDomainName,
		Scope: tokens.Scope{
			DomainName:  userDomainName,
			ProjectName: roleProjectName,
		},
	}
	token, err := tokens.Create(client, tokenOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating a token: %w", err)
	}

	return token, nil
}
