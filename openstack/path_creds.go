package openstack

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathCreds = "creds"

	credsHelpSyn  = "Manage the OpenStack credentials with roles."
	credsHelpDesc = `
This path allows you to create OpenStack token or temporary user using predefined roles.
`
)

var errRootNotToken = errors.New("can't generate non-token credentials for the root user")

func secretToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretTypeToken,
		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "OpenStack Token.",
			},
			"cloud": {
				Type:        framework.TypeString,
				Description: "Used cloud.",
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
			"cloud": {
				Type:        framework.TypeString,
				Description: "Used cloud.",
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

func getRootCredentials(client *gophercloud.ServiceClient, role *roleEntry, config *OsCloud) (*logical.Response, error) {
	if role.SecretType == "password" {
		return nil, errRootNotToken
	}
	tokenOpts := &tokens.AuthOptions{
		Username:   config.Username,
		Password:   config.Password,
		DomainName: config.UserDomainName,
		Scope: tokens.Scope{
			ProjectName: role.ProjectName,
			ProjectID:   role.ProjectID,
		},
	}
	token, err := createToken(client, tokenOpts)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"auth_url":   config.AuthURL,
		"token":      token.ID,
		"expires_at": token.ExpiresAt.String(),
	}
	secret := &logical.Secret{
		LeaseOptions: logical.LeaseOptions{
			TTL:       time.Until(token.ExpiresAt),
			IssueTime: time.Now(),
		},
		InternalData: map[string]interface{}{
			"secret_type": backendSecretTypeToken,
			"cloud":       config.Name,
		},
	}
	return &logical.Response{Data: data, Secret: secret}, nil
}

func getTmpUserCredentials(client *gophercloud.ServiceClient, role *roleEntry, config *OsCloud) (*logical.Response, error) {
	password := RandomString(PwdDefaultSet, 6)
	user, err := createUser(client, password, role.UserGroups, role.UserRoles)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	var secretInternal map[string]interface{}
	if role.SecretType == "token" {
		opts := &tokens.AuthOptions{
			Username: user.Name,
			Password: password,
			DomainID: user.DomainID,
		}
		token, err := createToken(client, opts)
		if err != nil {
			return nil, err
		}
		data = map[string]interface{}{
			"auth_url":   config.AuthURL,
			"token":      token.ID,
			"expires_at": token.ExpiresAt.String(),
		}
		secretInternal = map[string]interface{}{
			"secret_type": backendSecretTypeUser,
			"user_id":     user.ID,
			"cloud":       config.Name,
		}
	} else {
		data = map[string]interface{}{
			"auth_url": config.AuthURL,
			"username": user.Name,
			"password": password,
		}
		switch {
		case role.ProjectID != "":
			data["project_id"] = role.ProjectID
			data["project_domain_id"] = user.DomainID
		case role.ProjectName != "":
			data["project_name"] = role.ProjectName
			data["project_domain_id"] = user.DomainID
		default:
			data["user_domain_id"] = user.DomainID
		}

		secretInternal = map[string]interface{}{
			"secret_type": backendSecretTypeUser,
			"user_id":     user.ID,
			"cloud":       config.Name,
		}
	}
	return &logical.Response{
		Data: data,
		Secret: &logical.Secret{
			LeaseOptions: logical.LeaseOptions{
				TTL:       role.TTL * time.Second,
				IssueTime: time.Now(),
			},
			InternalData: secretInternal,
		},
	}, nil
}

func (b *backend) pathCredsRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	log.Printf("Path /creds/read passing")
	roleName := d.Get("role").(string)
	role, err := getRoleByName(ctx, roleName, r.Storage)
	if err != nil {
		return nil, err
	}

	log.Printf("role name: %v", roleName)
	log.Printf("role from storage: %v", role)

	sharedCloud := b.getSharedCloud(role.Cloud)
	cloudConfig, err := sharedCloud.getCloudConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	client, err := sharedCloud.getClient(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	if role.Root {
		return getRootCredentials(client, role, cloudConfig)
	}

	return getTmpUserCredentials(client, role, cloudConfig)
}

func (b *backend) tokenRevoke(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	log.Printf("Path /creds/revoke_token passing")
	tokenRaw, ok := d.GetOk("token")
	if !ok {
		return nil, errors.New("data 'token' not found")
	}

	token := tokenRaw.(string)

	cloudNameRaw, ok := r.Secret.InternalData["cloud"]
	if !ok {
		return nil, errors.New("internal data 'cloud' not found")
	}

	cloudName := cloudNameRaw.(string)

	sharedCloud := b.getSharedCloud(cloudName)
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

func (b *backend) userDelete(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	log.Printf("Path /creds/revoke_user started")
	userIDRaw, ok := r.Secret.InternalData["user_id"]
	if !ok {
		return nil, errors.New("internal data 'user_id' not found")
	}

	userID := userIDRaw.(string)

	cloudNameRaw, ok := r.Secret.InternalData["cloud"]
	if !ok {
		return nil, errors.New("internal data 'cloud' not found")
	}

	cloudName := cloudNameRaw.(string)

	sharedCloud := b.getSharedCloud(cloudName)
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

func createUser(client *gophercloud.ServiceClient, password string, userGroups, userRoles []string) (*users.User, error) {
	token := tokens.Get(client, client.Token())
	user, err := token.ExtractUser()
	if err != nil {
		return nil, fmt.Errorf("error extracting the user from token: %w", err)
	}

	username := RandomString(NameDefaultSet, 6)
	userCreateOpts := users.CreateOpts{
		Name:        username,
		Description: "Vault's temporary user",
		DomainID:    user.Domain.ID,
		Password:    password,
		DomainID:    user.Domain.ID,
	}
	newUser, err := users.Create(client, userCreateOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating a user: %w", err)
	}

	rolesToAdd, err := filterRoles(client, userRoles)
	if err != nil {
		return nil, err
	}

	for _, role := range rolesToAdd {
		assignOpts := roles.AssignOpts{
			UserID:   newUser.ID,
			DomainID: user.Domain.ID,
		}
		if err := roles.Assign(client, role.ID, assignOpts).ExtractErr(); err != nil {
			return nil, fmt.Errorf("cannot assign a role `%s` to a temporary user: %w", role, err)
		}
	}

	groupsToAssign, err := filterGroups(client, user.Domain.ID, userGroups)
	if err != nil {
		return nil, err
	}

	for _, group := range groupsToAssign {
		if err := users.AddToGroup(client, group.ID, newUser.ID).ExtractErr(); err != nil {
			return nil, fmt.Errorf("cannot add a temporary user to a group `%s`: %w", group, err)
		}
	}

	return newUser, nil
}

func createToken(client *gophercloud.ServiceClient, opts tokens.AuthOptionsBuilder) (*tokens.Token, error) {
	token, err := tokens.Create(client, opts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating a token: %w", err)
	}

	return token, nil
}

func filterRoles(client *gophercloud.ServiceClient, roleNames []string) ([]roles.Role, error) {
	if len(roleNames) == 0 {
		return nil, nil
	}

	rolePages, err := roles.List(client, nil).AllPages()
	if err != nil {
		return nil, fmt.Errorf("unable to query roles: %w", err)
	}

	roleList, err := roles.ExtractRoles(rolePages)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve roles: %w", err)
	}

	var filteredRoles []roles.Role
	for _, name := range roleNames {
		for _, role := range roleList {
			if role.Name == name {
				filteredRoles = append(filteredRoles, role)
				break
			}
		}
	}
	return filteredRoles, nil
}

func filterGroups(client *gophercloud.ServiceClient, domainID string, groupNames []string) ([]groups.Group, error) {
	if len(groupNames) == 0 {
		return nil, nil
	}

	groupPages, err := groups.List(client, groups.ListOpts{
		DomainID: domainID,
	}).AllPages()
	if err != nil {
		return nil, err
	}

	groupList, err := groups.ExtractGroups(groupPages)
	if err != nil {
		return nil, err
	}

	var filteredGroups []groups.Group
	for _, name := range groupNames {
		for _, group := range groupList {
			if group.Name == name {
				filteredGroups = append(filteredGroups, group)
				break
			}
		}
	}
	return filteredGroups, nil
}
