package openstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathCloud         = "cloud"
	pathClouds        = "clouds/?"
	cloudsStoragePath = "clouds"

	pathCloudHelpSyn = `Configure the root credentials for an OpenStack cloud.`
	pathCloudHelpDes = `
Configure the root credentials for an OpenStack cloud using the above parameters.
`
	pathCloudListHelpSyn  = `List existing OpenStack clouds.`
	pathCloudListHelpDesc = `List existing OpenStack clouds by name.`
)

func storageCloudKey(name string) string {
	return fmt.Sprintf("%s/%s", cloudsStoragePath, name)
}

func pathCloudKey(name string) string {
	return fmt.Sprintf("%s/%s", pathCloud, name)
}

func (c *sharedCloud) getCloudConfig(ctx context.Context, s logical.Storage) (*OsCloud, error) {
	entry, err := s.Get(ctx, storageCloudKey(c.name))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}
	cloud := &OsCloud{}
	if err := entry.DecodeJSON(cloud); err != nil {
		return nil, err
	}
	return cloud, nil
}

func (cloud *OsCloud) save(ctx context.Context, s logical.Storage) error {
	entry, err := logical.StorageEntryJSON(storageCloudKey(cloud.Name), cloud)
	if err != nil {
		return err
	}

	return s.Put(ctx, entry)
}

func (b *backend) pathCloud() *framework.Path {
	return &framework.Path{
		Pattern: fmt.Sprintf("%s/%s", pathCloud, framework.GenericNameWithAtRegex("name")),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Required:    true,
				Description: "Name of the cloud.",
			},
			"auth_url": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "URL of identity service authentication endpoint.",
			},
			"user_domain_name": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "Name of the domain of the root user.",
			},
			"username": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "OpenStack username of the root user.",
			},
			"username_template": {
				Type:        framework.TypeString,
				Default:     "vault{{random 8 | lowercase}}",
				Description: "Name template for temporary generated users.",
			},
			"password": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "OpenStack password of the root user.",
				DisplayAttrs: &framework.DisplayAttributes{
					Sensitive: true,
				},
			},
			"password_policy": {
				Type:        framework.TypeString,
				Description: "Name of the password policy to use to generate passwords for dynamic credentials.",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathCloudCreateUpdate,
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathCloudRead,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathCloudCreateUpdate,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathCloudDelete,
			},
		},
		HelpSynopsis:    pathCloudHelpSyn,
		HelpDescription: pathCloudHelpDes,
	}
}

func (b *backend) pathClouds() *framework.Path {
	return &framework.Path{
		Pattern: pathClouds,
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback: b.pathCloudList,
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathCloudList,
			},
		},
		HelpSynopsis:    pathCloudListHelpSyn,
		HelpDescription: pathCloudListHelpDesc,
	}
}

func (b *backend) pathCloudCreateUpdate(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)

	sCloud := b.getSharedCloud(name)

	cloudConfig, err := sCloud.getCloudConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	if cloudConfig == nil {
		cloudConfig = &OsCloud{
			Name: name,
		}
	}

	if authURL, ok := d.GetOk("auth_url"); ok {
		cloudConfig.AuthURL = authURL.(string)
	}
	if userDomainName, ok := d.GetOk("user_domain_name"); ok {
		cloudConfig.UserDomainName = userDomainName.(string)
	}
	if username, ok := d.GetOk("username"); ok {
		cloudConfig.Username = username.(string)
	}
	if password, ok := d.GetOk("password"); ok {
		cloudConfig.Password = password.(string)
	}
	if uTemplate, ok := d.GetOk("username_template"); ok {
		cloudConfig.UsernameTemplate = uTemplate.(string)
		// validate template first
		_, err := RandomTemporaryUsername(cloudConfig.UsernameTemplate, &roleEntry{})
		if err != nil {
			return logical.ErrorResponse("invalid username template: %s", err), nil
		}
	} else if r.Operation == logical.CreateOperation {
		cloudConfig.UsernameTemplate = d.Schema["username_template"].Default.(string)
	}
	if pwdPolicy, ok := d.GetOk("password_policy"); ok {
		cloudConfig.PasswordPolicy = pwdPolicy.(string)
	}

	sCloud.passwords = &Passwords{
		PolicyGenerator: b.System(),
		PolicyName:      cloudConfig.PasswordPolicy,
	}

	if err := cloudConfig.save(ctx, r.Storage); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	return nil, nil
}

func (b *backend) pathCloudRead(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)

	sCloud := b.getSharedCloud(name)

	cloudConfig, err := sCloud.getCloudConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"auth_url":          cloudConfig.AuthURL,
			"user_domain_name":  cloudConfig.UserDomainName,
			"username":          cloudConfig.Username,
			"password":          cloudConfig.Password,
			"username_template": cloudConfig.UsernameTemplate,
			"password_policy":   cloudConfig.PasswordPolicy,
		},
	}, nil
}

func (b *backend) pathCloudDelete(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)

	if err := r.Storage.Delete(ctx, storageCloudKey(name)); err != nil {
		return nil, fmt.Errorf("error deleting cloud: %w", err)
	}

	return nil, nil
}

func (b *backend) pathCloudList(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	clouds, err := r.Storage.List(ctx, cloudsStoragePath+"/")
	if err != nil {
		return nil, fmt.Errorf("error listing clouds: %w", err)
	}

	return logical.ListResponse(clouds), nil
}
