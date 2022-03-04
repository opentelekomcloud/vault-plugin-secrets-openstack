//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roleData struct {
	Cloud       string                 `json:"cloud"`
	Ttl         time.Duration          `json:"ttl"`
	ProjectID   string                 `json:"project_id"`
	ProjectName string                 `json:"project_name"`
	Extensions  map[string]interface{} `json:"extensions"`
	Root        bool                   `json:"root"`
	SecretType  string                 `json:"secret_type"`
	UserGroups  []interface{}          `json:"user_groups"`
}

func extractRoleData(t *testing.T, resp *http.Response) *roleData {
	t.Helper()

	raw := readJSONResponse(t, resp)
	var out struct {
		Data *roleData `json:"data"`
	}
	require.NoError(t, jsonutil.DecodeJSON([]byte(raw), &out))
	return out.Data
}

func (p *PluginTest) TestRoleLifecycle() {
	t := p.T()

	data := expectedRoleData()
	roleName := "test-write"

	t.Run("WriteRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodPost,
			roleURL(roleName),
			data,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))
	})

	t.Run("ReadRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodGet,
			roleURL(roleName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		expected := &roleData{
			Cloud:       data["cloud"].(string),
			Ttl:         data["ttl"].(time.Duration),
			ProjectID:   data["project_id"].(string),
			ProjectName: data["project_name"].(string),
			Extensions:  data["extensions"].(map[string]interface{}),
			Root:        data["root"].(bool),
			SecretType:  data["secret_type"].(string),
			UserGroups:  data["user_groups"].([]interface{}),
		}
		assert.Equal(t, expected, extractRoleData(t, resp))
	})

	t.Run("ListRoles", func(t *testing.T) {
		resp, err := p.vaultDo("LIST", "/v1/openstack/roles", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		names := listResponseKeys(t, resp)
		require.Len(t, names, 1)
		assert.Equal(t, roleName, names[0])
	})

	t.Run("DeleteRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodDelete,
			roleURL(roleName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func roleURL(name string) string {
	return fmt.Sprintf("/v1/openstack/role/%s", name)
}

func expectedRoleData() map[string]interface{} {
	expectedMap := map[string]interface{}{
		"cloud":        tools.RandomString("cl", 5),
		"ttl":          time.Hour / time.Second,
		"project_id":   "",
		"project_name": tools.RandomString("p", 5),
		"extensions":   map[string]interface{}{},
		"root":         false,
		"secret_type":  "token",
		"user_groups":  []interface{}{},
	}
	return expectedMap
}
