package openstack

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

type failVerb int

const (
	failVerbRead failVerb = iota
	failVerbPut
	failVerbList
	failVerbDelete
)

func testBackend(t *testing.T, fvs ...failVerb) (*backend, logical.Storage) {
	t.Helper()

	storageView := new(logical.InmemStorage)
	for _, fv := range fvs {
		switch fv {
		case failVerbRead:
			storageView.Underlying().FailGet(true)
		case failVerbPut:
			storageView.Underlying().FailPut(true)
		case failVerbList:
			storageView.Underlying().FailList(true)
		case failVerbDelete:
			storageView.Underlying().FailDelete(true)
		}
	}

	config := logical.TestBackendConfig()
	config.StorageView = storageView
	config.Logger = hclog.NewNullLogger()

	b, err := Factory(context.Background(), config)
	assert.NoError(t, err)

	return b.(*backend), config.StorageView
}
