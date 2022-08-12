//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) TestStaticCredsLifecycle() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)

	_, aux := openstackClient(t)

	type testCase struct {
		Cloud      string
		ProjectID  string
		DomainID   string
		Root       bool
		SecretType string
		Extensions map[string]interface{}
		Username   string
	}

	cases := map[string]testCase{
		"user_password": {
			Root:       false,
			Cloud:      cloud.Name,
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Username:   "static-test",
			SecretType: "password",
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			data := data

			_, err := p.vaultDo(
				http.MethodPost,
				pluginPwdPolicyEndpoint,
				map[string]interface{}{
					"policy": pwdPolicy,
				},
			)
			require.NoError(t, err)

			roleName := openstack.RandomString(openstack.NameDefaultSet, 4)

			resp, err := p.vaultDo(
				http.MethodPost,
				cloudURL(cloudName),
				cloudToCloudMap(cloud),
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))

			resp, err = p.vaultDo(
				http.MethodPost,
				staticRoleURL(roleName),
				cloudToStaticRoleMap(data.Root, data.Cloud, data.ProjectID, data.DomainID, data.Username, data.SecretType, data.Extensions),
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))

			resp, err = p.vaultDo(
				http.MethodGet,
				staticRoleURL(roleName),
				nil,
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))
			resp, err = p.vaultDo(
				http.MethodGet,
				staticCredsURL(roleName),
				nil,
			)
			logical.ErrorResponse("response of our err", err)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))

			//resp, err = p.vaultDo(
			//	http.MethodPost,
			//	"/v1/sys/leases/revoke-force/openstack/creds",
			//	nil,
			//)
			//require.NoError(t, err)
			//assertStatusCode(t, http.StatusNoContent, resp)
			//
			//resp, err = p.vaultDo(
			//	http.MethodDelete,
			//	roleURL(roleName),
			//	nil,
			//)
			//require.NoError(t, err)
			//assertStatusCode(t, http.StatusNoContent, resp)
			//
			//resp, err = p.vaultDo(
			//	http.MethodDelete,
			//	cloudURL(cloudName),
			//	nil,
			//)
			//require.NoError(t, err)
			//assertStatusCode(t, http.StatusNoContent, resp)
		})
	}
}

func staticCredsURL(roleName string) string {
	return fmt.Sprintf("/v1/openstack/static-creds/%s", roleName)
}

func staticRoleURL(name string) string {
	return fmt.Sprintf("/v1/openstack/static-role/%s", name)
}

func cloudToStaticRoleMap(root bool, cloud, projectID, domainID, username string, secretType string, extensions map[string]interface{}) map[string]interface{} {
	return fixtures.SanitizedMap(map[string]interface{}{
		"cloud":       cloud,
		"project_id":  projectID,
		"domain_id":   domainID,
		"root":        root,
		"secret_type": secretType,
		"extensions":  extensions,
		"username":    username,
	})
}
