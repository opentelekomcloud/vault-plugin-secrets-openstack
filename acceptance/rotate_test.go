//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
	"net/http"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) makeChildCloud(base *openstack.OsCloud) *openstack.OsCloud {
	t := p.T()
	t.Helper()

	client, aux := openstackClient(t)

	createUserOpts := users.CreateOpts{
		Name:        openstack.RandomString(openstack.NameDefaultSet, 10),
		Description: "Temporary root user",
		DomainID:    aux.DomainID,
		Password:    openstack.RandomString(openstack.PwdDefaultSet, 16),
	}
	user, err := users.Create(client, createUserOpts).Extract()
	require.NoError(t, err)

	rolePages, err := roles.List(client, roles.ListOpts{}).AllPages()
	require.NoError(t, err)

	rolesToAssign, err := roles.ExtractRoles(rolePages)
	require.NoError(t, err)

	for _, role := range rolesToAssign {
		assignOpts := roles.AssignOpts{
			UserID:    user.ID,
			ProjectID: user.DefaultProjectID,
			DomainID:  user.DomainID,
		}
		require.NoError(t, roles.Assign(client, role.ID, assignOpts).ExtractErr())
	}

	t.Cleanup(func() {
		require.NoError(t, users.Delete(client, user.ID).ExtractErr())
	})

	newRoot := &openstack.OsCloud{
		Name:           openstack.RandomString(openstack.NameDefaultSet, 4),
		AuthURL:        base.AuthURL,
		UserDomainName: base.UserDomainName,
		Username:       createUserOpts.Name,
		Password:       createUserOpts.Password,
	}
	p.makeCloud(newRoot)
	return newRoot
}

func (p *PluginTest) TestRootRotate() {
	t := p.T()

	cloud := openstackCloudConfig(t)
	require.NotEmpty(t, cloud)
	p.makeCloud(cloud)

	// create temporary user, so base root won't be rotated
	newCloud := p.makeChildCloud(cloud)

	r, err := p.vaultDo(
		http.MethodGet,
		fmt.Sprintf("/v1/%s/rotate-root/%s", pluginAlias, newCloud.Name),
		nil,
	)
	require.NoError(t, err)
	assertStatusCode(t, http.StatusOK, r)
}
