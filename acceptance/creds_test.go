//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) TestCredsLifecycle() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)

	newCloud := p.makeChildCloud(cloud)

	_, aux := openstackClient(t)
	roleName := openstack.RandomString(openstack.NameDefaultSet, 4)

	t.Run("CredsRootToken", func(t *testing.T) {
		resp, err := p.vaultDo(
			http.MethodPost,
			cloudURL(cloudName),
			cloudToCloudMap(newCloud),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))

		resp, err = p.vaultDo(
			http.MethodPost,
			roleURL(roleName),
			cloudToRoleMap(newCloud, aux),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))

		resp, err = p.vaultDo(
			http.MethodGet,
			credsURL(roleName),
			nil,
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))
	})
}

func credsURL(roleName string) string {
	return fmt.Sprintf("/v1/openstack/creds/%s", roleName)
}

func cloudToCloudMap(cloud *openstack.OsCloud) map[string]interface{} {
	return map[string]interface{}{
		"name":             cloud.Name,
		"auth_url":         cloud.AuthURL,
		"username":         cloud.Username,
		"password":         cloud.Password,
		"user_domain_name": cloud.UserDomainName,
	}
}

func cloudToRoleMap(cloud *openstack.OsCloud, auxData *AuxiliaryData) map[string]interface{} {
	return map[string]interface{}{
		"cloud":       cloud.Name,
		"ttl":         time.Hour / time.Second,
		"project_id":  auxData.ProjectID,
		"root":        true,
		"secret_type": "token",
	}
}
