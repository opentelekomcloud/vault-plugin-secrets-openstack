package openstack

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/pagination"
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

type credsStaticOpts struct {
	Role             *roleStaticEntry
	Config           *OsCloud
	PwdGenerator     *Passwords
	UsernameTemplate string
}

type staticUserEntry struct {
	User     *users.User `json:"user"`
	Password string      `json:"password"`
}

func staticCredsStoragePath(name string) string {
	return fmt.Sprintf("%s/%s", pathStaticCreds, name)
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

	opts := &credsStaticOpts{
		Role:   role,
		Config: cloudConfig,
	}

	//extractedUser, err := getUserInfo(ctx, d, r)
	//if err != nil {
	//	return nil, err
	//}

	var user *staticUserEntry
	//if extractedUser != nil {
	//	user = extractedUser
	//} else {
	password, err := opts.PwdGenerator.Generate(context.Background())
	if err != nil {
		return nil, err
	}

	userName, err := createStaticUser(client, opts.Role.Name, password, opts.Role)
	if err != nil {
		return nil, err
	}
	user.User = userName
	user.Password = password
	if err := saveUserInfo(ctx, user, r); err != nil {
		return nil, err
	}
	//}

	//if role.Root {
	//	return getStaticRootCredentials(client, opts, user)
	//}

	return getStaticUserCredentials(client, opts, user)
}

//func getStaticRootCredentials(client *gophercloud.ServiceClient, opts *credsStaticOpts, user *staticUserEntry) (*logical.Response, error) {
//	if opts.Role.SecretType == SecretPassword {
//		return nil, errRootNotToken
//	}
//	tokenOpts := &tokens.AuthOptions{
//		Username:   opts.Config.Username,
//		Password:   opts.Config.Password,
//		DomainName: opts.Config.UserDomainName,
//		Scope:      getScopeFromStaticRole(opts.Role),
//	}
//
//	token, err := createToken(client, tokenOpts)
//	if err != nil {
//		return nil, err
//	}
//
//	authResponse := &authResponseData{
//		AuthURL:    opts.Config.AuthURL,
//		Token:      token.ID,
//		DomainName: opts.Config.UserDomainName,
//	}
//
//	data := map[string]interface{}{
//		"auth": formAuthResponse(
//			opts.Role,
//			authResponse,
//		),
//		"auth_type": "token",
//	}
//	secret := &logical.Secret{
//		LeaseOptions: logical.LeaseOptions{
//			TTL:       time.Until(token.ExpiresAt),
//			IssueTime: time.Now(),
//		},
//		InternalData: map[string]interface{}{
//			"secret_type": backendSecretTypeToken,
//			"cloud":       opts.Config.Name,
//			"expires_at":  token.ExpiresAt.String(),
//		},
//	}
//	return &logical.Response{Data: data, Secret: secret}, nil
//}

func getStaticUserCredentials(client *gophercloud.ServiceClient, opts *credsStaticOpts, user *staticUserEntry) (*logical.Response, error) {
	var data map[string]interface{}
	var secretInternal map[string]interface{}
	switch r := opts.Role.SecretType; r {
	case SecretToken:
		tokenOpts := &tokens.AuthOptions{
			Username: user.User.Name,
			Password: user.Password,
			DomainID: user.User.DomainID,
			Scope:    getScopeFromStaticRole(opts.Role),
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
			"auth": formStaticAuthResponse(
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
			"auth": formStaticAuthResponse(
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

func createStaticUser(client *gophercloud.ServiceClient, username, password string, role *roleStaticEntry) (*users.User, error) {
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
		Description:      "Vault's static user",
		DomainID:         user.Domain.ID,
		Password:         password,
	}

	newUser, err := users.Create(client, userCreateOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("error creating a temporary user: %w", err)
	}

	return newUser, nil
}

func deleteUser(client *gophercloud.ServiceClient, userId string) error {
	deleteUser := users.Delete(client, userId)
	if deleteUser.Err != nil {
		return deleteUser.Err
	}
	return nil
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

func formStaticAuthResponse(role *roleStaticEntry, authResponse *authResponseData) map[string]interface{} {
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
		if role.Root {
			auth = map[string]interface{}{
				"user_domain_name": authResponse.DomainName,
			}
		} else {
			auth = map[string]interface{}{
				"user_domain_id": authResponse.DomainID,
			}
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
