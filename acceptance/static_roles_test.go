//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/acceptance/tools"
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
	Root             bool                   `json:"root"`
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

	cloud := &openstack.OsCloud{
		Name:             openstack.RandomString(openstack.NameDefaultSet, 10),
		AuthURL:          "https://example.com/v3",
		UserDomainName:   openstack.RandomString(openstack.NameDefaultSet, 10),
		Username:         openstack.RandomString(openstack.NameDefaultSet, 10),
		Password:         openstack.RandomString(openstack.PwdDefaultSet, 10),
		UsernameTemplate: "u-{{ .RoleName }}-{{ random 4 }}",
	}
	p.makeCloud(cloud)

	data := expectedStaticRoleData(cloud.Name)
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
			TTL:              data["ttl"].(time.Duration),
			RotationDuration: data["rotation_duration"].(time.Duration),
			ProjectID:        data["project_id"].(string),
			ProjectName:      data["project_name"].(string),
			Extensions:       data["extensions"].(map[string]interface{}),
			Root:             data["root"].(bool),
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
	return fmt.Sprintf("/v1/openstack/static-role/%s", name)
}

func expectedStaticRoleData(cloudName string) map[string]interface{} {
	expectedMap := map[string]interface{}{
		"cloud":             cloudName,
		"ttl":               time.Hour / time.Second,
		"rotation_duration": time.Hour / time.Second,
		"project_id":        "",
		"project_name":      tools.RandomString("p", 5),
		"extensions":        map[string]interface{}{},
		"root":              false,
		"secret_type":       "password",
		"username":          "static-test",
	}
	return expectedMap
}
