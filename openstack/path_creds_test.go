package openstack

import (
	"context"
	"fmt"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	thClient "github.com/gophercloud/gophercloud/testhelper/client"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func credsPath(name string) string {
	return fmt.Sprintf("%s/%s", "creds", name)
}

func TestCredentialsRead(t *testing.T) {
	userID, _ := uuid.GenerateUUID()
	fixtures.SetupKeystoneMock(t, userID, fixtures.EnabledMocks{
		TokenPost:   true,
		TokenDelete: true,
		UserPost:    true,
		UserDelete:  true,
	})

	testClient := thClient.ServiceClient()
	authURL := testClient.Endpoint + "v3"

	b, s := testBackend(t)
	cloudEntry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
		Name:           testCloudName,
		AuthURL:        authURL,
		UserDomainName: testUserDomainName,
		Username:       testUsername,
		Password:       testPassword1,
	})
	require.NoError(t, err)

	t.Run("root_password", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomRole(t, s, true, "password")

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
		assert.Equal(t, res.Data["username"], testUsername)
		assert.Equal(t, res.Data["password"], testPassword1)
	})
	t.Run("root_token", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomRole(t, s, true, "token")

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
		assert.NotEmpty(t, res.Data["expires_at"])
	})
	t.Run("user_token", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomRole(t, s, false, "token")

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
	})
	t.Run("user_password", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomRole(t, s, false, "password")

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
	})
	t.Run("root_token_revoke", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomRole(t, s, true, "token")

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
		require.Equal(t, res.Data["token"], testClient.TokenID)

		res, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    res.Secret,
			Data:      res.Data,
			Storage:   s,
		})
		require.NoError(t, err)
	})
	t.Run("user_password_revoke", func(t *testing.T) {
		require.NoError(t, s.Put(context.Background(), cloudEntry))

		roleName := createSaveRandomRole(t, s, false, "password")

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      credsPath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.Data)
		require.NotEmpty(t, res.Data["password"])
		require.NotEmpty(t, res.Secret.InternalData["user_id"])

		res, err = b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.RevokeOperation,
			Secret:    res.Secret,
			Data:      res.Data,
			Storage:   s,
		})
		require.NoError(t, err)
	})
}

func createSaveRandomRole(t *testing.T, s logical.Storage, root bool, sType string) string {
	roleName := randomRoleName()
	role := map[string]interface{}{
		"cloud":        testCloudName,
		"ttl":          time.Hour / time.Second,
		"project_name": tools.RandomString("p", 5),
		"root":         root,
		"secret_type":  sType,
	}
	saveRawRole(t, roleName, role, s)

	return roleName
}
