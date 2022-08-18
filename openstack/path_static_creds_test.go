package openstack

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	thClient "github.com/gophercloud/gophercloud/testhelper/client"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/require"
)

func credsStaticPath(name string) string {
	return fmt.Sprintf("%s/%s", "static-creds", name)
}

func TestStaticCredentialsRead_ok(t *testing.T) {
	userID, _ := uuid.GenerateUUID()
	secret, _ := uuid.GenerateUUID()
	projectName := tools.RandomString("p", 5)

	fixtures.SetupKeystoneMock(t, userID, projectName, fixtures.EnabledMocks{
		TokenPost: true,
		TokenGet:  true,
		UserList:  true,
	})

	testClient := thClient.ServiceClient()
	authURL := testClient.Endpoint + "v3"

	b, s := testBackend(t)
	cloudEntry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
		Name:             testCloudName,
		AuthURL:          authURL,
		UserDomainName:   testUserDomainName,
		Username:         testUsername,
		Password:         testPassword1,
		UsernameTemplate: testTemplate1,
	})
	require.NoError(t, err)

	t.Run("user_token", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomStaticRole(t, s, projectName, "token", secret)

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsStaticPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
	})
	t.Run("user_password", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomStaticRole(t, s, projectName, "password", secret)

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsStaticPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
	})
}

func TestStaticCredentialsRead_error(t *testing.T) {
	t.Run("read-fail", func(t *testing.T) {
		userID, _ := uuid.GenerateUUID()
		secret, _ := uuid.GenerateUUID()
		fixtures.SetupKeystoneMock(t, userID, "", fixtures.EnabledMocks{})

		b, s := testBackend(t, failVerbRead)

		roleName := createSaveRandomStaticRole(t, s, "", "token", secret)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Path:      credsStaticPath(roleName),
			Operation: logical.ReadOperation,
			Storage:   s,
		})
		require.Error(t, err)
	})

	type testCase struct {
		EnabledMocks fixtures.EnabledMocks
		ProjectName  string
		ServiceType  string
	}

	cases := map[string]testCase{
		"no-token-post": {
			EnabledMocks: fixtures.EnabledMocks{
				TokenGet: true,
			},
			ProjectName: tools.RandomString("p", 5),
			ServiceType: "token",
		},
		"no-token-get": {
			EnabledMocks: fixtures.EnabledMocks{
				TokenPost: true,
			},
			ProjectName: tools.RandomString("p", 5),
			ServiceType: "token",
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			data := data
			userID, _ := uuid.GenerateUUID()
			secret, _ := uuid.GenerateUUID()
			fixtures.SetupKeystoneMock(t, userID, data.ProjectName, data.EnabledMocks)

			b, s := testBackend(t)

			roleName := createSaveRandomStaticRole(t, s, data.ProjectName, data.ServiceType, secret)

			testClient := thClient.ServiceClient()
			authURL := testClient.Endpoint + "v3"

			cloudEntry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
				Name:             testCloudName,
				AuthURL:          authURL,
				UserDomainName:   testUserDomainName,
				Username:         testUsername,
				Password:         testPassword1,
				UsernameTemplate: testTemplate1,
			})
			require.NoError(t, err)
			require.NoError(t, s.Put(context.Background(), cloudEntry))

			_, err = b.HandleRequest(context.Background(), &logical.Request{
				Operation: logical.ReadOperation,
				Path:      credsStaticPath(roleName),
				Storage:   s,
			})
			require.Error(t, err)
		})
	}
}

func createSaveRandomStaticRole(t *testing.T, s logical.Storage, projectName, sType string, secret string) string {
	roleName := randomRoleName()
	role := map[string]interface{}{
		"name":         roleName,
		"cloud":        testCloudName,
		"project_name": projectName,
		"domain_id":    tools.RandomString("d", 5),
		"secret_type":  sType,
		"secret":       secret,
		"username":     roleName,
	}
	saveRawStaticRole(t, roleName, role, s)

	return roleName
}
