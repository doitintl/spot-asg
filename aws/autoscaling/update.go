package autoscaling

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/doitintl/spotzero/aws/ec2"
	"github.com/doitintl/spotzero/aws/sts"
)

const (
	spotAllocationStrategy = "capacity-optimized"
)

type awsAsgUpdater interface {
	UpdateAutoScalingGroupWithContext(aws.Context, *autoscaling.UpdateAutoScalingGroupInput, ...request.Option) (*autoscaling.UpdateAutoScalingGroupOutput, error)
}

type asgUpdaterService struct {
	asgsvc awsAsgUpdater
	ec2svc ec2.InstanceTypeDescriber
	config Config
}

// Updater ASG Updater interface
type Updater interface {
	CreateUpdateInput(context.Context, *autoscaling.Group) (*autoscaling.UpdateAutoScalingGroupInput, error)
	Update(context.Context, *autoscaling.Group) error
}

type Config struct {
	SimilarityConfig                    ec2.Config
	OnDemandBaseCapacity                int64
	OnDemandPercentageAboveBaseCapacity int64
}

// NewUpdater create new ASG Updater
func NewUpdater(role sts.AssumeRoleInRegion, config Config) Updater {
	return &asgUpdaterService{
		asgsvc: autoscaling.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region)),
		ec2svc: ec2.NewInstanceTypeDescriber(role),
		config: config,
	}
}

func (s *asgUpdaterService) CreateUpdateInput(ctx context.Context, group *autoscaling.Group) (*autoscaling.UpdateAutoScalingGroupInput, error) {
	// get overrides (types, weights) from asg
	overrides, err := s.getLaunchTemplateOverrides(ctx, group)
	if err != nil {
		return nil, err
	}
	// prepare request
	mixedInstancePolicy := &autoscaling.MixedInstancesPolicy{
		InstancesDistribution: &autoscaling.InstancesDistribution{
			OnDemandBaseCapacity:                aws.Int64(s.config.OnDemandBaseCapacity),
			OnDemandPercentageAboveBaseCapacity: aws.Int64(s.config.OnDemandPercentageAboveBaseCapacity),
			SpotAllocationStrategy:              aws.String(spotAllocationStrategy),
		},
		LaunchTemplate: &autoscaling.LaunchTemplate{
			Overrides: overrides,
		},
	}
	return &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: group.AutoScalingGroupName,
		MixedInstancesPolicy: mixedInstancePolicy,
	}, nil
}

func (s *asgUpdaterService) Update(ctx context.Context, group *autoscaling.Group) error {
	if group == nil {
		return nil
	}
	log.Printf("updating autoscaling group %v", *group.AutoScalingGroupARN)
	// skip ASG with LaunchConfiguration
	if group.LaunchConfigurationName != nil {
		return errors.New("autoscaling group with launch configuration is not supported, skipping")
	}
	input, err := s.CreateUpdateInput(ctx, group)
	if err != nil {
		return fmt.Errorf("failed to create autoscaling group update input: %v", err)
	}
	output, err := s.asgsvc.UpdateAutoScalingGroupWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("error updading autoscaling group: %v", err)
	}
	log.Printf("updated autoscaling group: %v", *output)
	return nil
}

func (s *asgUpdaterService) getLaunchTemplateOverrides(ctx context.Context, group *autoscaling.Group) ([]*autoscaling.LaunchTemplateOverrides, error) {
	instanceType := ""
	if group.LaunchConfigurationName != nil {
		// TODO:get instance type from LaunchConfiguration getInstanceTypeFromLaunchConfiguration
	} else if group.LaunchTemplate != nil {
		// get LaunchTemplate from asg group
		itype, err := s.ec2svc.GetInstanceType(ctx, group.LaunchTemplate)
		if err != nil {
			return nil, fmt.Errorf("error getting instance type from launch template: %v", err)
		}
		instanceType = itype
	} else if group.MixedInstancesPolicy != nil {
		// get LaunchTemplate from MixedInstancePolicy
		itype, err := s.ec2svc.GetInstanceType(ctx, group.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification)
		if err != nil {
			return nil, fmt.Errorf("error getting instance type from launch template: %v", err)
		}
		instanceType = itype
	}
	if instanceType == "" {
		return nil, fmt.Errorf("failed to detect instance type for autoscaling group: %v", group.AutoScalingGroupARN)
	}
	// iterate over good candidates and add them with weights based on #vCPU
	candidates := ec2.GetSimilarTypes(instanceType, s.config.SimilarityConfig)
	ltOverrides := make([]*autoscaling.LaunchTemplateOverrides, len(candidates))
	for i, c := range candidates {
		ltOverrides[i] = &autoscaling.LaunchTemplateOverrides{
			InstanceType:     aws.String(c.InstanceType),
			WeightedCapacity: aws.String(strconv.Itoa(c.Weight)),
		}
	}
	return ltOverrides, nil
}
