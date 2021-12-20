package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathConfig = "config"

	confHelpSyn = `Configure the OpenStack Secret backend.`
)

var (
	errEmptyConfigUpdate = errors.New("config not found during update operation")
	errReadingConfig     = errors.New("error reading OpenStack configuration")
	errWritingConfig     = errors.New("error storing OpenStack configuration")
	errDeleteConfig      = errors.New("error deleting OpenStack configuration")
	errEmptyConfig       = errors.New("config is empty")
)

type osConfig struct {
	AuthURL     string `json:"auth_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	ProjectName string `json:"project_name"`
	DomainName  string `json:"domain_name"`
	Region      string `json:"region"`
}

func (b *backend) getConfig(ctx context.Context, s logical.Storage) (*osConfig, error) {
	entry, err := s.Get(ctx, pathConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errReadingConfig, err.Error())
	}

	if entry == nil || len(entry.Value) == 0 {
		return nil, nil
	}

	cfg := new(osConfig)
	if err := entry.DecodeJSON(cfg); err != nil {
		return nil, fmt.Errorf("error decoding OpenStack configuration: %w", err)
	}
	return cfg, nil
}

func (b *backend) setConfig(ctx context.Context, config *osConfig, s logical.Storage) error {
	entry, err := logical.StorageEntryJSON(pathConfig, config)
	if err != nil {
		return fmt.Errorf("error creating OpenStack configuration JSON: %w", err)
	}

	if err := s.Put(ctx, entry); err != nil {
		return fmt.Errorf("%w: %s", errWritingConfig, err.Error())
	}

	// invalidate existing configuration
	b.reset()

	return nil
}

func (b *backend) pathConfig() *framework.Path {
	return &framework.Path{
		Pattern: pathConfig,
		Fields: map[string]*framework.FieldSchema{
			"auth_url": {
				Type:        framework.TypeString,
				Description: "The Identity authentication URL.",
				Required:    true,
			},
			"region": {
				Type:        framework.TypeString,
				Description: "The region to connect to.",
			},
			"username": {
				Type:        framework.TypeString,
				Description: "Username of the configured user",
			},
			"password": {
				Type:        framework.TypeString,
				Description: "The password of the configured user",
			},
			"project_name": {
				Type: framework.TypeString,
				Description: "The name of the Tenant (Identity v2) or Project (Identity v3) to login with.\n" +
					"This will change scope of generated token to the project.",
			},
			"domain_name": {
				Type:        framework.TypeString,
				Description: "The name of the Domain to scope to (Identity v3).",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathConfigRead,
			},
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
			},
			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathConfigDelete,
			},
		},
		ExistenceCheck: b.configExistenceCheck,
		HelpSynopsis:   confHelpSyn,
	}
}

func (b *backend) configExistenceCheck(ctx context.Context, r *logical.Request, _ *framework.FieldData) (bool, error) {
	config, err := b.getConfig(ctx, r.Storage)
	if err != nil {
		return false, err
	}

	return config != nil, err
}

func (b *backend) pathConfigRead(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	config, err := b.getConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return &logical.Response{}, nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"auth_url":     config.AuthURL,
			"region":       config.Region,
			"username":     config.Username,
			"domain_name":  config.DomainName,
			"project_name": config.ProjectName,
			"password":     config.Password,
		},
	}, nil
}

func (b *backend) pathConfigWrite(ctx context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	config, err := b.getConfig(ctx, r.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if r.Operation == logical.UpdateOperation {
			return nil, errEmptyConfigUpdate
		}
		config = new(osConfig)
	}

	if authURL, ok := d.GetOk("auth_url"); ok {
		config.AuthURL = authURL.(string)
	}
	if region, ok := d.GetOk("region"); ok {
		config.Region = region.(string)
	}
	if username, ok := d.GetOk("username"); ok {
		config.Username = username.(string)
	}
	if password, ok := d.GetOk("password"); ok {
		config.Password = password.(string)
	}
	if projectName, ok := d.GetOk("project_name"); ok {
		config.ProjectName = projectName.(string)
	}
	if domainName, ok := d.GetOk("domain_name"); ok {
		config.DomainName = domainName.(string)
	}

	if err := b.setConfig(ctx, config, r.Storage); err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}

func (b *backend) pathConfigDelete(ctx context.Context, r *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	if err := r.Storage.Delete(ctx, pathConfig); err != nil {
		return nil, fmt.Errorf("%w: %s", errDeleteConfig, err.Error())
	}

	b.Invalidate(ctx, pathConfig)

	return &logical.Response{}, nil
}
