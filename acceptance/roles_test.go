//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) TestRoleLifecycle() {
	t := p.T()

	roleMap := expectedRoleData()
	roleName := "test-write"

	t.Run("WriteRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodPost,
			roleURL(roleName),
			roleMap,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(resp))
	})

	t.Run("ReadRole", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodGet,
			roleURL(roleName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		json, err := jsonToMap(readJSONResponse(resp))
		require.NoError(t, err)
		assert.Equal(t, roleMap, json["data"])
	})

	t.Run("ListRoles", func(t *testing.T) {
		resp, err := p.vaultDo("LIST", roleListURL, nil)
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

var roleListURL = "/v1/openstack/roles"

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
