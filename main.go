package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/plugin"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/openstack"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: openstack.Factory,
		Logger:             logger,
	}); err != nil {
		logger.Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
