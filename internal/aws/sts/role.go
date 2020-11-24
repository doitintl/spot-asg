package sts

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
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
func NewRoleChecker(roleARN, externalID, region string) RoleChecker {
	return &stsService{svc: newSTSClient(roleARN, externalID, region)}
}

func (s *stsService) CheckRole(ctx context.Context) (string, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := s.svc.GetCallerIdentityWithContext(ctx, input)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func newSTSClient(roleARN, externalID, region string) *sts.STS {
	// NewEC2Client constructs a new ec2 client with credentials and session
	sess := session.Must(session.NewSession())

	config := aws.NewConfig()

	if region != "" {
		config = config.WithRegion(region)
	}

	if (externalID != "") && (roleARN != "") {
		creds := stscreds.NewCredentials(sess, roleARN, func(p *stscreds.AssumeRoleProvider) {
			p.ExternalID = &externalID
		})

		config = config.WithCredentials(creds)
	}

	return sts.New(sess, config)
}
