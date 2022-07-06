package openstack

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rolePath(name string) string {
	return fmt.Sprintf("%s/%s", "role", name)
}

func TestRoleStoragePath(t *testing.T) {
	name := tools.RandomString("role", 5)
	expected := "roles/" + name
	actual := roleStoragePath(name)
	assert.Equal(t, actual, expected)
}

func randomRoleName() string {
	return tools.RandomString("k", 5) + "m"
}

func expectedRoleData(cloudName string) (*roleEntry, map[string]interface{}) {
	expTTL := time.Hour
	expected := &roleEntry{
		Cloud:       cloudName,
		TTL:         expTTL / time.Second,
		ProjectName: tools.RandomString("p", 5),
		DomainName:  tools.RandomString("d", 5),
	}
	expectedMap := map[string]interface{}{
		"cloud":        expected.Cloud,
		"ttl":          expTTL,
		"project_id":   "",
		"project_name": expected.ProjectName,
		"domain_id":    "",
		"domain_name":  expected.DomainName,
		"extensions":   map[string]string{},
		"root":         false,
		"secret_type":  "token",
		"user_groups":  []string{},
		"user_roles":   []string{},
	}
	return expected, expectedMap
}

func saveRawRole(t *testing.T, name string, raw map[string]interface{}, s logical.Storage) {
	storeEntry, err := logical.StorageEntryJSON(roleStoragePath(name), raw)
	require.NoError(t, err)
	require.NoError(t, s.Put(context.Background(), storeEntry))
}

func TestRoleGet(t *testing.T) {
	t.Parallel()

	t.Run("existing", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)

		roleName := randomRoleName()
		_, expectedMap := expectedRoleData(randomRoleName())

		saveRawRole(t, roleName, expectedMap, s)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		assert.Equal(t, expectedMap, resp.Data)
	})

	t.Run("not-existing", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)
		roleName := tools.RandomString("k", 5)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Data["error"])
	})

	t.Run("get-err", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t, failVerbRead)
		roleName := tools.RandomString("k", 5)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ReadOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, errRoleGet)
	})
}

func TestRoleExistence(t *testing.T) {
	t.Parallel()

	t.Run("existing", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)

		roleName := randomRoleName()
		_, exp := expectedRoleData(randomRoleName())
		saveRawRole(t, roleName, exp, s)

		req := &logical.Request{Storage: s}
		fData := &framework.FieldData{
			Schema: b.pathRole().Fields,
			Raw:    map[string]interface{}{"name": roleName},
		}
		ok, err := b.roleExistenceCheck(context.Background(), req, fData)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("not-existing", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)

		roleName := randomRoleName()

		req := &logical.Request{Storage: s}
		fData := &framework.FieldData{
			Schema: b.pathRole().Fields,
			Raw:    map[string]interface{}{"name": roleName},
		}
		ok, err := b.roleExistenceCheck(context.Background(), req, fData)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("get-err", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t, failVerbRead)

		roleName := randomRoleName()

		req := &logical.Request{Storage: s}
		fData := &framework.FieldData{
			Schema: b.pathRole().Fields,
			Raw:    map[string]interface{}{"name": roleName},
		}
		_, err := b.roleExistenceCheck(context.Background(), req, fData)
		assert.Error(t, err, errRoleGet)
	})
}

func TestRoleList(t *testing.T) {
	t.Parallel()

	t.Run("ok", func(t *testing.T) {
		b, s := testBackend(t)
		roleCount := tools.RandomInt(1, 10)
		roleNames := make([]string, roleCount)

		for i := 0; i < roleCount; i++ {
			name := randomRoleName()
			roleNames[i] = name
			_, exp := expectedRoleData(randomRoleName())
			saveRawRole(t, name, exp, s)
		}

		lst, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ListOperation,
			Path:      "roles/",
			Storage:   s,
		})
		require.NoError(t, err)
		require.NotEmpty(t, lst.Data)
		require.Len(t, lst.Data["keys"], roleCount)
		for _, name := range roleNames {
			assert.Contains(t, lst.Data["keys"], name)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t, failVerbList)
		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ListOperation,
			Path:      "roles/",
			Storage:   s,
		})
		require.Error(t, err)
	})

	t.Run("filter", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)
		name1 := randomRoleName()
		expRole1, expMap1 := expectedRoleData(randomRoleName())
		saveRawRole(t, name1, expMap1, s)
		name2 := randomRoleName()
		_, expMap2 := expectedRoleData(randomRoleName())
		saveRawRole(t, name2, expMap2, s)

		lst, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ListOperation,
			Path:      "roles/",
			Data: map[string]interface{}{
				"cloud": expRole1.Cloud,
			},
			Storage: s,
		})
		require.NoError(t, err)
		assert.Len(t, lst.Data["keys"], 1)
		assert.Equal(t, name1, lst.Data["keys"].([]string)[0])
	})

	t.Run("filter-get-err", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t, failVerbRead)
		name1 := randomRoleName()
		expRole1, expMap1 := expectedRoleData(randomRoleName())
		saveRawRole(t, name1, expMap1, s)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.ListOperation,
			Path:      "roles/",
			Data: map[string]interface{}{
				"cloud": expRole1.Cloud,
			},
			Storage: s,
		})
		require.Error(t, err)
	})
}

