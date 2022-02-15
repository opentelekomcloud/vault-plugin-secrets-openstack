package openstack

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathCloud = "cloud"

	pathCloudHelpSyn = `bla-bla`
	pathCloudHelpDes = `bla-bla`
)

type OsCloud struct {
	Name           string
	AuthURL        string `json:"auth_url"`
	UserDomainName string `json:"user_domain_name"`
	Username       string `json:"username"`
	Password       string `json:"password"`
}

func getCloud(ctx context.Context, name string, s logical.Storage) (*OsCloud, error) {
	entry, err := s.Get(ctx, fmt.Sprintf("%s/%s", pathCloud, name))
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
	entry, err := logical.StorageEntryJSON(fmt.Sprintf("%s/%s", pathCloud, cloud.Name), cloud)
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
			"password": {
				Type:        framework.TypeString,
				Required:    true,
				Description: "OpenStack password of the root user.",
				DisplayAttrs: &framework.DisplayAttributes{
					Sensitive: true,
				},
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathCloudCreateUpdate,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathCloudCreateUpdate,
			},
		},
		HelpSynopsis:    pathCloudHelpSyn,
		HelpDescription: pathCloudHelpDes,
	}
}

func (b *backend) pathCloudCreateUpdate(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name := d.Get("name").(string)

	cloud, err := getCloud(ctx, name, r.Storage)
	if err != nil {
		return nil, err
	}

	if cloud == nil {
		cloud = &OsCloud{
			Name: name,
		}
	}

	if authURL, ok := d.GetOk("auth_url"); ok {
		cloud.AuthURL = authURL.(string)
	}
	if userDomainName, ok := d.GetOk("user_domain_name"); ok {
		cloud.UserDomainName = userDomainName.(string)
	}
	if username, ok := d.GetOk("username"); ok {
		cloud.Username = username.(string)
	}
	if password, ok := d.GetOk("password"); ok {
		cloud.Password = password.(string)
	}

	if err := cloud.save(ctx, r.Storage); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	return nil, nil
}

	"github.com/hashicorp/vault/sdk/logical"
)

const pathCloud = "cloud"

func cloudKey(name string) string {
	return fmt.Sprintf("%s/%s", pathCloud, name)
}

func (c *sharedCloud) getCloudConfig(ctx context.Context, s logical.Storage) (*OsCloud, error) {
	entry, err := s.Get(ctx, cloudKey(c.name))
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
