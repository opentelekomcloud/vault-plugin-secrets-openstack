package openstack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/pagination"
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

type credsOpts struct {
	Role             *roleEntry
	Config           *OsCloud
	PwdGenerator     *Passwords
	UsernameTemplate string
}

var errRootNotToken = errors.New("can't generate non-token credentials for the root user")

func secretToken(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretTypeToken,
		Fields: map[string]*framework.FieldSchema{
			"cloud": {
				Type:        framework.TypeString,
				Description: "Used cloud.",
			},
			"auth": {
				Type:        framework.TypeMap,
				Description: "Auth entry for OpenStack clouds.yaml",
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

func getRootCredentials(client *gophercloud.ServiceClient, opts *credsOpts) (*logical.Response, error) {
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

func getTmpUserCredentials(client *gophercloud.ServiceClient, opts *credsOpts) (*logical.Response, error) {
	password, err := opts.PwdGenerator.Generate(context.Background())
	if err != nil {
		return nil, err
	}

	username, err := RandomTemporaryUsername(opts.UsernameTemplate, opts.Role)
	if err != nil {
		return logical.ErrorResponse("error generating username for temporary user: %s", err), nil
	}

	user, err := createUser(client, username, password, opts.Role)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	var secretInternal map[string]interface{}
	switch r := opts.Role.SecretType; r {
	case SecretToken:
		tokenOpts := &tokens.AuthOptions{
			Username: user.Name,
			Password: password,
			DomainID: user.DomainID,
			Scope:    getScopeFromRole(opts.Role),
		}

		token, err := createToken(client, tokenOpts)
		if err != nil {
			return nil, err
		}

		authResponse := &authResponseData{
			AuthURL:  opts.Config.AuthURL,
			Token:    token.ID,
			DomainID: user.DomainID,
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
			"user_id":     user.ID,
			"cloud":       opts.Config.Name,
			"expires_at":  token.ExpiresAt.String(),
		}
	case SecretPassword:
		authResponse := &authResponseData{
			AuthURL:  opts.Config.AuthURL,
			Username: user.Name,
			Password: password,
			DomainID: user.DomainID,
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
			"user_id":     user.ID,
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

	opts := &credsOpts{
		Role:             role,
		Config:           cloudConfig,
		PwdGenerator:     sharedCloud.passwords,
		UsernameTemplate: cloudConfig.UsernameTemplate,
	}

	if role.Root {
		return getRootCredentials(client, opts)
	}

	return getTmpUserCredentials(client, opts)
}

func (b *backend) tokenRevoke(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	authInfoRaw, ok := d.GetOk("auth")
	if !ok {
		return nil, errors.New("data 'auth' not found")
	}

	authInfo := authInfoRaw.(map[string]interface{})
	token := authInfo["token"].(string)

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

func createUser(client *gophercloud.ServiceClient, username, password string, role *roleEntry) (*users.User, error) {
	token := tokens.Get(client, client.Token())
	user, err := token.ExtractUser()
	if err != nil {
		return nil, fmt.Errorf("error extracting the user from token: %w", err)
	}

	projectID := role.ProjectID
	if projectID == "" && role.ProjectName != "" {
		err := projects.List(client, projects.ListOpts{Name: role.ProjectName}).EachPage(func(page pagination.Page) (bool, error) {
			project, err := projects.ExtractProjects(page)
			if err != nil {
				return false, err
			}
			if len(project) > 0 {
				projectID = project[0].ID
				return true, nil
			}

			return false, fmt.Errorf("failed to find project with the name: %s", role.ProjectName)
		})
		if err != nil {
			return nil, err
		}
	}

	userCreateOpts := users.CreateOpts{
		Name:             username,
		DefaultProjectID: projectID,
		Description:      "Vault's temporary user",
		DomainID:         user.Domain.ID,
		Password:         password,
	}

	newUser, err := users.Create(client, userCreateOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating a temporary user: %w", err)
	}

	rolesToAdd, err := filterRoles(client, role.UserRoles)
	if err != nil {
		return nil, err
	}

	for _, identityRole := range rolesToAdd {
		assignOpts := roles.AssignOpts{
			UserID:    newUser.ID,
			ProjectID: projectID,
		}
		if err := roles.Assign(client, identityRole.ID, assignOpts).ExtractErr(); err != nil {
			return nil, fmt.Errorf("cannot assign a role `%s` to a temporary user: %w", identityRole, err)
		}
	}

	groupsToAssign, err := filterGroups(client, user.Domain.ID, role.UserGroups)
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

func getScopeFromRole(role *roleEntry) tokens.Scope {
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

type authResponseData struct {
	AuthURL    string
	Username   string
	Password   string
	Token      string
	DomainID   string
	DomainName string
}

func formAuthResponse(role *roleEntry, authResponse *authResponseData) map[string]interface{} {
	var auth map[string]interface{}

	switch {
	case role.ProjectID != "":
		auth = map[string]interface{}{
			"project_id": role.ProjectID,
		}
	case role.ProjectName != "":
		if role.Root {
			auth = map[string]interface{}{
				"project_name":        role.ProjectName,
				"project_domain_name": authResponse.DomainName,
			}
		} else {
			auth = map[string]interface{}{
				"project_name":      role.ProjectName,
				"project_domain_id": authResponse.DomainID,
			}
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
