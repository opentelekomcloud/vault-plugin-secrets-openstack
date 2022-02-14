package openstack

import (
	"context"
	"sync"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/stretchr/testify/assert"
)

func TestSharedCloud_getCloudConfig(t *testing.T) {
	// TODO: this test must be implemented after implementing getCloudConfig
	cloud := &sharedCloud{
		client: new(gophercloud.ServiceClient),
		lock:   sync.Mutex{},
	}
	_, s := testBackend(t)

	_, err := cloud.getCloudConfig(context.Background(), s)
	assert.NoError(t, err)
}
