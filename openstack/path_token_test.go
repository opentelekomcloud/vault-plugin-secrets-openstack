package openstack

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testToken = `MIIF3wYJKoZIhvcNAQcCoIIF0DCCBcwNFnKcdf-YvrqbwBBb6LV4NN2TLr4uPvOmR+g==`
)

var (
	testTokenExp = time.Now().Add(time.Hour * 24).Format(time.RFC3339)
)

func TestTokenPath_read(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		t.Parallel()
		b, storage := testBackend(t)

		ts := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				t.Helper()

				body, _ := json.Marshal(map[string]interface{}{
					"token":      testToken,
					"expires_at": testTokenExp,
				})
				w.WriteHeader(http.StatusCreated)
				w.Write(body)
			}),
		)
		defer ts.Close()

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      pathConfig,
			Data: map[string]interface{}{
				"auth_url": ts.URL,
			},
		})
		assert.NoError(t, err)

		r, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      pathToken,
		})
		assert.NoError(t, err)

		assert.Equal(t, r.Data["expires_at"].(string), testTokenExp)
		assert.Equal(t, r.Data["token"].(string), testToken)
	})
}
