package openstack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	rolesStoragePath = "roles"
	pathRoles        = `roles/?`

	errInvalidForRoot = "impossible to set %s for the root user"

	rolesListHelpSyn  = `List existing roles.`
	rolesListHelpDesc = `
List existing roles by name. Supports filtering by cloud.
`
	roleHelpSyn  = "Manage the Vault roles used to generate OpenStack credentials."
	roleHelpDesc = `
This path allows you to read and write roles that are used to generate OpenStack login
credentials. These roles are associated with either an existing user, or a list of user groups,
which are used to control permissions to OpenStack resources.
`
)

var (
	pathRole = fmt.Sprintf("role/%s", framework.GenericNameRegex("name"))

	errRoleGet = errors.New("error searching for the role")
)

func (b *backend) pathRoles() *framework.Path {
	return &framework.Path{
		Pattern: pathRoles,
		Fields: map[string]*framework.FieldSchema{
			"cloud": {
				Type:        framework.TypeNameString,
				Description: "Specifies root configuration of the created role.",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback: b.pathRolesList,
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathRolesList,
			},
		},
		HelpSynopsis:    rolesListHelpSyn,
		HelpDescription: rolesListHelpDesc,
	}
}

func (b *backend) pathRole() *framework.Path {
	return &framework.Path{
		Pattern: pathRole,
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "Specifies the name of the role to create. This is part of the request URL.",
			},
			"cloud": {
				Type:        framework.TypeString,
				Description: "Specifies root configuration of the created role.",
			},
			"root": {
				Type:        framework.TypeBool,
				Description: "Specifies whenever to use the root user as a role actor.",
				Default:     false,
			},
			"username": {
				Type:        framework.TypeString,
				Description: "Specifies the username of the static user.",
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Specifies TTL value for the dynamically created users as a string duration with time suffix.",
				Default:     "1h",
			},
			"secret_type": {
				Type:          framework.TypeLowerCaseString,
				Description:   "Specifies what kind of secret will configuration contain.",
				AllowedValues: []interface{}{"token", "password"},
				Default:       SecretToken,
			},
			"user_groups": {
				Type:        framework.TypeCommaStringSlice,
				Description: "Specifies list of existing OpenStack groups this Vault role is allowed to assume.",
			},
			"user_roles": {
				Type:        framework.TypeCommaStringSlice,
				Description: "Specifies list of existing OpenStack roles this Vault role is allowed to assume.",
			},
			"project_id": {
				Type:        framework.TypeLowerCaseString,
				Description: "Specifies a project ID for project-scoped role.",
			},
			"project_name": {
				Type:        framework.TypeNameString,
				Description: "Specifies a project name for project-scoped role.",
			},
			"domain_id": {
				Type:        framework.TypeLowerCaseString,
				Description: "Specifies a domain ID for domain-scoped role.",
			},
			"domain_name": {
				Type:        framework.TypeNameString,
				Description: "Specifies a domain name for domain-scoped role.",
			},
			"extensions": {
				Type: framework.TypeKVPairs,
				Description: "A list of strings representing a key/value pair to be used as extensions to the cloud " +
					"configuration (e.g. `volume_api_version` or endpoint overrides).",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathRoleRead,
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathRoleUpdate,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathRoleUpdate,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathRoleDelete,
			},
		},
		ExistenceCheck:  b.roleExistenceCheck,
		HelpSynopsis:    roleHelpSyn,
		HelpDescription: roleHelpDesc,
	}
}

func (b *backend) roleExistenceCheck(ctx context.Context, r *logical.Request, d *framework.FieldData) (bool, error) {
	role, err := getRole(ctx, d, r.Storage)
	if err != nil {
		return false, err
	}
	return role != nil, nil
}

type secretType string

const (
	SecretPassword secretType = "password"
	SecretToken    secretType = "token"
)

type roleEntry struct {
	Name        string            `json:"name"`
	Cloud       string            `json:"cloud"`
	Root        bool              `json:"root"`
	TTL         time.Duration     `json:"ttl,omitempty"`
	SecretType  secretType        `json:"secret_type"`
	UserGroups  []string          `json:"user_groups"`
	UserRoles   []string          `json:"user_roles"`
	Username    string            `json:"username"`
	ProjectID   string            `json:"project_id"`
	ProjectName string            `json:"project_name"`
	DomainID    string            `json:"domain_id"`
	DomainName  string            `json:"domain_name"`
	Extensions  map[string]string `json:"extensions"`
}

func roleStoragePath(name string) string {
	return fmt.Sprintf("%s/%s", rolesStoragePath, name)
}

func getRole(ctx context.Context, d *framework.FieldData, s logical.Storage) (*roleEntry, error) {
	name := d.Get("name").(string)
	return getRoleByName(ctx, name, s)
}

func saveRole(ctx context.Context, e *roleEntry, s logical.Storage) error {
	storageEntry, err := logical.StorageEntryJSON(roleStoragePath(e.Name), e)
	if err != nil {
		return err
	}
	return s.Put(ctx, storageEntry)
}

