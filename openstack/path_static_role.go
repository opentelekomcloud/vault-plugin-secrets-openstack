package openstack

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	staticRolesStoragePath = "static-roles"

	staticRoleHelpSyn = "Manages the Vault role for generating static Openstack users."

	staticRoleHelpDesc = `
This path allows you to read and write static roles that are used to store OpenStack login
credentials. These roles are associated with either an existing user, or a list of user groups,
which are used to control permissions to OpenStack resources.
`
)

var (
	staticPathRole = fmt.Sprintf("static-role/%s", framework.GenericNameRegex("name"))
)

func (b *backend) pathStaticRoles() *framework.Path {
	return &framework.Path{
		Pattern: fmt.Sprintf("%s/?$", staticRolesStoragePath),
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
		HelpSynopsis:    rolesListHelpSyn,
		HelpDescription: rolesListHelpDesc,
	}
}

func (b *backend) pathStaticRole() *framework.Path {
	return &framework.Path{
		Pattern: staticPathRole,
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "Specifies the name of the static role to create. This is part of the request URL.",
			},
			"cloud": {
				Type:        framework.TypeString,
				Description: "Specifies root configuration of the created static role.",
			},
			"rotation_duration": {
				Type:        framework.TypeDurationSecond,
				Description: "Specifies the duration of static role password rotation.",
				Default:     "1h",
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Internal field which specifies the remaining time for the next password rotation.",
				Default:     "1h",
			},
			"secret_type": {
				Type:          framework.TypeLowerCaseString,
				Description:   "Specifies what kind of secret will configuration contain.",
				AllowedValues: []interface{}{"token", "password"},
				Default:       SecretToken,
			},
			"secret": {
				Type: framework.TypeString,
				Description: "Internal field for Openstack user password which will be rotated " +
					"upon static role creation.",
			},
			"username": {
				Type:        framework.TypeNameString,
				Description: "Specifies a username for static role.",
			},
			"user_id": {
				Type:        framework.TypeNameString,
				Description: "Internal field with static user id for further user management. Set once on role creation",
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
	role, err := getStaticRole(ctx, d, r)
	if err != nil {
		return false, err
	}
	return role != nil, nil
}

type roleStaticEntry struct {
	Name             string            `json:"name"`
	Cloud            string            `json:"cloud"`
	TTL              time.Duration     `json:"ttl,omitempty"`
	RotationDuration time.Duration     `json:"rotation_duration,omitempty"`
	SecretType       secretType        `json:"secret_type"`
	Secret           string            `json:"secret"`
	Username         string            `json:"username"`
	UserID           string            `json:"user_id"`
	ProjectID        string            `json:"project_id"`
	ProjectName      string            `json:"project_name"`
	DomainID         string            `json:"domain_id"`
	DomainName       string            `json:"domain_name"`
	Extensions       map[string]string `json:"extensions"`
}

func roleStaticStoragePath(name string) string {
	return fmt.Sprintf("%s/%s", "static-roles", name)
}

func getStaticRole(ctx context.Context, d *framework.FieldData, s *logical.Request) (*roleStaticEntry, error) {
	name := d.Get("name").(string)
	return getStaticRoleByName(ctx, name, s)
}

func saveStaticRole(ctx context.Context, e *roleStaticEntry, s *logical.Request) error {
	storageEntry, err := logical.StorageEntryJSON(roleStaticStoragePath(e.Name), e)
	if err != nil {
		return err
	}

	return s.Storage.Put(ctx, storageEntry)
}

func getStaticRoleByName(ctx context.Context, name string, s *logical.Request) (*roleStaticEntry, error) {
	entry, err := s.Storage.Get(ctx, roleStaticStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	role := new(roleStaticEntry)
	if err := entry.DecodeJSON(role); err != nil {
		return nil, err
	}
	return role, nil
}

func staticRoleToMap(src *roleStaticEntry) map[string]interface{} {
	return map[string]interface{}{
		"cloud":             src.Cloud,
		"rotation_duration": src.RotationDuration,
		"secret_type":       string(src.SecretType),
		"username":          src.Username,
		"project_id":        src.ProjectID,
		"project_name":      src.ProjectName,
		"domain_id":         src.DomainID,
		"domain_name":       src.DomainName,
		"extensions":        src.Extensions,
	}
}

func (b *backend) pathStaticRoleRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entry, err := getStaticRole(ctx, d, req)
	if err != nil {
		return nil, errRoleGet
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
			return logical.ErrorResponse("cloud is required when creating a static role"), nil
		}
	}

	cloud := b.getSharedCloud(cloudName)
	cloudConfig, err := cloud.getCloudConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if cloudConfig == nil {
		return logical.ErrorResponse("cloud `%s` doesn't exist", cloudName), nil
	}

	name := d.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("name is required"), nil
	}

	entry, err := getStaticRoleByName(ctx, name, req)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		if req.Operation == logical.UpdateOperation {
			return logical.ErrorResponse("static role `%s` not found during update operation", name), nil
		}
		entry = &roleStaticEntry{Name: name, Cloud: cloudName}
	}

	if username, ok := d.GetOk("username"); ok && req.Operation == logical.CreateOperation {
		entry.Username = username.(string)
		password, err := Passwords{}.Generate(ctx)
		if err != nil {
			return nil, err
		}

		userId, err := b.rotateUserPassword(ctx, req, cloud, username.(string), password)
		if err != nil {
			return logical.ErrorResponse("error during role creation: %s", err), nil
		}

		entry.UserID = userId
		entry.Secret = password

	} else if req.Operation == logical.CreateOperation {
		return logical.ErrorResponse("username is required when creating a static role"), nil
	}

	if rotation, ok := d.GetOk("rotation_duration"); ok {
		entry.RotationDuration = time.Duration(rotation.(int))
		entry.TTL = time.Duration(rotation.(int))
	} else if req.Operation == logical.CreateOperation {
		entry.RotationDuration = time.Hour / time.Second
		entry.TTL = time.Hour / time.Second
	}

	if typ, ok := d.GetOk("secret_type"); ok {
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

	if err := saveStaticRole(ctx, entry, req); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathStaticRoleDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)
	entry, err := req.Storage.Get(ctx, roleStaticStoragePath(name))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return &logical.Response{}, nil
	}

	err = req.Storage.Delete(ctx, roleStaticStoragePath(name))
	return nil, err
}

func (b *backend) pathStaticRolesList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roles, err := req.Storage.List(ctx, fmt.Sprintf("%s/", staticRolesStoragePath))
	if err != nil {
		return nil, err
	}

	// filter by cloud
	if cloud, ok := d.GetOk("cloud"); ok {
		var refinedRoles []string
		for _, name := range roles {
			role, err := getStaticRoleByName(ctx, name, req)
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
