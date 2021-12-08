package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
)

func main() {
	meta := &api.PluginAPIClientMeta{}

	flags := meta.FlagSet()
	if err := flags.Parse(os.Args[1:]); err != nil {
		fatalErr(err)
	}

	tlsConfig := meta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: openstack.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		fatalErr(err)
	}
}

func fatalErr(err error) {
	hclog.New(&hclog.LoggerOptions{}).Error(
		"plugin shutting down",
		"error",
		err,
	)
	os.Exit(1)
}
