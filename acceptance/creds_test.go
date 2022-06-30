//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"github.com/gophercloud/gophercloud/acceptance/tools"
	"net/http"
	"testing"

	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	Cloud      string
	Username   string
	ProjectID  string
	DomainID   string
	Root       bool
	SecretType string
	UserRoles  []string
	Extensions map[string]interface{}
}

func (p *PluginTest) TestCredsLifecycle() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)

	_, aux := openstackClient(t)

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
			Extensions: map[string]interface{}{
				"identity_api_version": "3",
			},
		},
		"user_password": {
			Cloud:      cloud.Name,
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Root:       false,
			SecretType: "password",
			Extensions: map[string]interface{}{
				"object_store_endpoint_override": "https://swift.example.com",
			},
		},
		"static_user_token": {
			Cloud:      cloud.Name,
			Username:   tools.RandomString("vault-iam-", 3),
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Root:       false,
			SecretType: "token",
			UserRoles:  []string{"member"},
			Extensions: map[string]interface{}{
				"identity_api_version": "3",
			},
		},
		"static_user_password": {
			Cloud:      cloud.Name,
			Username:   tools.RandomString("vault-iam-", 3),
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Root:       false,
			SecretType: "password",
			Extensions: map[string]interface{}{
				"object_store_endpoint_override": "https://swift.example.com",
			},
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
				roleURL(roleName),
				cloudToRoleMap(data),
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))

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
				http.MethodPost,
				"/v1/sys/leases/revoke-force/openstack/creds",
				nil,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusNoContent, resp)

			resp, err = p.vaultDo(
				http.MethodDelete,
				roleURL(roleName),
				nil,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusNoContent, resp)

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
		"name":              cloud.Name,
		"auth_url":          cloud.AuthURL,
		"username":          cloud.Username,
		"password":          cloud.Password,
		"user_domain_name":  cloud.UserDomainName,
		"username_template": cloud.UsernameTemplate,
		"password_policy":   cloud.PasswordPolicy,
	}
}

func cloudToRoleMap(data testCase) map[string]interface{} {
	return fixtures.SanitizedMap(map[string]interface{}{
		"cloud":       data.Cloud,
		"username":    data.Username,
		"project_id":  data.ProjectID,
		"domain_id":   data.DomainID,
		"root":        data.Root,
		"secret_type": data.SecretType,
		"user_roles":  data.UserRoles,
		"extensions":  data.Extensions,
	})
}
