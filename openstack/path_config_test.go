package openstack

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/acceptance/tools"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPath_read(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		b, storage := testBackend(t)

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      pathConfig,
		})
		assert.NoError(t, err)
		assert.EqualValues(t, 0, len(resp.Data))
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		expected := randomConfig()

		b, storage := testBackend(t)
		assert.NoError(t, b.setConfig(context.Background(), expected, storage))

		resp, err := b.HandleRequest(context.Background(), &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      pathConfig,
		})
		assert.NoError(t, err)

		jsonExpected, _ := jsonutil.EncodeJSON(expected)
		jsonActual, _ := jsonutil.EncodeJSON(resp.Data)
		assert.JSONEq(t, string(jsonExpected), string(jsonActual))
	})
}

func randomConfig() *osConfig {
	return &osConfig{
		AuthURL:     tools.RandomString("url-", 5),
		Username:    tools.RandomString("username-", 5),
		Password:    tools.RandomString("password-", 5),
		ProjectName: tools.RandomString("project-", 5),
		DomainName:  tools.RandomString("domain-", 5),
		Region:      tools.RandomString("region-", 5),
	}
}

func randomConfigData() map[string]interface{} {
	return map[string]interface{}{
		"auth_url":     tools.RandomString("url-", 5),
		"region":       tools.RandomString("region-", 5),
		"username":     tools.RandomString("username-", 5),
		"domain_name":  tools.RandomString("domain-", 5),
		"project_name": tools.RandomString("project-", 5),
		"password":     tools.RandomString("pwd-", 5),
	}
}

func TestPathConfig_write(t *testing.T) {
	cases := map[string]struct {
		operation logical.Operation
		initial   *osConfig
		data      map[string]interface{}
		err       error
	}{
		"create/empty_init": {
			logical.CreateOperation,
			nil,
			randomConfigData(),
			nil,
		},
		"create/empty_data": {
			logical.CreateOperation,
			randomConfig(),
			nil,
			nil,
		},
		"update/empty_init": {
			logical.UpdateOperation,
			nil,
			randomConfigData(),
			errEmptyConfigUpdate,
		},
		"update/empty_data": {
			logical.UpdateOperation,
			randomConfig(),
			nil,
			nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			b, storage := testBackend(t)
			if c.initial != nil {
				require.NoError(t, b.setConfig(context.Background(), c.initial, storage))
			}

			_, err := b.HandleRequest(context.Background(), &logical.Request{
				Storage:   storage,
				Operation: c.operation,
				Path:      pathConfig,
				Data:      c.data,
			})
			if !errors.Is(c.err, err) {
				t.Fatalf("expected error to be `%v`, but got `%v`", c.err, err)
			}
		})
	}
}
func TestPathConfig_errStorage(t *testing.T) {
	cases := []struct {
		err       error
		verb      failVerb
		operation logical.Operation
	}{
		{
			errReadingConfig,
			failVerbRead,
			logical.ReadOperation,
		},
		{
			errWritingConfig,
			failVerbPut,
			logical.CreateOperation,
		},
		{
			errDeleteConfig,
			failVerbDelete,
			logical.DeleteOperation,
		},
	}

	for _, v := range cases {
		t.Run(string(v.operation), func(t *testing.T) {
			b, storage := testBackend(t, v.verb)

			_, err := b.HandleRequest(context.Background(), &logical.Request{
				Storage:   storage,
				Operation: v.operation,
				Path:      pathConfig,
			})
			assert.Error(t, err)
			assert.ErrorIs(t, err, v.err)
		})
	}
}

func TestPathConfig_delete(t *testing.T) {
	b, storage := testBackend(t)
	_, err := b.HandleRequest(context.Background(), &logical.Request{
		Storage:   storage,
		Operation: logical.DeleteOperation,
		Path:      pathConfig,
	})
	assert.NoError(t, err)
}

func TestPathConfig_checkExists(t *testing.T) {
	for _, expected := range []bool{true, false} {
		name := fmt.Sprintf("exists/%v", expected)
		t.Run(name, func(t *testing.T) {
			b, storage := testBackend(t)
			if expected {
				require.NoError(t, b.setConfig(context.Background(), randomConfig(), storage))
			}
			checked, ok, err := b.HandleExistenceCheck(context.Background(), &logical.Request{
				Storage:   storage,
				Operation: logical.UpdateOperation,
				Path:      pathConfig,
			})
			assert.NoError(t, err)
			assert.Equal(t, true, checked)
			assert.Equal(t, expected, ok)
		})
	}
}
