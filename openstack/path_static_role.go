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
	staticRolesStoragePath = "static-roles"
	pathStaticRoles        = `static-roles/?`

	staticRolesListHelpSyn  = `List existing static roles.`
	staticRolesListHelpDesc = `
List existing roles by name. Supports filtering by cloud.
`
	staticRoleHelpSyn  = "Manage the Vault static roles used to generate OpenStack credentials."
	staticRoleHelpDesc = `
This path allows you to read and write static roles ...
`
)

var (
	pathStaticRole = fmt.Sprintf("static-role/%s", framework.GenericNameRegex("name"))

	errStaticRoleGet = errors.New("error searching for the static role")
)

func (b *backend) pathStaticRoles() *framework.Path {
	return &framework.Path{
		Pattern: pathStaticRoles,
		Fields: map[string]*framework.FieldSchema{
			"cloud": {
				Type:        framework.TypeNameString,
				Description: "Specifies root configuration of the created role.",
				Required:    true,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback: b.pathStaticRolesList,
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathStaticRolesList,
			},
		},
		HelpSynopsis:    staticRolesListHelpSyn,
		HelpDescription: staticRolesListHelpDesc,
	}
}

func (b *backend) pathStaticRole() *framework.Path {
	return &framework.Path{
		Pattern: pathStaticRole,
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "Specifies the name of the role to create. This is part of the request URL.",
			},
			"cloud": {
				Type:        framework.TypeString,
				Description: "Specifies root configuration of the created role.",
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Specifies TTL value for the dynamically created users as a string duration with time suffix.",
				Default:     "1h",
			},
			"username": {
				Type:        framework.TypeString,
				Description: "...",
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
				Callback: b.pathStaticRoleRead,
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathStaticRoleUpdate,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathStaticRoleUpdate,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathStaticRoleDelete,
			},
		},
		ExistenceCheck:  b.staticRoleExistenceCheck,
		HelpSynopsis:    staticRoleHelpSyn,
		HelpDescription: staticRoleHelpDesc,
	}
}

func (b *backend) staticRoleExistenceCheck(ctx context.Context, r *logical.Request, d *framework.FieldData) (bool, error) {
	role, err := getRole(ctx, d, r.Storage)
	if err != nil {
		return false, err
	}
	return role != nil, nil
}

type staticRoleEntry struct {
	Name        string            `json:"name"`
	Cloud       string            `json:"cloud"`
	Root        bool              `json:"root"`
	TTL         time.Duration     `json:"ttl,omitempty"`
	SecretType  secretType        `json:"secret_type"`
	UserGroups  []string          `json:"user_groups"`
	UserRoles   []string          `json:"user_roles"`
	ProjectID   string            `json:"project_id"`
	ProjectName string            `json:"project_name"`
	DomainID    string            `json:"domain_id"`
	DomainName  string            `json:"domain_name"`
	Extensions  map[string]string `json:"extensions"`
}

func staticRoleStoragePath(name string) string {
	return fmt.Sprintf("%s/%s", staticRolesStoragePath, name)
}

func getStaticRole(ctx context.Context, d *framework.FieldData, s logical.Storage) (*staticRoleEntry, error) {
	name := d.Get("name").(string)
	return getStaticRoleByName(ctx, name, s)
}

func saveStaticRole(ctx context.Context, e *staticRoleEntry, s logical.Storage) error {
	storageEntry, err := logical.StorageEntryJSON(staticRoleStoragePath(e.Name), e)
	if err != nil {
		return err
	}
	return s.Put(ctx, storageEntry)
}

func getStaticRoleByName(ctx context.Context, name string, s logical.Storage) (*staticRoleEntry, error) {
	entry, err := s.Get(ctx, staticRoleStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	role := new(staticRoleEntry)
	if err := entry.DecodeJSON(role); err != nil {
		return nil, err
	}
	return role, nil
}

func staticRoleToMap(src *staticRoleEntry) map[string]interface{} {
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

func (b *backend) pathStaticRoleRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entry, err := getStaticRole(ctx, d, req.Storage)
	if err != nil {
		return nil, errStaticRoleGet
	}
	if entry == nil {
		return logical.ErrorResponse("static role not found"), nil
	}

	data := staticRoleToMap(entry)
	return &logical.Response{
		Data: data,
	}, nil
}

func (b *backend) pathStaticRoleUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
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

	entry, err := getStaticRoleByName(ctx, name, req.Storage)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		if req.Operation == logical.UpdateOperation {
			return logical.ErrorResponse("role `%s` not found during update operation", name), nil
		}
		entry = &staticRoleEntry{Name: name, Cloud: cloudName}
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

	if err := saveStaticRole(ctx, entry, req.Storage); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathStaticRoleDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)
	entry, err := req.Storage.Get(ctx, staticRoleStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return &logical.Response{}, nil
	}

	err = req.Storage.Delete(ctx, staticRoleStoragePath(name))
	return nil, err
}

func (b *backend) pathStaticRolesList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roles, err := req.Storage.List(ctx, staticRolesStoragePath+"/")
	if err != nil {
		return nil, err
	}

	// filter by cloud
	if cloud, ok := d.GetOk("cloud"); ok {
		var refinedRoles []string
		for _, name := range roles {
			role, err := getStaticRoleByName(ctx, name, req.Storage)
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