func TestRoleDelete(t *testing.T) {
	t.Parallel()

	t.Run("existing", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)

		roleName := randomRoleName()
		_, expectedMap := expectedRoleData(randomRoleName())
		saveRawRole(t, roleName, expectedMap, s)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.Empty(t, resp)
	})

	t.Run("not-existing", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)
		roleName := randomRoleName()
		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.NoError(t, err)
		require.Empty(t, resp)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t, failVerbDelete)

		roleName := randomRoleName()
		_, expectedMap := expectedRoleData(randomRoleName())
		saveRawRole(t, roleName, expectedMap, s)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.Error(t, err)
	})

	t.Run("error-get", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t, failVerbRead)

		roleName := randomRoleName()
		_, expectedMap := expectedRoleData(randomRoleName())
		saveRawRole(t, roleName, expectedMap, s)

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      rolePath(roleName),
			Storage:   s,
		})
		require.Error(t, err)
	})
}

func TestRoleCreate(t *testing.T) {
	t.Parallel()

	id, _ := uuid.GenerateUUID()
	t.Run("ok", func(t *testing.T) {

		b, s := testBackend(t)
		cloudName := preCreateCloud(t, s)

		cases := map[string]*roleEntry{
			"admin": {
				Name:  randomRoleName(),
				Cloud: cloudName,
				Root:  true,
			},
			"token": {
				Name:        randomRoleName(),
				Cloud:       cloudName,
				ProjectName: randomRoleName(),
				SecretType:  SecretToken,
				UserGroups:  []string{"default", "testing"},
			},
			"password": {
				Name:        randomRoleName(),
				Cloud:       cloudName,
				ProjectName: randomRoleName(),
				SecretType:  SecretPassword,
				UserGroups:  []string{"default", "testing"},
			},
			"ttl": {
				Name:        randomRoleName(),
				Cloud:       cloudName,
				ProjectName: randomRoleName(),
				SecretType:  SecretToken,
				UserGroups:  []string{"default", "testing"},
				TTL:         24 * time.Hour,
			},
			"endpoint-override": {
				Name:      randomRoleName(),
				Cloud:     cloudName,
				ProjectID: id,
				Extensions: map[string]string{
					"volume_api_version":             "3",
					"object_store_endpoint_override": "https://swift.example.com",
				},
			},
		}

		for name, data := range cases {
			t.Run(name, func(t *testing.T) {
				data := data
				t.Parallel()

				roleName := data.Name
				inputRole := fixtures.SanitizedMap(roleToMap(data))

				resp, err := b.HandleRequest(context.Background(), &logical.Request{
					Operation: logical.CreateOperation,
					Path:      rolePath(roleName),
					Data:      inputRole,
					Storage:   s,
				})
				require.NoError(t, err)
				require.Empty(t, resp)

				entry, err := s.Get(context.Background(), roleStoragePath(roleName))
				require.NoError(t, err)
				require.NotEmpty(t, entry)
				role := new(roleEntry)
				assert.NoError(t, entry.DecodeJSON(role))

				fillRoleDefaultFields(b, data) // otherwise there will be false positives
				assert.Equal(t, data, role)
			})
		}
	})

	t.Run("error", func(t *testing.T) {
		type errRoleEntry struct {
			*roleEntry
			errorRegex *regexp.Regexp
		}

		b, s := testBackend(t)
		cloudName := preCreateCloud(t, s)

		notForRootRe := regexp.MustCompile(`impossible to set .+ for the root user`)
		cases := map[string]*errRoleEntry{
			"root-ttl": {
				roleEntry: &roleEntry{
					Cloud: cloudName,
					Root:  true,
					TTL:   1 * time.Hour,
				},
				errorRegex: notForRootRe,
			},
			"root-password": {
				roleEntry: &roleEntry{
					Cloud:      cloudName,
					Root:       true,
					SecretType: SecretPassword,
				},
				errorRegex: notForRootRe,
			},
			"root-user-groups": {
				roleEntry: &roleEntry{
					Cloud:      cloudName,
					Root:       true,
					UserGroups: []string{"ug-1"},
				},
				errorRegex: notForRootRe,
			},
			"root-user-roles": {
				roleEntry: &roleEntry{
					Cloud:     cloudName,
					Root:      true,
					UserRoles: []string{"ur-1"},
				},
				errorRegex: notForRootRe,
			},
			"without-cloud": {
				roleEntry:  &roleEntry{},
				errorRegex: regexp.MustCompile(`cloud is required when creating a role`),
			},
		}

		for name, data := range cases {
			t.Run(name, func(t *testing.T) {
				data := data
				t.Parallel()

				roleName := randomRoleName()
				inputRole := fixtures.SanitizedMap(roleToMap(data.roleEntry))

				resp, err := b.HandleRequest(context.Background(), &logical.Request{
					Operation: logical.CreateOperation,
					Path:      rolePath(roleName),
					Data:      inputRole,
					Storage:   s,
				})
				require.NoError(t, err)
				require.True(t, resp.IsError())
				assert.Regexp(t, data.errorRegex, resp.Data["error"])
			})
		}
	})

	t.Run("not-existing-cloud", func(t *testing.T) {
		t.Parallel()
		b, s := testBackend(t)

		role := &roleEntry{
			Name:  randomRoleName(),
			Cloud: randomRoleName(),
		}
		inputRole := fixtures.SanitizedMap(roleToMap(role))

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.CreateOperation,
			Path:      rolePath(role.Name),
			Data:      inputRole,
			Storage:   s,
		})
		require.NoError(t, err)
		require.True(t, resp.IsError())
		assert.Regexp(t, regexp.MustCompile(`cloud .+ doesn't exist`), resp.Data["error"])
	})

	t.Run("save-store-err", func(t *testing.T) {
		_, s := testBackend(t, failVerbPut)
		t.Parallel()

		d, _ := expectedRoleData(randomRoleName())
		err := saveRole(context.Background(), d, s)
		require.Error(t, err)
	})
}

