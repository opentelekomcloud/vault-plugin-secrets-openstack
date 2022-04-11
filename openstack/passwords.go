package openstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/helper/base62"
)

const (
	PasswordLength = 16
)

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
