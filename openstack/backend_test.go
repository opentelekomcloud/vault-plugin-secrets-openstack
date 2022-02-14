package openstack

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	th "github.com/gophercloud/gophercloud/testhelper"
	thClient "github.com/gophercloud/gophercloud/testhelper/client"
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

func TestBackend_sharedCloud(t *testing.T) {
	expected := &sharedCloud{
		client: new(gophercloud.ServiceClient),
		lock:   sync.Mutex{},
	}
	cloudKey := tools.RandomString("cl", 5)
	back := backend{
		clouds: map[string]*sharedCloud{
			cloudKey: expected,
		},
	}

	t.Run("existing", func(t *testing.T) {
		actual := back.getSharedCloud(cloudKey)
		assert.Equal(t, expected, actual)
	})

	t.Run("non-existing", func(t *testing.T) {
		actual := back.getSharedCloud("no")
		assert.NotEqual(t, expected, actual)
		assert.Empty(t, actual.client)
	})
}

func TestSharedCloud_client(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	testClient := thClient.ServiceClient()
	_, s := testBackend(t)

	t.Run("existing-client", func(t *testing.T) {
		cloud := &sharedCloud{
			client: thClient.ServiceClient(),
			lock:   sync.Mutex{},
		}
		client, err := cloud.getClient(context.Background(), s)
		assert.NoError(t, err)
		assert.Equal(t, testClient, client)
	})

	t.Run("new-client", func(t *testing.T) {
		th.Mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, "POST")
			th.TestHeader(t, r, "Content-Type", "application/json")
			th.TestHeader(t, r, "Accept", "application/json")

			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{
			"token": {
				"expires_at": "2014-10-02T13:45:00.000000Z"
			}
		}`)
		})

		cloud := &sharedCloud{name: tools.RandomString("cl", 5)}
		authURL := testClient.Endpoint + "v3"

		entry, err := logical.StorageEntryJSON(cloudKey(cloud.name), OsCloud{
			AuthURL:        authURL,
			Username:       tools.RandomString("u", 5),
			Password:       tools.RandomString("p", 5),
			UserDomainName: tools.RandomString("d", 5),
		})
		assert.NoError(t, err)
		assert.NoError(t, s.Put(context.Background(), entry))

		_, err = cloud.getClient(context.Background(), s)
		assert.NoError(t, err)
	})
}
