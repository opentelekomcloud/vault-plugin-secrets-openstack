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
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type testStaticCase struct {
	cloud      string
	projectID  string
	domainID   string
	secretType string
	username   string
	extensions map[string]interface{}
}

func (p *PluginTest) TestStaticCredsLifecycle() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)

	client, aux := openstackClient(t)

	userRoleName := "member"
	allRoles := getAllRoles(t, client)

	dataCloud := map[string]interface{}{
		"auth_url":         cloud.AuthURL,
		"username":         cloud.Username,
		"password":         cloud.Password,
		"user_domain_name": cloud.UserDomainName,
	}

	cases := map[string]testStaticCase{
		"user_password": {
			cloud:      cloud.Name,
			projectID:  aux.ProjectID,
			domainID:   aux.DomainID,
			username:   "static-test-1",
			secretType: "password",
		},
		"user_token": {
			cloud:      cloud.Name,
			projectID:  aux.ProjectID,
			domainID:   aux.DomainID,
			username:   "static-test-2",
			secretType: "token",
			extensions: map[string]interface{}{
				"identity_api_version": "3",
			},
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			data := data

			roleName := openstack.RandomString(openstack.NameDefaultSet, 4)

			userId := userSetup(t, client, data, aux, allRoles[userRoleName].ID)
			t.Cleanup(func() {
				require.NoError(t, users.Delete(client, userId).ExtractErr())
			})

			resp, err := p.vaultDo(
				http.MethodPost,
				cloudURL(cloudName),
				dataCloud,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusNoContent, resp)

			resp, err = p.vaultDo(
				http.MethodPost,
				staticRoleURL(roleName),
				cloudToStaticRoleMap(data),
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusNoContent, resp)

			resp, err = p.vaultDo(
				http.MethodGet,
				staticRoleURL(roleName),
				nil,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusOK, resp)

			resp, err = p.vaultDo(
				http.MethodGet,
				staticCredsURL(roleName),
				nil,
			)
			require.NoError(t, err)
			assertStatusCode(t, http.StatusOK, resp)

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
		})
	}
}

func staticCredsURL(roleName string) string {
	return fmt.Sprintf("/v1/openstack/static-creds/%s", roleName)
}

func cloudToStaticRoleMap(data testStaticCase) map[string]interface{} {
	return fixtures.SanitizedMap(map[string]interface{}{
		"cloud":       data.cloud,
		"project_id":  data.projectID,
		"domain_id":   data.domainID,
		"secret_type": data.secretType,
		"username":    data.username,
		"extensions":  data.extensions,
	})
}

func getAllRoles(t *testing.T, client *gophercloud.ServiceClient) map[string]roles.Role {
	rolePages, err := roles.List(client, nil).AllPages()
	require.NoError(t, err)

	roleList, err := roles.ExtractRoles(rolePages)
	require.NoError(t, err)

	result := make(map[string]roles.Role, len(roleList))

	for _, role := range roleList {
		result[role.Name] = role
	}

	return result
}

func userSetup(t *testing.T, client *gophercloud.ServiceClient, data testStaticCase, aux *AuxiliaryData, roleID string) string {
	createUserOpts := users.CreateOpts{
		Name:             data.username,
		Description:      "Static user",
		DefaultProjectID: aux.ProjectID,
		DomainID:         aux.DomainID,
		Password:         openstack.RandomString(openstack.PwdDefaultSet, 16),
	}
	user, err := users.Create(client, createUserOpts).Extract()
	require.NoError(t, err)

	assignOpts := roles.AssignOpts{
		UserID:    user.ID,
		ProjectID: aux.ProjectID,
	}

	err = roles.Assign(client, roleID, assignOpts).ExtractErr()
	require.NoError(t, err)

	return user.ID
}
