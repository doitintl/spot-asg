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

const (
	OnDemandMarketType = "on-demand"
	SpotMarketType     = "spot"
)

// define interface for used methods only (simplify testing)
type launchTemplateVersionDescriber interface {
	DescribeLaunchTemplateVersionsWithContext(aws.Context, *ec2.DescribeLaunchTemplateVersionsInput, ...request.Option) (*ec2.DescribeLaunchTemplateVersionsOutput, error)
}

type ltDescriberService struct {
	svc launchTemplateVersionDescriber
}

type InstanceDetails struct {
	TypeName   string
	MarketType string
}

// InstanceDescriber contains methods for extracting and inspecting instance types
type InstanceDescriber interface {
	GetInstanceDetails(ctx context.Context, ltSpec *autoscaling.LaunchTemplateSpecification) (*InstanceDetails, error)
}

// NewInstanceDescriber create new InstanceDescriber
func NewInstanceDescriber(role sts.AssumeRoleInRegion) InstanceDescriber {
	return &ltDescriberService{
		svc: ec2.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region)),
	}
}

// GetInstanceDetails extract EC2 instance details from the provided LaunchTemplate.
// It returns EC2 instance details: type name, market type
func (s *ltDescriberService) GetInstanceDetails(ctx context.Context, ltSpec *autoscaling.LaunchTemplateSpecification) (*InstanceDetails, error) {
	input := &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: ltSpec.LaunchTemplateId,
		Versions:         []*string{ltSpec.Version},
	}
	output, err := s.svc.DescribeLaunchTemplateVersionsWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error describing launch template version: %v", err)
	}
	if output.LaunchTemplateVersions == nil || len(output.LaunchTemplateVersions) != 1 {
		return nil, errors.New("expected to get a single launch template version")
	}
	if output.LaunchTemplateVersions[0].LaunchTemplateData == nil {
		return nil, errors.New("expected to get non-empty launch template data")
	}
	marketType := OnDemandMarketType
	if output.LaunchTemplateVersions[0].LaunchTemplateData.InstanceMarketOptions != nil &&
		output.LaunchTemplateVersions[0].LaunchTemplateData.InstanceMarketOptions.MarketType != nil {
		marketType = *output.LaunchTemplateVersions[0].LaunchTemplateData.InstanceMarketOptions.MarketType
	}
	instanceType := InstanceDetails{
		*output.LaunchTemplateVersions[0].LaunchTemplateData.InstanceType,
		marketType,
	}
	return &instanceType, nil
}
