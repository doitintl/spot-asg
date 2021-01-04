// Package ec2 contains functions for inspecting EC2 instance types and searching for a similar instance types.
package ec2

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/doitintl/spotzero/aws/sts"
)

// define interface for used methods only (simplify testing)
type launchTemplateVersionDescriber interface {
	DescribeLaunchTemplateVersionsWithContext(aws.Context, *ec2.DescribeLaunchTemplateVersionsInput, ...request.Option) (*ec2.DescribeLaunchTemplateVersionsOutput, error)
}

type ltDescriberService struct {
	svc launchTemplateVersionDescriber
}

// InstanceTypeDescriber contains methods for extracting and inspecting instance types
type InstanceTypeDescriber interface {
	GetInstanceType(ctx context.Context, ltSpec *autoscaling.LaunchTemplateSpecification) (string, error)
}

// NewInstanceTypeDescriber create new InstanceTypeDescriber
func NewInstanceTypeDescriber(role sts.AssumeRoleInRegion) InstanceTypeDescriber {
	return &ltDescriberService{
		svc: ec2.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region)),
	}
}

// GetInstanceType extract EC2 instance type name from the provided LaunchTemplate.
// It returns EC2 instance type name.
func (s *ltDescriberService) GetInstanceType(ctx context.Context, ltSpec *autoscaling.LaunchTemplateSpecification) (string, error) {
	input := &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: ltSpec.LaunchTemplateId,
		Versions:         []*string{ltSpec.Version},
	}
	output, err := s.svc.DescribeLaunchTemplateVersionsWithContext(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error describing launch template version: %v", err)
	}
	if output.LaunchTemplateVersions == nil || len(output.LaunchTemplateVersions) != 1 {
		return "", errors.New("expected to get a single launch template version")
	}
	if output.LaunchTemplateVersions[0].LaunchTemplateData == nil {
		return "", errors.New("expected to get non-empty launch template data")
	}
	instanceType := output.LaunchTemplateVersions[0].LaunchTemplateData.InstanceType
	return *instanceType, nil
}
