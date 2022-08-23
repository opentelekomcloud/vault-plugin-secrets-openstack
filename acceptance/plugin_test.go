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

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	pluginBin   = "vault-plugin-secrets-openstack"
	pluginAlias = "openstack"

	policyAlias = "openstack-policy"

	pwdPolicy = `
length=20

rule "charset" {
  charset = "abcdefghijklmnopqrstuvwxyz"
  min-chars = 1
}

rule "charset" {
  charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
  min-chars = 1
}

rule "charset" {
  charset = "0123456789"
  min-chars = 1
}

rule "charset" {
  charset = "!@#$%^&*"
  min-chars = 1
}`
)

var (
	pluginCatalogEndpoint   = fmt.Sprintf("/v1/sys/plugins/catalog/secret/%s", pluginAlias)
	pluginMountEndpoint     = fmt.Sprintf("/v1/sys/mounts/%s", pluginAlias)
	pluginPwdPolicyEndpoint = fmt.Sprintf("/v1/sys/policies/password/%s", policyAlias)

	cloudBaseEndpoint = fmt.Sprintf("/v1/%s/clouds", pluginAlias)
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
	requireStatusCode(t, http.StatusNoContent, resp)
}

func (p *PluginTest) unregisterPlugin() {
	t := p.T()
	t.Helper()

	resp, err := p.vaultDo(http.MethodDelete, pluginCatalogEndpoint, nil)
	require.NoError(t, err)
	requireStatusCode(t, http.StatusNoContent, resp)
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
	return sum
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

var (
	cloudConfig = os.Getenv("OS_CLIENT_CONFIG_FILE")
	cloudName   = os.Getenv("OS_CLOUD")
)

type AuxiliaryData struct {
	UserID      string
	DomainID    string
	ProjectID   string
	ProjectName string
}

func openstackClient(t *testing.T) (*gophercloud.ServiceClient, *AuxiliaryData) {
	t.Helper()
	opts := &clientconfig.ClientOpts{Cloud: cloudName}
	client, err := clientconfig.NewServiceClient("identity", opts)
	require.NoError(t, err)

	token := tokens.Get(client, client.Token())
	require.NoError(t, token.Err)
	project, err := token.ExtractProject()
	require.NoError(t, err)
	user, err := token.ExtractUser()
	require.NoError(t, err)

	aux := &AuxiliaryData{
		UserID:      user.ID,
		DomainID:    user.Domain.ID,
		ProjectID:   project.ID,
		ProjectName: project.Name,
	}
	return client, aux
}

func openstackCloudConfig(t *testing.T) *openstack.OsCloud {
	t.Helper()

	if cloudConfig == "" || cloudName == "" {
		t.Fatal("Both OS_CLIENT_CONFIG_FILE and OS_CLOUD needs to be set for the tests")
	}

	clientOpts := &clientconfig.ClientOpts{Cloud: cloudName}

	cloud, err := clientconfig.GetCloudFromYAML(clientOpts)
	require.NoError(t, err)

	return &openstack.OsCloud{
		Name:             cloudName,
		AuthURL:          cloud.AuthInfo.AuthURL,
		UserDomainName:   getDomainName(cloud.AuthInfo),
		Username:         cloud.AuthInfo.Username,
		Password:         cloud.AuthInfo.Password,
		UsernameTemplate: "vault-{{ .RoleName }}-{{ random 4 }}",
		PasswordPolicy:   "openstack-policy",
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

func assertStatusCode(t *testing.T, expected int, r *http.Response) bool {
	t.Helper()
	return assert.Equal(t, expected, r.StatusCode, readJSONResponse(t, r))
}

func requireStatusCode(t *testing.T, expected int, r *http.Response) {
	t.Helper()
	if assertStatusCode(t, expected, r) {
		return
	}
	t.FailNow()
}

func (p *PluginTest) makeCloud(cloud *openstack.OsCloud) {
	t := p.T()
	t.Helper()

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
	requireStatusCode(t, http.StatusNoContent, r)
	t.Logf("Cloud with name `%s` was created", cloud.Name)

	t.Cleanup(func() {
		r, err := p.vaultDo(
			http.MethodDelete,
			cloudEndpoint(cloud.Name),
			nil,
		)
		require.NoError(t, err)
		requireStatusCode(t, http.StatusNoContent, r)
		t.Logf("Cloud with name `%s` has been removed", cloud.Name)
	})
}

var listMethods = []string{"LIST", http.MethodGet}

func testListMethods(t *testing.T, f func(t *testing.T, m string)) {
	for _, method := range listMethods {
		t.Run(fmt.Sprintf("method-%s", method), func(t *testing.T) {
			method := method
			t.Parallel()
			f(t, method)
		})
	}
}
