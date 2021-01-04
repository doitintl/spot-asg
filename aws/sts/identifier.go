// Package sts provides primitives to simplify STS AssumeRole functionality.
package sts

import (
	"context"

	"github.com/aws/aws-sdk-go/service/sts"
)

type stsService struct {
	svc *sts.STS
}

// Identifier interface
type Identifier interface {
	GetIdentity(ctx context.Context) (string, error)
}

// AssumeRoleInRegion role to assume in the region (with external ID)
type AssumeRoleInRegion struct {
	Arn        string `json:"role-arn"`
	ExternalID string `json:"external-id"`
	Region     string `json:"region"`
}

// NewIdentifier creates a new Identifier
func NewIdentifier(role AssumeRoleInRegion) Identifier {
	return &stsService{svc: sts.New(MustAwsSession(role.Arn, role.ExternalID, role.Region))}
}

// GetIdentity can be used to check assumed caller identity.
// It return the assumed identity name.
func (s *stsService) GetIdentity(ctx context.Context) (string, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := s.svc.GetCallerIdentityWithContext(ctx, input)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
