package sts

import (
	"context"

	"github.com/aws/aws-sdk-go/service/sts"
)

type stsService struct {
	svc *sts.STS
}

// RoleChecker interface
type RoleChecker interface {
	CheckRole(ctx context.Context) (string, error)
}

// AssumeRoleInRegion role to assume in region (with external ID)
type AssumeRoleInRegion struct {
	Arn        string `json:"role-arn"`
	ExternalID string `json:"external-id"`
	Region     string `json:"region"`
}

// NewRoleChecker create new RoleChecker
func NewRoleChecker(role AssumeRoleInRegion) RoleChecker {
	return &stsService{svc: sts.New(MustAwsSession(role.Arn, role.ExternalID, role.Region))}
}

func (s *stsService) CheckRole(ctx context.Context) (string, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := s.svc.GetCallerIdentityWithContext(ctx, input)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
