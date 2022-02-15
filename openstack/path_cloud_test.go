package openstack

import (
	"context"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	th "github.com/gophercloud/gophercloud/testhelper"
	thClient "github.com/gophercloud/gophercloud/testhelper/client"
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
	th.SetupHTTP()
	defer th.TeardownHTTP()

	testClient := thClient.ServiceClient()

	t.Run("Success", func(t *testing.T) {
		b, storage := testBackend(t)

		authURL := testClient.Endpoint + "v3"

		entry, err := logical.StorageEntryJSON(cloudKey(testCloudName), &OsCloud{
			Name:           testCloudName,
			AuthURL:        authURL,
			UserDomainName: testUserDomainName,
			Username:       testUsername,
			Password:       testPassword1,
		})
		assert.NoError(t, err)
		assert.NoError(t, storage.Put(context.Background(), entry))

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      cloudKey(testCloudName),
		})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		b, storage := testBackend(t)

		res, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      cloudKey(testCloudName),
		})
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("FullConfig", func(t *testing.T) {
		b, storage := testBackend(t)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      cloudKey(testCloudName),
			Data: map[string]interface{}{
				"name":             testCloudName,
				"auth_url":         testAuthURL,
				"user_domain_name": testUserDomainName,
				"username":         testUsername,
				"password":         testPassword1,
			},
		})
		assert.NoError(t, err)

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		assert.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.UserDomainName, testUserDomainName)
		assert.Equal(t, cloudConfig.Username, testUsername)
		assert.Equal(t, cloudConfig.Password, testPassword1)
		assert.Equal(t, cloudConfig.Name, testCloudName)
	})

	t.Run("FullConfigUpdate", func(t *testing.T) {
		b, storage := testBackend(t)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      cloudKey(testCloudName),
			Data: map[string]interface{}{
				"name":             testCloudName,
				"auth_url":         testAuthURL,
				"user_domain_name": testUserDomainName,
				"username":         testUsername,
				"password":         testPassword1,
			},
		})
		assert.NoError(t, err)

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		assert.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.UserDomainName, testUserDomainName)
		assert.Equal(t, cloudConfig.Username, testUsername)
		assert.Equal(t, cloudConfig.Password, testPassword1)
		assert.Equal(t, cloudConfig.Name, testCloudName)

		_, err = b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      cloudKey(testCloudName),
			Data: map[string]interface{}{
				"name":             testCloudName,
				"auth_url":         testAuthURL,
				"user_domain_name": testUserDomainName,
				"username":         testUsername,
				"password":         testPassword2,
			},
		})
		assert.NoError(t, err)

		cloudConfig, err = sCloud.getCloudConfig(context.Background(), storage)
		assert.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.UserDomainName, testUserDomainName)
		assert.Equal(t, cloudConfig.Username, testUsername)
		assert.Equal(t, cloudConfig.Password, testPassword2)
		assert.Equal(t, cloudConfig.Name, testCloudName)
	})
}
