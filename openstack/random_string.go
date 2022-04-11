package openstack

import (
	"crypto/rand"

	"github.com/hashicorp/vault/sdk/helper/template"
)

const (
	NameDefaultSet = `0123456789abcdefghijklmnopqrstuvwxyz`
	PwdDefaultSet  = `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz~!@#$%^&*()_+-={}[]:"'<>,./|\'?`
)

func RandomString(charset string, size int) string {
	var bytes = make([]byte, size)
	_, _ = rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes)
}

type usernameTemplateData struct {
	CloudName string
	RoleName  string
}

func RandomTemporaryUsername(templateString string, role *roleEntry) (string, error) {
	t, err := template.NewTemplate(template.Template(templateString))
	if err != nil {
		return "", err
	}
	data := usernameTemplateData{
		CloudName: role.Cloud,
		RoleName:  role.Name,
	}
	return t.Generate(data)
}
