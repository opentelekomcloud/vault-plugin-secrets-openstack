package openstack

import (
	"context"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var (
	testCloudName      = strings.ToLower(tools.RandomString("cloud", 3))
	testAuthURL        = tools.RandomString("https://test.com/", 3)
	testUsername       = tools.RandomString("user", 3)
	testUserDomainName = tools.RandomString("domain", 3)
	testPassword1      = tools.RandomString("password1", 3)
	testPassword2      = tools.RandomString("password2", 3)
	testTemplate1      = "asdf{{random 4}}"
	testTemplate2      = "u-{{ .RoleName }}-{{ random 5 }}"
	testPolicy1        = "default"
	testPolicy2        = "openstack"
)

func TestLifecyle(t *testing.T) {
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
				"name":              testCloudName,
				"auth_url":          testAuthURL,
				"user_domain_name":  testUserDomainName,
				"username":          testUsername,
				"password":          testPassword1,
				"username_template": testTemplate1,
				"password_policy":   testPolicy1,
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
		assert.Equal(t, cloudConfig.PasswordPolicy, testPolicy1)
	})

	t.Run("Update", func(t *testing.T) {
		b, storage := testBackend(t)

		entry, err := logical.StorageEntryJSON(storageCloudKey(testCloudName), &OsCloud{
			Name:             testCloudName,
			AuthURL:          testAuthURL,
			UserDomainName:   testUserDomainName,
			Username:         testUsername,
			Password:         testPassword1,
			UsernameTemplate: testTemplate1,
			PasswordPolicy:   testPolicy1,
		})
		require.NoError(t, err)
		require.NoError(t, storage.Put(context.Background(), entry))

		sCloud := b.getSharedCloud(testCloudName)
		cloudConfig, err := sCloud.getCloudConfig(context.Background(), storage)
		require.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.Password, testPassword1)
		assert.Equal(t, cloudConfig.UsernameTemplate, testTemplate1)
		assert.Equal(t, cloudConfig.PasswordPolicy, testPolicy1)

		r, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      pathCloudKey(testCloudName),
			Data: map[string]interface{}{
				"password":          testPassword2,
				"username_template": testTemplate2,
				"password_policy":   testPolicy2,
			},
		})
		require.NoError(t, err)
		require.False(t, r.IsError(), "update failed: %s", r.Error())

		cloudConfig, err = sCloud.getCloudConfig(context.Background(), storage)
		require.NoError(t, err)
		assert.Equal(t, cloudConfig.AuthURL, testAuthURL)
		assert.Equal(t, cloudConfig.UserDomainName, testUserDomainName)
		assert.Equal(t, cloudConfig.Username, testUsername)
		assert.Equal(t, cloudConfig.Password, testPassword2)
		assert.Equal(t, cloudConfig.Name, testCloudName)
		assert.Equal(t, cloudConfig.UsernameTemplate, testTemplate2)
		assert.Equal(t, cloudConfig.PasswordPolicy, testPolicy2)
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
		require.NoError(t, err)
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
		require.NoError(t, err)
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

func TestConfig(t *testing.T) {
	b, s := testBackend(t)

	tests := []struct {
		name     string
		config   map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "root_password_ttl defaults to 6 months",
			config: map[string]interface{}{
				"auth_url":          "https://test-001.com/v3",
				"username":          "test-username-1",
				"user_domain_name":  "testUserDomainName",
				"password":          "testUserPassword",
				"username_template": "user-{{ .RoleName }}-{{ random 4 }}",
			},
			expected: map[string]interface{}{
				"auth_url":          "https://test-001.com/v3",
				"username":          "test-username-1",
				"user_domain_name":  "testUserDomainName",
				"username_template": "user-{{ .RoleName }}-{{ random 4 }}",
				"root_password_ttl": 15768000,
				"password_policy":   "",
			},
		},
		{
			name: "root_password_ttl is provided",
			config: map[string]interface{}{
				"auth_url":          "https://test-001.com/v3",
				"username":          "test-username-2",
				"user_domain_name":  "testUserDomainName",
				"password":          "testUserPassword",
				"root_password_ttl": "1m",
			},
			expected: map[string]interface{}{
				"auth_url":          "https://test-001.com/v3",
				"username":          "test-username-2",
				"user_domain_name":  "testUserDomainName",
				"password_policy":   "",
				"root_password_ttl": 60,
				"username_template": "vault{{random 8 | lowercase}}"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var cloudName = strings.ToLower(tools.RandomString("cloud", 3))
			testConfigCreateUpdate(t, b, s, tc.config, cloudName)
			testConfigRead(t, b, s, tc.expected, cloudName)

			// Test that updating one element retains the others
			tc.expected["user_domain_name"] = "800e371d-ee51-4145-9ac8-5c43e4ceb79b"
			configSubset := map[string]interface{}{
				"user_domain_name": "800e371d-ee51-4145-9ac8-5c43e4ceb79b",
			}

			testConfigCreateUpdate(t, b, s, configSubset, cloudName)
			testConfigRead(t, b, s, tc.expected, cloudName)
		})
	}
}

func testConfigCreateUpdate(t *testing.T, b logical.Backend, s logical.Storage, expected map[string]interface{}, name string) {
	t.Helper()
	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Storage:   s,
		Operation: logical.CreateOperation,
		Path:      pathCloudKey(name),
		Data:      expected,
	})
	require.NoError(t, err)
}

func testConfigRead(t *testing.T, b logical.Backend, s logical.Storage, expected map[string]interface{}, name string) {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Storage:   s,
		Operation: logical.ReadOperation,
		Path:      pathCloudKey(name),
	})
	require.NoError(t, err)

	expected["next_rotation"] = resp.Data["next_rotation"]
	assert.Equal(t, expected, resp.Data)
}
