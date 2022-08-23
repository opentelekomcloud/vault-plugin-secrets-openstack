//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticRoleData struct {
	Cloud            string                 `json:"cloud"`
	TTL              time.Duration          `json:"ttl"`
	RotationDuration time.Duration          `json:"rotation_duration"`
	ProjectID        string                 `json:"project_id"`
	ProjectName      string                 `json:"project_name"`
	Extensions       map[string]interface{} `json:"extensions"`
	SecretType       string                 `json:"secret_type"`
	Username         string                 `json:"username"`
}

func extractStaticRoleData(t *testing.T, resp *http.Response) *staticRoleData {
	t.Helper()

	raw := readJSONResponse(t, resp)
	var out struct {
		Data *staticRoleData `json:"data"`
	}
	require.NoError(t, jsonutil.DecodeJSON([]byte(raw), &out))
	return out.Data
}

func (p *PluginTest) TestStaticRoleLifecycle() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)

	client, aux := openstackClient(t)

	dataCloud := map[string]interface{}{
		"auth_url":         cloud.AuthURL,
		"username":         cloud.Username,
		"password":         cloud.Password,
		"user_domain_name": cloud.UserDomainName,
	}

	resp, err := p.vaultDo(
		http.MethodPost,
		cloudURL(cloudName),
		dataCloud,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))

	createUserOpts := users.CreateOpts{
		Name:        "vault-test",
		Description: "Static user",
		DomainID:    aux.DomainID,
		Password:    openstack.RandomString(openstack.PwdDefaultSet, 16),
	}
	user, err := users.Create(client, createUserOpts).Extract()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, users.Delete(client, user.ID).ExtractErr())
	})

	data := expectedStaticRoleData(cloud.Name, aux)
	roleName := "test-write"
	t.Run("WriteRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodPost,
			staticRoleURL(roleName),
			data,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))
	})
	t.Run("ReadRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodGet,
			staticRoleURL(roleName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		expected := &staticRoleData{
			Cloud:            data["cloud"].(string),
			RotationDuration: data["rotation_duration"].(time.Duration),
			ProjectID:        data["project_id"].(string),
			ProjectName:      data["project_name"].(string),
			Extensions:       data["extensions"].(map[string]interface{}),
			SecretType:       data["secret_type"].(string),
			Username:         data["username"].(string),
		}
		assert.Equal(t, expected, extractStaticRoleData(t, resp))
	})

	t.Run("ListRoles", func(t *testing.T) {
		testListMethods(t, func(t *testing.T, m string) {
			resp, err := p.vaultDo(m, "/v1/openstack/static-roles", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			names := listResponseKeys(t, resp)
			require.Len(t, names, 1)
			assert.Equal(t, roleName, names[0])
		})
	})

	t.Run("DeleteRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodDelete,
			staticRoleURL(roleName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func staticRoleURL(name string) string {
	return fmt.Sprintf("/v1/openstack/static-roles/%s", name)
}

func expectedStaticRoleData(cloudName string, aux *AuxiliaryData) map[string]interface{} {
	expectedMap := map[string]interface{}{
		"cloud":             cloudName,
		"ttl":               time.Hour / time.Second,
		"rotation_duration": time.Hour / time.Second,
		"project_id":        aux.ProjectID,
		"domain_id":         aux.DomainID,
		"project_name":      tools.RandomString("p", 5),
		"extensions":        map[string]interface{}{},
		"secret_type":       "password",
		"username":          "vault-test",
	}
	return expectedMap
}
