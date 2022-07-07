package openstack

import (
	"context"
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"time"
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

	DefaultUsernameTemplate = "vault{{random 8 | lowercase}}"
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

func secretRoot(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: backendSecretTypeRoot,
		Fields: map[string]*framework.FieldSchema{
			"password": {
				Type:        framework.TypeString,
				Description: "OpenStack password of the root user.",
			},
			"cloud": {
				Type:        framework.TypeString,
				Description: "Used cloud.",
			},
		},
		Renew: b.rootRenew,
	}
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
				Default:     DefaultUsernameTemplate,
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
			"password_expire": {
				Type:        framework.TypeDurationSecond,
				Description: "Specifies password expire duration for the root user as a string duration with time suffix.",
				Default:     "30d",
			},
			"validate_cloud": {
				Type:        framework.TypeBool,
				Description: "Specifies whether to try to authenticate with root credentials.",
				Default:     false,
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
		ExistenceCheck:  b.cloudExistenceCheck,
		HelpSynopsis:    pathCloudHelpSyn,
		HelpDescription: pathCloudHelpDes,
	}
}

func (b *backend) cloudExistenceCheck(ctx context.Context, r *logical.Request, d *framework.FieldData) (bool, error) {
	cloud := b.getSharedCloud(d.Get("name").(string))
	cloudCfg, err := cloud.getCloudConfig(ctx, r.Storage)
	if err != nil {
		return false, err
	}
	return cloudCfg != nil, nil
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
		cloudConfig.UsernameTemplate = DefaultUsernameTemplate
	}
	if pwdPolicy, ok := d.GetOk("password_policy"); ok {
		cloudConfig.PasswordPolicy = pwdPolicy.(string)
	}
	if passwordExpire, ok := d.GetOk("password_expire"); ok {
		cloudConfig.PasswordExpire = time.Duration(passwordExpire.(int))
	} else if r.Operation == logical.CreateOperation {
		cloudConfig.PasswordExpire = (30 * 24 * time.Hour) / time.Second
	}
	if validateCloud, ok := d.GetOk("validate_cloud"); ok {
		cloudConfig.ValidateCloud = validateCloud.(bool)
	} else if r.Operation == logical.CreateOperation {
		cloudConfig.ValidateCloud = false
	}

	sCloud.passwords = &Passwords{
		PolicyGenerator: b.System(),
		PolicyName:      cloudConfig.PasswordPolicy,
	}

	if cloudConfig.ValidateCloud {
		if err := validateRootCloud(cloudConfig); err != nil {
			return logical.ErrorResponse(err.Error()), nil
		}
	}

	if err := cloudConfig.save(ctx, r.Storage); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	return &logical.Response{
		Secret: &logical.Secret{
			LeaseOptions: logical.LeaseOptions{
				TTL:       cloudConfig.PasswordExpire,
				Renewable: true,
				IssueTime: time.Now(),
			},
			InternalData: map[string]interface{}{
				"secret_type": backendSecretTypeRoot,
				"password":    cloudConfig.Password,
				"cloud":       cloudConfig.Name,
			},
		},
	}, nil
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
			"username_template": cloudConfig.UsernameTemplate,
			"password_policy":   cloudConfig.PasswordPolicy,
			"password_expire":   cloudConfig.PasswordExpire,
			"validate_cloud":    cloudConfig.ValidateCloud,
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

func (b *backend) rootRenew(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	userIDRaw, ok := r.Data.InternalData["cloud"]
	if !ok {
		return nil, errors.New("internal data 'cloud' not found")
	}

	return &logical.Response{}, nil
}

func validateRootCloud(cloud *OsCloud) error {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: cloud.AuthURL,
		Username:         cloud.Username,
		Password:         cloud.Password,
		DomainName:       cloud.UserDomainName,
		Scope: &gophercloud.AuthScope{
			DomainName: cloud.UserDomainName,
		},
	}

	_, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return fmt.Errorf("error authenticate root user: %w", err)
	}

	return nil
}
