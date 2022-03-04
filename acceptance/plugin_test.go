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

	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
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

	cloudBaseEndpoint = fmt.Sprintf("/v1/%s/cloud", pluginAlias)
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

func readJSONResponse(t *testing.T, r *http.Response) string {
	t.Helper()

	defer func() {
		t.Helper()
		require.NoError(t, r.Body.Close())
	}()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		require.NoError(t, err)
	}

	var data string
	if len(body) > 0 {
		dst := &bytes.Buffer{}
		if err := json.Indent(dst, body, "", "  "); err != nil {
			require.NoError(t, err)
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
	require.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))
}

func (p *PluginTest) unmountPlugin() {
	t := p.T()
	t.Helper()

	resp, err := p.vaultDo(http.MethodDelete, pluginMountEndpoint, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, readJSONResponse(t, resp))
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

type keyListData struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
}

func listResponseKeys(t *testing.T, r *http.Response) []string {
	t.Helper()

	raw := readJSONResponse(t, r)
	var out = keyListData{}
	require.NoError(t, jsonutil.DecodeJSON([]byte(raw), &out))
	return out.Data.Keys
}

func openstackCloudConfig(t *testing.T) *openstack.OsCloud {
	t.Helper()

	cloudConfig := os.Getenv("OS_CLIENT_CONFIG_FILE")
	cloudName := os.Getenv("OS_CLOUD")

	if cloudConfig == "" || cloudName == "" {
		t.Fatal("Both OS_CLIENT_CONFIG_FILE and OS_CLOUD needs to be set for the tests")
	}

	clientOpts := &clientconfig.ClientOpts{Cloud: cloudName}

	cloud, err := clientconfig.GetCloudFromYAML(clientOpts)
	require.NoError(t, err)

	return &openstack.OsCloud{
		Name:           cloudName,
		AuthURL:        cloud.AuthInfo.AuthURL,
		UserDomainName: getDomainName(cloud.AuthInfo),
		Username:       cloud.AuthInfo.Username,
		Password:       cloud.AuthInfo.Password,
	}
}

func getDomainName(authInfo *clientconfig.AuthInfo) string {
	for _, name := range []string{
		authInfo.UserDomainName,
		authInfo.DomainName,
		authInfo.ProjectDomainName,
	} {
		if name != "" {
			return name
		}
	}
	return ""
}

func cloudEndpoint(name string) string {
	return fmt.Sprintf("%s/%s", cloudBaseEndpoint, name)
}

func (p *PluginTest) prepareCloud() *openstack.OsCloud {
	t := p.T()
	t.Helper()

	cloud := openstackCloudConfig(t)
	r, err := p.vaultDo(
		http.MethodPost,
		cloudEndpoint(cloud.Name),
		map[string]interface{}{
			"auth_url":         cloud.AuthURL,
			"username":         cloud.Username,
			"password":         cloud.Password,
			"user_domain_name": cloud.UserDomainName,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)

	t.Cleanup(func() { // remove cloud after each test
		r, err := p.vaultDo(
			http.MethodDelete,
			cloudEndpoint(cloud.Name),
			nil,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, r.StatusCode)
	})

	return cloud
}
