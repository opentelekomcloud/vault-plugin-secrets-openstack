package openstack

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/opentelekomcloud/vault-plugin-secrets-openstack/vars"
)

const infoPattern = "info"

var (
	pathInfo = &framework.Path{
		Pattern:      infoPattern,
		HelpSynopsis: "Get general plugin info",
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: pathInfoRead,
			},
		},
	}
)

func pathInfoRead(context.Context, *logical.Request, *framework.FieldData) (*logical.Response, error) {
	return &logical.Response{
		Data: map[string]interface{}{
			"project_name":   vars.ProjectName,
			"project_docs":   vars.ProjectDocs,
			"build_version":  vars.BuildVersion,
			"build_revision": vars.BuildRevision,
			"build_date":     vars.BuildDate,
		},
	}, nil
}
