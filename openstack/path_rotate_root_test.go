package openstack

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	thClient "github.com/gophercloud/gophercloud/testhelper/client"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/require"
)

func TestRotateRootCredentials_ok(t *testing.T) {
	userID, _ := uuid.GenerateUUID()
	fixtures.SetupKeystoneMock(t, userID, fixtures.EnabledMocks{TokenPost: true, TokenGet: true, PasswordChange: true})

	b, s := testBackend(t)

	cloud := &sharedCloud{name: tools.RandomString("cl", 5)}

	testClient := thClient.ServiceClient()
	authURL := testClient.Endpoint + "v3"

	entry, err := logical.StorageEntryJSON(storageCloudKey(cloud.name), OsCloud{
		Name:           cloud.name,
		AuthURL:        authURL,
		Username:       tools.RandomString("u", 5) + "r",
		Password:       tools.MakeNewPassword(""),
		UserDomainName: tools.RandomString("d", 5),
	})
	require.NoError(t, err)
	require.NoError(t, s.Put(context.Background(), entry))

	_, err = b.HandleRequest(context.Background(), &logical.Request{
		Path:      "rotate-root/" + cloud.name,
		Operation: logical.ReadOperation,
		Storage:   s,
	})
	require.NoError(t, err)
}

func TestRotateRootCredentials_error(t *testing.T) {
	t.Parallel()

	t.Run("read-fail", func(t *testing.T) {
		userID, _ := uuid.GenerateUUID()
		fixtures.SetupKeystoneMock(t, userID, fixtures.EnabledMocks{})

		b, s := testBackend(t, failVerbRead)

		cloud := &sharedCloud{name: tools.RandomString("cl", 5)}

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Path:      "rotate-root/" + cloud.name,
			Operation: logical.ReadOperation,
			Storage:   s,
		})
		require.Error(t, err)
	})

	cases := map[string]fixtures.EnabledMocks{
		"no-change": {
			TokenPost: true, TokenGet: true,
		},
		"no-post": {
			TokenGet: true, PasswordChange: true,
		},
		"no-get": {
			TokenPost: true, PasswordChange: true,
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			data := data
			userID, _ := uuid.GenerateUUID()
			fixtures.SetupKeystoneMock(t, userID, data)

			b, s := testBackend(t)

			cloud := &sharedCloud{name: tools.RandomString("cl", 5)}

			testClient := thClient.ServiceClient()
			authURL := testClient.Endpoint + "v3"

			entry, err := logical.StorageEntryJSON(storageCloudKey(cloud.name), OsCloud{
				Name:           cloud.name,
				AuthURL:        authURL,
				Username:       tools.RandomString("u", 5) + "r",
				Password:       tools.MakeNewPassword(""),
				UserDomainName: tools.RandomString("d", 5),
			})
			require.NoError(t, err)
			require.NoError(t, s.Put(context.Background(), entry))

			_, err = b.HandleRequest(context.Background(), &logical.Request{
				Path:      "rotate-root/" + cloud.name,
				Operation: logical.ReadOperation,
				Storage:   s,
			})
			require.Error(t, err)
		})
	}
}
