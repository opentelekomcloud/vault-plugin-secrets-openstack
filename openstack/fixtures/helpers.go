package fixtures

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
)

// isEmpty gets whether the specified object is considered empty or not.
func isEmpty(object interface{}) bool {

	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
		// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
		// for all other types, compare against the zero value
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

// SanitizedMap removes empty values from map
func SanitizedMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range in {
		if !isEmpty(v) {
			out[k] = v
		}
	}
	return out
}

func handleCreateToken(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	th.TestHeader(t, r, "Content-Type", "application/json")
	th.TestHeader(t, r, "Accept", "application/json")

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, `
{
  "token": {
    "expires_at": "2014-10-02T13:45:00.000000Z",
    "catalog": [
      {
        "endpoints": [
          {
            "id": "id",
            "interface": "public",
            "region": "RegionOne",
            "region_id": "RegionOne",
            "url": "%s"
          }
        ],
        "id": "idk",
        "name": "keystone",
        "type": "identity"
      }
    ]
  }
}
`, client.ServiceClient().Endpoint)
}

func handleGetToken(t *testing.T, w http.ResponseWriter, r *http.Request, userID string) {
	t.Helper()

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `
{
  "token": {
    "user": {
      "id": "%s"
    }
  }
}
`, userID)
}

type EnabledMocks struct {
	TokenPost      bool
	TokenGet       bool
	PasswordChange bool
}

func SetupKeystoneMock(t *testing.T, userID string, enabled EnabledMocks) {
	t.Helper()

	th.SetupHTTP()
	t.Cleanup(th.TeardownHTTP)

	th.Mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if enabled.TokenPost {
				handleCreateToken(t, w, r)
			}
		case "GET":
			if enabled.TokenGet {
				handleGetToken(t, w, r, userID)
			}
		default:
			w.WriteHeader(404)
		}
	})

	if enabled.PasswordChange {
		th.Mux.HandleFunc(fmt.Sprintf("/v3/users/%s/password", userID), func(w http.ResponseWriter, r *http.Request) {
			th.TestHeader(t, r, "Content-Type", "application/json")
			th.TestHeader(t, r, "Accept", "application/json")

			w.WriteHeader(http.StatusNoContent)
		})
	}
}
