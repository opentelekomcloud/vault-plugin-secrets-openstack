package openstack

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tokenRequest = `{
  "auth": {
    "identity": {
      "methods": [
        "password"
      ],
      "password": {
        "user": {
          "domain": {
            "name": "domain"
          },
          "name": "user",
          "password": "password"
        }
      }
    },
    "scope": {
      "project": {
        "domain": {
          "name": "domain"
        },
        "name": "project"
      }
    }
  }
}
`

	tokenResponse = `{
  "token": {
    "expires_at": "2014-10-02T13:45:00.000000Z"
  }
}`
)

func TestTokenPath_read(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		t.Parallel()
		b, storage := testBackend(t)

		th.SetupHTTP()
		defer th.TeardownHTTP()

		th.Mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, "POST")
			th.TestHeader(t, r, "Content-Type", "application/json")
			th.TestHeader(t, r, "Accept", "application/json")
			th.TestJSONRequest(t, r, tokenRequest)

			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, tokenResponse)
		})

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      pathConfig,
			Data: map[string]interface{}{
				"auth_url":     th.Endpoint() + "v3/",
				"password":     "password",
				"username":     "user",
				"domain_name":  "domain",
				"project_name": "project",
			},
		})
		require.NoError(t, err)

		r, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      pathToken,
		})
		require.NoError(t, err)

		assert.NotNil(t, r)
		assert.Equal(t, r.Data["expires_at"].(string), "2014-10-02 13:45:00 +0000 UTC")
	})
}
