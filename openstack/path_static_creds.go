package openstack

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"time"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
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

type staticUserEntry struct {
	User     *users.User
	Password string
}

func staticCredsStoragePath(name string) string {
	return fmt.Sprintf("%s/%s", "static-creds/", name)
}

func saveUserInfo(ctx context.Context, e *staticUserEntry, s *logical.Request) error {
	storageEntry, err := logical.StorageEntryJSON(staticCredsStoragePath(e.User.Name), e)
	if err != nil {
		return err
	}
	return s.Storage.Put(ctx, storageEntry)
}

func getUserInfo(ctx context.Context, d *framework.FieldData, s *logical.Request) (*staticUserEntry, error) {
	name := d.Get("role").(string)
	return getUserInfoByName(ctx, name, s)
}

func getUserInfoByName(ctx context.Context, name string, s *logical.Request) (*staticUserEntry, error) {
	entry, err := s.Storage.Get(ctx, staticCredsStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	user := new(staticUserEntry)
	if err := entry.DecodeJSON(user); err != nil {
		return nil, err
	}
	return user, nil
}

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

func (b *backend) pathStaticCredsRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("role").(string)
	role, err := getRoleByName(ctx, roleName, r)
	if err != nil {
		return nil, err
	}

	sharedCloud := b.getSharedCloud(role.Cloud)

	client, err := sharedCloud.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	cloudConfig, err := sharedCloud.getCloudConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	opts := &credsOpts{
		Role:             role,
		Config:           cloudConfig,
		PwdGenerator:     sharedCloud.passwords,
		UsernameTemplate: cloudConfig.UsernameTemplate,
	}

	extractedUser, err := getUserInfo(ctx, d, r)
	if err != nil {
		return nil, err
	}

	var user *staticUserEntry
	if extractedUser != nil {
		user = extractedUser
	} else {
		password, err := opts.PwdGenerator.Generate(context.Background())
		if err != nil {
			return nil, err
		}

		userName, err := createUser(client, opts.Role.Name, password, opts.Role)
		if err != nil {
			return nil, err
		}
		user.User = userName
		user.Password = password
		if err := saveUserInfo(ctx, user, r); err != nil {
			return nil, err
		}
	}

	//if role.Root {
	//	return getStaticRootCredentials(client, opts, user)
	//}

	return getStaticUserCredentials(client, opts, user)
}

func getStaticRootCredentials(client *gophercloud.ServiceClient, opts *credsOpts, user *staticUserEntry) (*logical.Response, error) {
	if opts.Role.SecretType == SecretPassword {
		return nil, errRootNotToken
	}
	tokenOpts := &tokens.AuthOptions{
		Username:   opts.Config.Username,
		Password:   opts.Config.Password,
		DomainName: opts.Config.UserDomainName,
		Scope:      getScopeFromRole(opts.Role),
	}

	token, err := createToken(client, tokenOpts)
	if err != nil {
		return nil, err
	}

	authResponse := &authResponseData{
		AuthURL:    opts.Config.AuthURL,
		Token:      token.ID,
		DomainName: opts.Config.UserDomainName,
	}

	data := map[string]interface{}{
		"auth": formAuthResponse(
			opts.Role,
			authResponse,
		),
		"auth_type": "token",
	}
	secret := &logical.Secret{
		LeaseOptions: logical.LeaseOptions{
			TTL:       time.Until(token.ExpiresAt),
			IssueTime: time.Now(),
		},
		InternalData: map[string]interface{}{
			"secret_type": backendSecretTypeToken,
			"cloud":       opts.Config.Name,
			"expires_at":  token.ExpiresAt.String(),
		},
	}
	return &logical.Response{Data: data, Secret: secret}, nil
}

func getStaticUserCredentials(client *gophercloud.ServiceClient, opts *credsOpts, user *staticUserEntry) (*logical.Response, error) {
	var data map[string]interface{}
	var secretInternal map[string]interface{}
	switch r := opts.Role.SecretType; r {
	case SecretToken:
		tokenOpts := &tokens.AuthOptions{
			Username: user.User.Name,
			Password: user.Password,
			DomainID: user.User.DomainID,
			Scope:    getScopeFromRole(opts.Role),
		}

		token, err := createToken(client, tokenOpts)
		if err != nil {
			return nil, err
		}

		authResponse := &authResponseData{
			AuthURL:  opts.Config.AuthURL,
			Token:    token.ID,
			DomainID: user.User.DomainID,
		}

		data = map[string]interface{}{
			"auth": formAuthResponse(
				opts.Role,
				authResponse,
			),
			"auth_type": "token",
		}
		secretInternal = map[string]interface{}{
			"secret_type": backendSecretTypeUser,
			"user_id":     user.User.ID,
			"cloud":       opts.Config.Name,
			"expires_at":  token.ExpiresAt.String(),
		}
	case SecretPassword:
		authResponse := &authResponseData{
			AuthURL:  opts.Config.AuthURL,
			Username: user.User.Name,
			Password: user.Password,
			DomainID: user.User.DomainID,
		}
		data = map[string]interface{}{
			"auth": formAuthResponse(
				opts.Role,
				authResponse,
			),
			"auth_type": "password",
		}

		secretInternal = map[string]interface{}{
			"secret_type": backendSecretTypeUser,
			"user_id":     user.User.ID,
			"cloud":       opts.Config.Name,
		}
	default:
		return nil, fmt.Errorf("invalid secret type: %s", r)
	}

	for extensionKey, extensionValue := range opts.Role.Extensions {
		data[extensionKey] = extensionValue
	}

	return &logical.Response{
		Data: data,
		Secret: &logical.Secret{
			LeaseOptions: logical.LeaseOptions{
				TTL:       opts.Role.TTL * time.Second,
				IssueTime: time.Now(),
			},
			InternalData: secretInternal,
		},
	}, nil
}
