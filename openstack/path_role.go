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
)

var (
	pathRole = fmt.Sprintf("role/%s", framework.GenericNameRegex("name"))
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
		},
	}
}

func (b *backend) pathRole() *framework.Path {
	return &framework.Path{
		Pattern: pathRole,
		Fields: map[string]*framework.FieldSchema{
			"cloud": {
				Type:        framework.TypeNameString,
				Description: "Specifies root configuration of the created role.",
				Required:    true,
			},
			"name": {
				Type:        framework.TypeNameString,
				Description: "Specifies the name of the role to create. This is part of the request URL.",
			},
			"root": {
				Type:        framework.TypeBool,
				Description: "Specifies whenever to use the root user as a role actor.",
				Default:     false,
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
				Default:       "token",
			},
			"user_groups": {
				Type:        framework.TypeCommaStringSlice,
				Description: "Specifies list of existing OpenStack groups this Vault role is allowed to assume.",
			},
			"project_id": {
				Type:        framework.TypeLowerCaseString,
				Description: "Specifies a project ID for project-scoped role.",
			},
			"project_name": {
				Type:        framework.TypeNameString,
				Description: "Specifies a project ID for project-scoped role.",
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

		ExistenceCheck: b.roleExistenceCheck,
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
	Cloud       string            `json:"cloud"`
	Root        bool              `json:"root"`
	TTL         time.Duration     `json:"ttl"`
	SecretType  secretType        `json:"secret_type"`
	UserGroups  []string          `json:"user_groups"`
	ProjectID   string            `json:"project_id"`
	ProjectName string            `json:"project_name"`
	Extensions  map[string]string `json:"extensions"`
}

func roleStoragePath(name string) string {
	return fmt.Sprintf("%s/%s", rolesStoragePath, name)
}

func getRole(ctx context.Context, d *framework.FieldData, s logical.Storage) (*roleEntry, error) {
	name := d.Get("name").(string)
	return getRoleByName(ctx, name, s)
}

func saveRole(ctx context.Context, name string, e *roleEntry, s logical.Storage) error {
	storageEntry, err := logical.StorageEntryJSON(roleStoragePath(name), e)
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
		"project_id":   src.ProjectID,
		"project_name": src.ProjectName,
		"extensions":   src.Extensions,
	}
}

var errRoleGet = errors.New("error searching for the role")

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

const errInvalidForRoot = "impossible to set %s for the root user"

func (b *backend) pathRoleUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)

	entry, err := getRoleByName(ctx, name, req.Storage)
	if err != nil {
		return logical.ErrorResponse("failed to find the role"), err
	}
	if entry == nil {
		if req.Operation == logical.UpdateOperation {
			return logical.ErrorResponse("role `%s` not found during update operation", name), nil
		}
		entry = new(roleEntry)
	}

	if cloud, ok := d.GetOk("cloud"); ok {
		entry.Cloud = cloud.(string)
	}

	if isRoot, ok := d.GetOk("root"); ok {
		entry.Root = isRoot.(bool)
	}

	if !entry.Root {
		if ttl, ok := d.GetOk("ttl"); ok {
			entry.TTL = time.Duration(ttl.(int)) * time.Second
		} else if req.Operation == logical.CreateOperation {
			entry.TTL = time.Hour
		}
	} else {
		if _, ok := d.GetOk("ttl"); ok {
			return logical.ErrorResponse(errInvalidForRoot, "ttl"), nil
		}
	}

	if typ, ok := d.GetOk("secret_type"); ok {
		if entry.Root {
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

	if ext, ok := d.GetOk("extensions"); ok {
		entry.Extensions = ext.(map[string]string)
	}

	if groups, ok := d.GetOk("user_groups"); ok {
		if entry.Root {
			return logical.ErrorResponse(errInvalidForRoot, "user groups"), nil
		}
		entry.UserGroups = groups.([]string)
	}

	if err := saveRole(ctx, name, entry, req.Storage); err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
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
	return &logical.Response{}, err
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
