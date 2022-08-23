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
	th.TestMethod(t, r, "POST")

	w.Header().Add("X-Subject-Token", client.TokenID)
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

	th.TestMethod(t, r, "GET")

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

func handleDeleteToken(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	th.TestMethod(t, r, "DELETE")

	w.WriteHeader(http.StatusNoContent)
}

func handleCreateUser(t *testing.T, w http.ResponseWriter, r *http.Request, userID string) {
	t.Helper()

	th.TestHeader(t, r, "Content-Type", "application/json")
	th.TestHeader(t, r, "Accept", "application/json")
	th.TestMethod(t, r, "POST")

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, `
{
    "user": {
        "default_project_id": "project",
        "description": "James Doe user",
        "domain_id": "domain",
        "email": "jdoe@example.com",
        "enabled": true,
        "id": "%[1]s",
        "links": {
            "self": "https://example.com/identity/v3/users/%[1]s"
        },
        "name": "James Doe",
        "password_expires_at": "2016-11-06T15:32:17.000000"
    }
}
`, userID)
}

func handleUpdateUser(t *testing.T, w http.ResponseWriter, r *http.Request, userID string) {
	t.Helper()

	th.TestHeader(t, r, "Content-Type", "application/json")
	th.TestHeader(t, r, "Accept", "application/json")
	th.TestMethod(t, r, "PATCH")

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `
{
    "user": {
        "default_project_id": "project",
        "description": "James Doe user",
        "domain_id": "domain",
        "email": "jdoe@example.com",
        "enabled": true,
        "id": "%s",
        "links": {
            "self": "https://example.com/identity/v3/users/29148f9awu90f1u2"
        },
        "name": "James Doe",
        "password_expires_at": "2016-11-06T15:32:17.000000"
    }
}
`, userID)
}

func handleGetUser(t *testing.T, w http.ResponseWriter, r *http.Request, userID string) {
	t.Helper()

	th.TestHeader(t, r, "Accept", "application/json")
	th.TestMethod(t, r, "GET")

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `
{
    "user": {
        "default_project_id": "project",
        "description": "James Doe user",
        "domain_id": "domain",
        "email": "jdoe@example.com",
        "enabled": true,
        "id": "%s",
        "links": {
            "self": "https://example.com/identity/v3/users/29148f9awu90f1u2"
        },
        "name": "James Doe",
        "password_expires_at": "2016-11-06T15:32:17.000000"
    }
}
`, userID)
}

func handleListUsers(t *testing.T, w http.ResponseWriter, r *http.Request, userID string, userName string) {
	t.Helper()

	th.TestHeader(t, r, "Accept", "application/json")
	th.TestMethod(t, r, "GET")

	w.Header().Add("Content-Type", "application/json")

	_, _ = fmt.Fprintf(w, `
{
  "users": [
    {
        "default_project_id": "project",
        "description": "James Doe user",
        "domain_id": "domain",
        "email": "jdoe@example.com",
        "enabled": true,
        "id": "%s",
        "links": {
            "self": "https://example.com/identity/v3/users/29148f9awu90f1u2"
        },
        "name": "%s",
        "password_expires_at": "2016-11-06T15:32:17.000000"
    }
  ],
  "links": {
    "next": null,
    "previous": null
  }
}
`, userID, userName)
}

func handleProjectList(t *testing.T, w http.ResponseWriter, r *http.Request, projectName string) {
	t.Helper()

	th.TestHeader(t, r, "Accept", "application/json")
	th.TestMethod(t, r, "GET")

	w.Header().Add("Content-Type", "application/json")

	_, _ = fmt.Fprintf(w, `
{
  "projects": [
    {
      "is_domain": false,
      "description": "The team that is red",
      "domain_id": "default",
      "enabled": true,
      "id": "1234",
      "name": "%[1]s"
    },
    {
      "is_domain": false,
      "description": "The team that is blue",
      "domain_id": "default",
      "enabled": true,
      "id": "9876",
      "name": "Blue Team"
    }
  ],
  "links": {
    "next": null,
    "previous": null
  }
}
`, projectName)
}

type EnabledMocks struct {
	TokenPost      bool
	TokenGet       bool
	TokenDelete    bool
	PasswordChange bool
	ProjectList    bool
	UserPost       bool
	UserPatch      bool
	UserList       bool
	UserDelete     bool
	UserGet        bool
}

func SetupKeystoneMock(t *testing.T, userID, projectName string, enabled EnabledMocks) {
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
		case "DELETE":
			if enabled.TokenDelete {
				handleDeleteToken(t, w, r)
			}
		default:
			w.WriteHeader(404)
		}
	})

	th.Mux.HandleFunc("/v3/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if enabled.UserPost {
				handleCreateUser(t, w, r, userID)
			}
		case "GET":
			if enabled.UserList {
				handleListUsers(t, w, r, userID, projectName)
			}
		default:
			w.WriteHeader(404)
		}
	})

	th.Mux.HandleFunc("/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			if enabled.ProjectList {
				handleProjectList(t, w, r, projectName)
			}
		default:
			w.WriteHeader(404)
		}
	})

	if enabled.PasswordChange {
		th.Mux.HandleFunc(fmt.Sprintf("/v3/users/%s/password", userID), func(w http.ResponseWriter, r *http.Request) {
			th.TestHeader(t, r, "Content-Type", "application/json")
			th.TestHeader(t, r, "Accept", "application/json")
			th.TestMethod(t, r, "POST")

			w.WriteHeader(http.StatusNoContent)
		})
	}

	if enabled.UserGet {
		th.Mux.HandleFunc(fmt.Sprintf("/v3/users/%s", userID), func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, "GET")

			handleGetUser(t, w, r, userID)
		})
	}

	if enabled.UserPatch {
		th.Mux.HandleFunc(fmt.Sprintf("/v3/users/%s", userID), func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, "PATCH")

			handleUpdateUser(t, w, r, userID)
		})
	}

	if enabled.UserDelete {
		th.Mux.HandleFunc(fmt.Sprintf("/v3/users/%s", userID), func(w http.ResponseWriter, r *http.Request) {
			th.TestHeader(t, r, "Accept", "application/json")
			th.TestMethod(t, r, "DELETE")

			w.WriteHeader(http.StatusNoContent)
		})
	}
}