func preCreateCloud(t *testing.T, s logical.Storage) string {
	t.Helper()

	name := randomRoleName()
	cloudStoragePath := storageCloudKey(name)

	entry, err := logical.StorageEntryJSON(cloudStoragePath, &OsCloud{
		Name:           testCloudName,
		AuthURL:        testAuthURL,
		UserDomainName: testUserDomainName,
		Username:       testUsername,
		Password:       testPassword1,
	})
	require.NoError(t, err)
	require.NoError(t, s.Put(context.Background(), entry))

	t.Cleanup(func() {
		require.NoError(t, s.Delete(context.Background(), cloudStoragePath))
	})
	return name
}

func TestRoleUpdate(t *testing.T) {
	t.Parallel()

	b, s := testBackend(t)
	cloudName := preCreateCloud(t, s)

	t.Run("ok", func(t *testing.T) {
		roleName := randomRoleName()
		_, exp := expectedRoleData(randomRoleName())
		exp2 := &roleEntry{
			Cloud:       cloudName,
			ProjectID:   "",
			ProjectName: tools.RandomString("p", 5),
		}
		saveRawRole(t, roleName, exp, s)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      rolePath(roleName),
			Data:      fixtures.SanitizedMap(roleToMap(exp2)),
			Storage:   s,
		})
		require.NoError(t, err)
		assert.False(t, resp.IsError(), resp)
	})

	t.Run("not-existing", func(t *testing.T) {
		roleName := randomRoleName()
		_, exp := expectedRoleData(cloudName)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      rolePath(roleName),
			Data:      exp,
			Storage:   s,
		})
		require.NoError(t, err)
		assert.True(t, resp.IsError())
		assert.Regexp(t, regexp.MustCompile(`role .+ not found during update operation`), resp.Data["error"])
	})
}

func fillRoleDefaultFields(b *backend, entry *roleEntry) {
	pr := b.pathRole()
	flds := pr.Fields
	if entry.SecretType == "" {
		entry.SecretType = flds["secret_type"].Default.(secretType)
	}
	if !entry.Root {
		if entry.TTL == 0 {
			entry.TTL = time.Hour
		}
	}
	entry.TTL /= time.Second
}
