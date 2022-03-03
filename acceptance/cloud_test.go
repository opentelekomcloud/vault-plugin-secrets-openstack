//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) TestCloudLifecycle() {
	t := p.T()

	cloudData := map[string]interface{}{
		"auth_url":         "https://example.com/v3/",
		"username":         "admin",
		"password":         "RcigTiYrJjVmEkrV71Cd",
		"user_domain_name": "Default",
	}
	cloudName := "test-write"

	t.Run("WriteCloud", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodPost,
			cloudURL(cloudName),
			cloudData,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(resp))
	})

	t.Run("ReadCloud", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodGet,
			cloudURL(cloudName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		json, err := jsonToMap(readJSONResponse(resp))
		require.NoError(t, err)
		assert.Equal(t, cloudData, json["data"])
	})

	t.Run("ListClouds", func(t *testing.T) {
		resp, err := p.vaultDo("LIST", cloudsListURL, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		names := listResponseKeys(t, resp)
		require.Len(t, names, 1)
		assert.Equal(t, cloudName, names[0])
	})

	t.Run("DeleteCloud", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodDelete,
			cloudURL(cloudName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func jsonToMap(src string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := jsonutil.DecodeJSON([]byte(src), &result)
	if err != nil {
		return nil, err
	}

	return result, err
}

var cloudsListURL = "/v1/openstack/clouds"

func cloudURL(name string) string {
	return fmt.Sprintf("/v1/openstack/cloud/%s", name)
}
