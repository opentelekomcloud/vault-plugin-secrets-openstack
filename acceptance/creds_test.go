//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) TestCredsLifecycle() {
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
		UserRoles  []string
	}

	cases := map[string]testCase{
		"root_token": {
			Cloud:     cloud.Name,
			ProjectID: aux.ProjectID,
			DomainID:  aux.DomainID,
			Root:      true,
		},
		"user_token": {
			Cloud:      cloud.Name,
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Root:       false,
			SecretType: "token",
			UserRoles:  []string{"member"},
		},
		"user_password": {
			Cloud:      cloud.Name,
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Root:       false,
			SecretType: "password",
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			data := data
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
				roleURL(roleName),
				cloudToRoleMap(data.Root, data.Cloud, data.ProjectID, data.DomainID, data.SecretType, data.UserRoles),
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))

			resp, err = p.vaultDo(
				http.MethodGet,
				roleURL(roleName),
				nil,
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

			resp, err = p.vaultDo(
				http.MethodDelete,
				roleURL(roleName),
				nil,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusOK, resp)

			resp, err = p.vaultDo(
				http.MethodDelete,
				cloudURL(cloudName),
				nil,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusNoContent, resp)
		})
	}
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

func cloudToRoleMap(root bool, cloud, projectID, domainID, secretType string, userRoles []string) map[string]interface{} {
	return fixtures.SanitizedMap(map[string]interface{}{
		"cloud":       cloud,
		"project_id":  projectID,
		"domain_id":   domainID,
		"root":        root,
		"secret_type": secretType,
		"user_roles":  userRoles,
	})
}
