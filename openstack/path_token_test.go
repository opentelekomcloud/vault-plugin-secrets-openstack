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
		b, storage := testBackend(t)

		ts := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				t.Helper()

				body, _ := json.Marshal(map[string]interface{}{
					"token": map[string]interface{}{
						"expires_at": testTokenExp,
						"catalog": []map[string]interface{}{
							{
								"endpoints": []map[string]interface{}{
									{
										"id":        "6872ffd84d4a84c29967739ae3fd",
										"interface": "public",
										"region":    "eu-de",
										"region_id": "eu-de",
										"url":       "https://example.com/v3",
									},
								},
								"id":   "45a4010411237f79b727c52b",
								"name": "keystone",
								"type": "identity",
							},
						},
					},
				})
				w.Header().Set("X-Subject-Token", testToken)
				w.WriteHeader(http.StatusCreated)
				w.Write(body)
			}),
		)
		defer ts.Close()

		configData := randomConfigData()
		configData["auth_url"] = ts.URL + "/v3"
		configData["region"] = "eu-de"

		_, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.CreateOperation,
			Path:      pathConfig,
			Data:      configData,
		})
		assert.NoError(t, err)

		r, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      pathToken,
		})
		assert.NoError(t, err)

		assert.NotNil(t, r)
		assert.Equal(t, r.Data["expires_at"].(string), testTokenExp)
		assert.Equal(t, r.Data["token"].(string), testToken)
	})
}
