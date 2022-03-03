//go:build acceptance
// +build acceptance

package acceptance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	pluginBin   = "vault-plugin-secrets-openstack"
	pluginAlias = "openstack"
)

var (
	pluginCatalogEndpoint = fmt.Sprintf("/v1/sys/plugins/catalog/secret/%s", pluginAlias)
	pluginMountEndpoint   = fmt.Sprintf("/v1/sys/mounts/%s", pluginAlias)
)

type vaultCfg struct {
	Address string
	Token   string
}

type PluginTest struct {
	suite.Suite

	vaultConfig vaultCfg
}

func (p *PluginTest) SetupSuite() {
	p.vaultConfig = vaultCfg{
		Address: envOrDefault("VAULT_ADDR", "http://127.0.0.1:8200"),
		Token:   envOrDefault("VAULT_TOKEN", "root"),
	}
	// openstack configuration should be loaded automatically from env vars

	p.registerPlugin()

	p.unmountPlugin()
	p.mountPlugin()
}

func (p *PluginTest) TearDownSuite() {
	p.unregisterPlugin()
	p.unmountPlugin()
}

func readJSONResponse(r *http.Response) string {
	defer func() {
		_ = r.Body.Close()
	}()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	var data string
	if len(body) > 0 {
		dst := &bytes.Buffer{}
		if err := json.Indent(dst, body, "", "  "); err != nil {
			panic(err)
		}
		data = dst.String()
	}
	return data
}

func (p *PluginTest) registerPlugin() {
	t := p.T()
	t.Helper()

	pluginDir := os.Getenv("VAULT_PLUGIN_DIR")
	if pluginDir == "" {
		t.Fatal("plugin path is not set (set VAULT_PLUGIN_DIR)")
	}

	resp, err := p.vaultDo(http.MethodPut, pluginCatalogEndpoint, map[string]interface{}{
		"sha256":  fileSHA256(t, path.Join(pluginDir, pluginBin)),
		"command": pluginBin,
	})
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusNoContent)
}

func (p *PluginTest) unregisterPlugin() {
	t := p.T()
	t.Helper()

	resp, err := p.vaultDo(http.MethodDelete, pluginCatalogEndpoint, nil)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusNoContent)
}

func (p *PluginTest) mountPlugin() {
	t := p.T()
	t.Helper()

	resp, err := p.vaultDo(http.MethodPost, pluginMountEndpoint, map[string]interface{}{
		"type":        pluginBin,
		"description": "Test OpenStack plugin",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(resp))
}

func (p *PluginTest) unmountPlugin() {
	t := p.T()
	t.Helper()

	resp, err := p.vaultDo(http.MethodDelete, pluginMountEndpoint, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(resp))
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()

	hasher := sha256.New()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()
	_, err = io.Copy(hasher, f)
	require.NoError(t, err)
	sum := hex.EncodeToString(hasher.Sum(nil))
	return string(sum)
}

func TestPlugin(t *testing.T) {
	suite.Run(t, new(PluginTest))
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (p *PluginTest) vaultDo(method, endpoint string, body map[string]interface{}) (res *http.Response, err error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, p.vaultConfig.Address+endpoint, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", p.vaultConfig.Token)
	return http.DefaultClient.Do(req)
}

func listResponseKeys(t *testing.T, r *http.Response) []string {
	t.Helper()
	data := getResponseData(t, r)
	keys, ok := data["keys"].([]interface{})
	require.True(t, ok)
	keysStr := make([]string, len(keys))
	for i, v := range keys {
		keysStr[i] = v.(string)
	}
	return keysStr
}

func getResponseData(t *testing.T, r *http.Response) map[string]interface{} {
	t.Helper()
	res, err := jsonToMap(readJSONResponse(r))
	require.NoError(t, err)

	data, ok := res["data"].(map[string]interface{})
	require.True(t, ok)
	return data
}
