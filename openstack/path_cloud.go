package openstack

import (
	"context"
	"fmt"

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
