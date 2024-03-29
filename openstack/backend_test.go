package openstack

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	th "github.com/gophercloud/gophercloud/testhelper"
	thClient "github.com/gophercloud/gophercloud/testhelper/client"
	"github.com/hashicorp/go-hclog"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

type failVerb int

const (
	failVerbRead failVerb = iota
	failVerbPut
	failVerbList
	failVerbDelete
	defaultLeaseTTLHr = 1 * time.Hour
	maxLeaseTTLHr     = 12 * time.Hour
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
	config.System = logical.TestSystemView()
	config.Logger = hclog.NewNullLogger()

	b, err := Factory(context.Background(), config)
	assert.NoError(t, err)

	assert.NoError(t, b.Setup(context.Background(), config))

	return b.(*backend), config.StorageView
}

func TestBackend_sharedCloud(t *testing.T) {
	expected := &sharedCloud{
		client:    new(gophercloud.ServiceClient),
		passwords: new(Passwords),
		lock:      sync.Mutex{},
	}
	cloudKey := tools.RandomString("cl", 5)
	back := backend{
		clouds: map[string]*sharedCloud{
			cloudKey: expected,
		},
		Backend: &framework.Backend{},
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
			client:    thClient.ServiceClient(),
			expiresAt: time.Now().Add(time.Hour),
			lock:      sync.Mutex{},
		}

		client, err := cloud.getClient(context.Background(), s)
		assert.NoError(t, err)
		assert.Equal(t, testClient, client)
	})

	t.Run("new-client", func(t *testing.T) {
		authURL := testClient.Endpoint + "v3"

		th.Mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				th.TestMethod(t, r, "POST")
				th.TestHeader(t, r, "Content-Type", "application/json")
				th.TestHeader(t, r, "Accept", "application/json")

				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprintf(w, `
{
  "token": {
    "expires_at": "2014-10-02T13:45:00.000000Z",
    "catalog": [
      {
        "endpoints": [
          {
            "id": "id",
            "interface": "public",
            "region": "RegionOne",
            "region_id": "RegionOne",
            "url": "%s"
          }
        ],
        "id": "idk",
        "name": "keystone",
        "type": "identity"
      }
    ]
  }
}
`, authURL)
			case "GET":
				th.TestMethod(t, r, "GET")

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `
{
  "token": {
    "expires_at": "2023-10-02T13:45:00.000000Z"
  }
}
`)
			}
		})

		cloud := &sharedCloud{name: tools.RandomString("cl", 5)}

		entry, err := logical.StorageEntryJSON(storageCloudKey(cloud.name), OsCloud{
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

func TestPeriodicFuncNilConfig(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	b, _ := testBackend(t)

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLHr,
			MaxLeaseTTLVal:     maxLeaseTTLHr,
		},
		StorageView: &logical.InmemStorage{},
	}
	err := b.Setup(context.Background(), config)
	if err != nil {
		t.Fatalf("unable to create backend: %v", err)
	}

	err = b.periodicFunc(context.Background(), &logical.Request{
		Storage: config.StorageView,
	})

	if err != nil {
		t.Fatalf("periodicFunc error not nil: %v", err)
	}
}
