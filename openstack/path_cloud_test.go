package openstack

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

var (
	testCloudName      = strings.ToLower(tools.RandomString("cloud", 3))
	testAuthURL        = tools.RandomString("https://test.com/", 3)
	testUsername       = tools.RandomString("user", 3)
	testUserDomainName = tools.RandomString("domain", 3)
	testPassword1      = tools.RandomString("password1", 3)
	testPassword2      = tools.RandomString("password2", 3)
)

func TestCloudCreate(t *testing.T) {
	t.Run("EmptyConfig", func(t *testing.T) {
		b, storage := testBackend(t)

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      pathCloudKey(testCloudName),
		})
		require.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Create", func(t *testing.T) {
		b, storage := testBackend(t)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      pathCloudKey(testCloudName),
			Data: map[string]interface{}{
				"name":             testCloudName,
				"auth_url":         testAuthURL,
				"user_domain_name": testUserDomainName,
				"username":         testUsername,
				"password":         testPassword1,
			},
		})
		require.NoError(t, err)

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		require.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.UserDomainName, testUserDomainName)
		assert.Equal(t, cloudConfig.Username, testUsername)
		assert.Equal(t, cloudConfig.Password, testPassword1)
		assert.Equal(t, cloudConfig.Name, testCloudName)
	})

	t.Run("Update", func(t *testing.T) {
		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
			Name:           testCloudName,
			AuthURL:        testAuthURL,
			UserDomainName: testUserDomainName,
			Username:       testUsername,
			Password:       testPassword1,
		})
		require.NoError(t, err)
		require.NoError(t, storage.Put(context.Background(), entry))

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.Password, testPassword1)

		_, err = b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      pathCloudKey(testCloudName),
			Data: map[string]interface{}{
				"password": testPassword2,
			},
		})
		require.NoError(t, err)

		cloudConfig, err = sCloud.getCloudConfig(context.Background(), storage)
		require.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.UserDomainName, testUserDomainName)
		assert.Equal(t, cloudConfig.Username, testUsername)
		assert.Equal(t, cloudConfig.Password, testPassword2)
		assert.Equal(t, cloudConfig.Name, testCloudName)
	})

	t.Run("Read", func(t *testing.T) {
		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
			Name:           testCloudName,
			AuthURL:        testAuthURL,
			UserDomainName: testUserDomainName,
			Username:       testUsername,
			Password:       testPassword1,
		})
		require.NoError(t, err)
		require.NoError(t, storage.Put(context.Background(), entry))

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.Password, testPassword1)

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      pathCloudKey(testCloudName),
		})
		require.NoError(t, err)
		assert.Equal(t, res.Data["auth_url"], testAuthURL)
		assert.Equal(t, res.Data["user_domain_name"], testUserDomainName)
		assert.Equal(t, res.Data["username"], testUsername)
		assert.Equal(t, res.Data["password"], testPassword1)
	})

	t.Run("Delete", func(t *testing.T) {
		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
			Name:           testCloudName,
			AuthURL:        testAuthURL,
			UserDomainName: testUserDomainName,
			Username:       testUsername,
			Password:       testPassword1,
		})
		require.NoError(t, err)
		require.NoError(t, storage.Put(context.Background(), entry))

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.Password, testPassword1)

		_, err = b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.DeleteOperation,
			Path:      pathCloudKey(testCloudName),
		})
		require.NoError(t, err)
	})

	t.Run("List", func(t *testing.T) {
		b, storage := testBackend(t)

		cloudCount := tools.RandomInt(1, 10)

		for i := 0; i < cloudCount; i++ {
			name := strings.ToLower(tools.RandomString("name", 3))

			tmpCloud := &OsCloud{
				Name:           name,
				AuthURL:        testAuthURL,
				UserDomainName: testUserDomainName,
				Username:       testUsername,
				Password:       testPassword1,
			}
			require.NoError(t, tmpCloud.save(context.Background(), storage))
		}

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ListOperation,
			Path:      "clouds/",
		})
		require.NoError(t, err)
		assert.Len(t, res.Data["keys"], cloudCount)
	})
}
