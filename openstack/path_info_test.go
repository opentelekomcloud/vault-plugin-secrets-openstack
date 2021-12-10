package openstack

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/vars"
	"github.com/stretchr/testify/assert"
)

func TestInfoRead(t *testing.T) {
	t.Parallel()
	b, storage := testBackend(t)

	vars.ProjectName = tools.RandomString("proj-", 5)
	vars.ProjectDocs = tools.RandomString("docs-", 10)
	vars.BuildDate = tools.RandomString("date-", 5)
	vars.BuildRevision = tools.RandomString("rev-", 5)
	vars.BuildVersion = tools.RandomString("v_", 3)
	expected := map[string]interface{}{
		"project_name":   vars.ProjectName,
		"project_docs":   vars.ProjectDocs,
		"build_date":     vars.BuildDate,
		"build_revision": vars.BuildRevision,
		"build_version":  vars.BuildVersion,
	}

	res, err := b.HandleRequest(context.Background(), &logical.Request{
		Storage:   storage,
		Operation: logical.ReadOperation,
		Path:      infoPattern,
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, res.Data)
}
