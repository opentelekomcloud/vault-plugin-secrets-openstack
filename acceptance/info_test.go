//go:build acceptance
// +build acceptance

package acceptance

import (
	"fmt"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (p *PluginTest) TestInfo() {
	t := p.T()

	resp, err := p.vaultDo(http.MethodGet, "/v1/openstack/info", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data := getResponseData(t, resp)
	fields := []string{"build_date", "build_revision", "build_version", "project_docs", "project_name"}
	for _, f := range fields {
		assert.NotEmpty(t, data[f], fmt.Sprintf("field `%s` is empty", f))
	}
}
