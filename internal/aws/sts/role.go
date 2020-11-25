package sts

import (
	"context"

	"github.com/aws/aws-sdk-go/service/sts"
)

type stsService struct {
	svc *sts.STS
}

//RoleChecker interface
type RoleChecker interface {
	CheckRole(ctx context.Context) (string, error)
}

//NewRoleChecker create new RoleChecker
func NewRoleChecker(roleArn, externalID, region string) RoleChecker {
	return &stsService{svc: sts.New(MustAwsSession(roleArn, externalID, region))}
}

func (s *stsService) CheckRole(ctx context.Context) (string, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := s.svc.GetCallerIdentityWithContext(ctx, input)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
