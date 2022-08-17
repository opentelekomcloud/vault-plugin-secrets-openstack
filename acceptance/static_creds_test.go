//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type testStaticCase struct {
	Cloud      string
	ProjectID  string
	DomainID   string
	Root       bool
	SecretType string
	Username   string
	Extensions map[string]interface{}
}

func (p *PluginTest) TestStaticCredsLifecycle() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)

	client, aux := openstackClient(t)

	userRoleName := "member"

	dataCloud := map[string]interface{}{
		"auth_url":         cloud.AuthURL,
		"username":         cloud.Username,
		"password":         cloud.Password,
		"user_domain_name": cloud.UserDomainName,
	}

	cases := map[string]testStaticCase{
		"user_password": {
			Cloud:      cloud.Name,
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Username:   "static-test-1",
			SecretType: "password",
		},
		"user_token": {
			Cloud:      cloud.Name,
			ProjectID:  aux.ProjectID,
			DomainID:   aux.DomainID,
			Username:   "static-test-2",
			SecretType: "token",
			Extensions: map[string]interface{}{
				"identity_api_version": "3",
			},
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			data := data

			roleName := openstack.RandomString(openstack.NameDefaultSet, 4)

			resp, err := p.vaultDo(
				http.MethodPost,
				cloudURL(cloudName),
				dataCloud,
			)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))

			createUserOpts := users.CreateOpts{
				Name:             data.Username,
				Description:      "Static user",
				DefaultProjectID: aux.ProjectID,
				DomainID:         aux.DomainID,
				Password:         openstack.RandomString(openstack.PwdDefaultSet, 16),
			}
			user, err := users.Create(client, createUserOpts).Extract()
			require.NoError(t, err)

			rolesToAdd, err := filterRole(client, userRoleName)
			require.NoError(t, err)

			assignOpts := roles.AssignOpts{
				UserID:    user.ID,
				ProjectID: aux.ProjectID,
			}

			err = roles.Assign(client, rolesToAdd.ID, assignOpts).ExtractErr()
			require.NoError(t, err)

			resp, err = p.vaultDo(
				http.MethodPost,
				staticRoleURL(roleName),
				cloudToStaticRoleMap(data),
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
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode, readJSONResponse(t, resp))

			resp, err = p.vaultDo(
				http.MethodDelete,
				staticRoleURL(roleName),
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

			t.Cleanup(func() {
				require.NoError(t, users.Delete(client, user.ID).ExtractErr())
			})
		})
	}
}

func staticCredsURL(roleName string) string {
	return fmt.Sprintf("/v1/openstack/static-creds/%s", roleName)
}

func cloudToStaticRoleMap(data testStaticCase) map[string]interface{} {
	return fixtures.SanitizedMap(map[string]interface{}{
		"cloud":       data.Cloud,
		"project_id":  data.ProjectID,
		"domain_id":   data.DomainID,
		"secret_type": data.SecretType,
		"username":    data.Username,
		"extensions":  data.Extensions,
	})
}

func filterRole(client *gophercloud.ServiceClient, roleName string) (*roles.Role, error) {
	rolePages, err := roles.List(client, roles.ListOpts{Name: roleName}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("unable to query roles: %w", err)
	}

	roleList, err := roles.ExtractRoles(rolePages)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve roles: %w", err)
	}

	return &roleList[0], nil
}
