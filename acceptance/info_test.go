//go:build acceptance
// +build acceptance

package acceptance

import (
	"net/http"
	"testing"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type infoData struct {
	BuildDate     string `json:"build_date"`
	BuildRevision string `json:"build_revision"`
	BuildVersion  string `json:"build_version"`
	ProjectDocs   string `json:"project_docs"`
	ProjectName   string `json:"project_name"`
}

func extractInfoData(t *testing.T, resp *http.Response) *infoData {
	t.Helper()

	raw := readJSONResponse(t, resp)
	var out struct {
		Data *infoData `json:"data"`
	}
	require.NoError(t, jsonutil.DecodeJSON([]byte(raw), &out))
	return out.Data
}

func (p *PluginTest) TestInfo() {
	t := p.T()

	resp, err := p.vaultDo(http.MethodGet, "/v1/openstack/info", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data := extractInfoData(t, resp)
	assert.NotEmpty(t, data)
}
