package openstack

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/hashicorp/vault/sdk/helper/base62"
	"github.com/hashicorp/vault/sdk/helper/template"
)

const (
	PasswordLength = 16

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

type PasswordGenerator interface {
	GeneratePasswordFromPolicy(ctx context.Context, policyName string) (password string, err error)
}

type Passwords struct {
	PolicyGenerator PasswordGenerator
	PolicyName      string
}

func (p Passwords) Generate(ctx context.Context) (string, error) {
	if p.PolicyName == "" {
		return base62.Random(PasswordLength)
	}
	if p.PolicyGenerator == nil {
		return "", fmt.Errorf("policy set, but no policy generator specified")
	}
	return p.PolicyGenerator.GeneratePasswordFromPolicy(ctx, p.PolicyName)
}