func getRoleByName(ctx context.Context, name string, s logical.Storage) (*roleEntry, error) {
	entry, err := s.Get(ctx, roleStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	role := new(roleEntry)
	if err := entry.DecodeJSON(role); err != nil {
		return nil, err
	}
	return role, nil
}

func roleToMap(src *roleEntry) map[string]interface{} {
	return map[string]interface{}{
		"cloud":        src.Cloud,
		"root":         src.Root,
		"ttl":          src.TTL,
		"secret_type":  string(src.SecretType),
		"user_groups":  src.UserGroups,
		"user_roles":   src.UserRoles,
		"project_id":   src.ProjectID,
		"project_name": src.ProjectName,
		"domain_id":    src.DomainID,
		"domain_name":  src.DomainName,
		"extensions":   src.Extensions,
	}
}

func (b *backend) pathRoleRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entry, err := getRole(ctx, d, req.Storage)
	if err != nil {
		return nil, errRoleGet
	}
	if entry == nil {
		return logical.ErrorResponse("role not found"), nil
	}

	data := roleToMap(entry)
	return &logical.Response{
		Data: data,
	}, nil
}

func (b *backend) pathRoleUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	var cloudName string
	if cloud, ok := d.GetOk("cloud"); ok {
		cloudName = cloud.(string)
	} else {
		if req.Operation == logical.CreateOperation {
			return logical.ErrorResponse("cloud is required when creating a role"), nil
		}
	}
	cld, err := b.getSharedCloud(cloudName).getCloudConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if cld == nil {
		return logical.ErrorResponse("cloud `%s` doesn't exist", cloudName), nil
	}

	name := d.Get("name").(string)

	entry, err := getRoleByName(ctx, name, req.Storage)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		if req.Operation == logical.UpdateOperation {
			return logical.ErrorResponse("role `%s` not found during update operation", name), nil
		}
		entry = &roleEntry{Name: name, Cloud: cloudName}
	}

	if isRoot, ok := d.GetOk("root"); ok {
		entry.Root = isRoot.(bool)
	}

	if !entry.Root {
		if ttl, ok := d.GetOk("ttl"); ok {
			entry.TTL = time.Duration(ttl.(int))
		} else if req.Operation == logical.CreateOperation {
			entry.TTL = time.Hour / time.Second
		}
	} else {
		if _, ok := d.GetOk("ttl"); ok {
			return logical.ErrorResponse(errInvalidForRoot, "ttl"), nil
		}
	}

	if typ, ok := d.GetOk("secret_type"); ok {
		if entry.Root && typ != SecretToken {
			return logical.ErrorResponse(errInvalidForRoot, "secret type"), nil
		}
		entry.SecretType = secretType(typ.(string))
	} else if req.Operation == logical.CreateOperation {
		entry.SecretType = SecretToken
	}

	if name, ok := d.GetOk("project_name"); ok {
		entry.ProjectName = name.(string)
	}

	if id, ok := d.GetOk("project_id"); ok {
		entry.ProjectID = id.(string)
	}

	if name, ok := d.GetOk("domain_name"); ok {
		entry.DomainName = name.(string)
	}

	if id, ok := d.GetOk("domain_id"); ok {
		entry.DomainID = id.(string)
	}

	if ext, ok := d.GetOk("extensions"); ok {
		entry.Extensions = ext.(map[string]string)
	}

	if groups, ok := d.GetOk("user_groups"); ok {
		if entry.Root {
			return logical.ErrorResponse(errInvalidForRoot, "user groups"), nil
		}
		entry.UserGroups = groups.([]string)
	}

	if roles, ok := d.GetOk("user_roles"); ok {
		if entry.Root {
			return logical.ErrorResponse(errInvalidForRoot, "user roles"), nil
		}
		entry.UserRoles = roles.([]string)
	}

	if username, ok := d.GetOk("username"); ok {
		if entry.Root {
			return logical.ErrorResponse(errInvalidForRoot, "username"), nil
		}
		entry.Username = username.(string)
	}

	if err := saveRole(ctx, entry, req.Storage); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathRoleDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)
	entry, err := req.Storage.Get(ctx, roleStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return &logical.Response{}, nil
	}

	err = req.Storage.Delete(ctx, roleStoragePath(name))
	return nil, err
}

func (b *backend) pathRolesList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roles, err := req.Storage.List(ctx, rolesStoragePath+"/")
	if err != nil {
		return nil, err
	}

	// filter by cloud
	if cloud, ok := d.GetOk("cloud"); ok {
		var refinedRoles []string
		for _, name := range roles {
			role, err := getRoleByName(ctx, name, req.Storage)
			if err != nil {
				return nil, err
			}
			if role.Cloud != cloud {
				continue
			}
			refinedRoles = append(refinedRoles, name)
		}
		roles = refinedRoles
	}

	return logical.ListResponse(roles), nil
}
