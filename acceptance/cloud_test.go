//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cloudData struct {
	AuthURL          string `json:"auth_url"`
	UserDomainName   string `json:"user_domain_name"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	UsernameTemplate string `json:"username_template"`
	PasswordPolicy   string `json:"password_policy"`
}

func extractCloudData(t *testing.T, resp *http.Response) *cloudData {
	t.Helper()

	raw := readJSONResponse(t, resp)
	var out struct {
		Data *cloudData `json:"data"`
	}
	require.NoError(t, jsonutil.DecodeJSON([]byte(raw), &out))
	return out.Data
}

func (p *PluginTest) TestCloudLifecycle() {
	t := p.T()

	data := map[string]interface{}{
		"auth_url":         "https://example.com/v3/",
		"username":         tools.RandomString("us", 4),
		"password":         tools.RandomString("", 15),
		"user_domain_name": "Default",
		"password_policy":  tools.RandomString("p", 5),
	}
	cloudName := "test-write"

	t.Run("WriteCloud", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodPost,
			cloudURL(cloudName),
			data,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))
	})

	t.Run("ReadCloud", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodGet,
			cloudURL(cloudName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		expected := &cloudData{
			AuthURL:          data["auth_url"].(string),
			UserDomainName:   data["user_domain_name"].(string),
			Username:         data["username"].(string),
			UsernameTemplate: openstack.DefaultUsernameTemplate,
			PasswordPolicy:   data["password_policy"].(string),
		}
		assert.Equal(t, expected, extractCloudData(t, resp))
	})

	t.Run("ListClouds", func(t *testing.T) {
		testListMethods(t, func(t *testing.T, m string) {
			resp, err := p.vaultDo(m, cloudsListURL, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			names := listResponseKeys(t, resp)
			require.Len(t, names, 1)
			assert.Equal(t, cloudName, names[0])
		})
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

var cloudsListURL = "/v1/openstack/clouds"

func cloudURL(name string) string {
	return fmt.Sprintf("/v1/openstack/clouds/%s", name)
}
