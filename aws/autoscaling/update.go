package autoscaling

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/doitintl/spotzero/aws/ec2"
	"github.com/doitintl/spotzero/aws/sts"
)

const (
	spotAllocationStrategy = "capacity-optimized"
	maxAsgTypes            = 20
	// spotzero updated tags
	spotzeroUpdatedTag     = "spotzero:updated"
	spotzeroUpdatedTimeTag = "spotzero:updated:time"
)

type awsAsgUpdater interface {
	CreateOrUpdateTagsWithContext(aws.Context, *autoscaling.CreateOrUpdateTagsInput, ...request.Option) (*autoscaling.CreateOrUpdateTagsOutput, error)
	UpdateAutoScalingGroupWithContext(aws.Context, *autoscaling.UpdateAutoScalingGroupInput, ...request.Option) (*autoscaling.UpdateAutoScalingGroupOutput, error)
}

type asgUpdaterService struct {
	asgsvc awsAsgUpdater
	ec2svc ec2.InstanceTypeDescriber
	config Config
}

// Updater interface contains methods for updating EC2 Auto Scaling groups
type Updater interface {
	CreateUpdateInput(context.Context, *autoscaling.Group) (*autoscaling.UpdateAutoScalingGroupInput, error)
	Update(context.Context, *autoscaling.Group) error
}

// A Config is used for update configuration tuning
type Config struct {
	// SimilarityConfig configures EC2 similarity matching algorithm.
	SimilarityConfig ec2.Config
	// OnDemandBaseCapacity the minimum amount of the Auto Scaling group's capacity that must be fulfilled by On-Demand Instances.
	// This base portion is provisioned first as your group scales. Defaults to 0 if not specified.
	// Set the value of OnDemandBaseCapacity in terms of the number of capacity units (VCPU), and not the number of instances.
	OnDemandBaseCapacity int64
	// OnDemandPercentageAboveBaseCapacity controls the percentages of On-Demand Instances and Spot Instances for your additional capacity
	// beyond OnDemandBaseCapacity. Expressed as a number (for example, 20 specifies 20% On-Demand Instances, 80% Spot Instances).
	// Defaults to 100 if not specified. If set to 100, only On-Demand Instances are provisioned.
	OnDemandPercentageAboveBaseCapacity int64
}

// NewUpdater create new Updater
func NewUpdater(role sts.AssumeRoleInRegion, config Config) Updater {
	return &asgUpdaterService{
		asgsvc: autoscaling.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region)),
		ec2svc: ec2.NewInstanceTypeDescriber(role),
		config: config,
	}
}

// CreateUpdateInput automatically creates a new MixedInstancePolicy for the provided EC2 Auto Scaling group.
// It returns a properly configured UpdateAutoScalingGroupInput request.
func (s *asgUpdaterService) CreateUpdateInput(ctx context.Context, group *autoscaling.Group) (*autoscaling.UpdateAutoScalingGroupInput, error) {
	// get overrides (types, weights) from asg
	overrides, err := s.createLaunchTemplateOverrides(ctx, group)
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

// Update automatically updates the provided EC2 Auto Scaling group with an automatically generated MixedInstancePolicy.
func (s *asgUpdaterService) Update(ctx context.Context, group *autoscaling.Group) error {
	if group == nil {
		return nil
	}
	log.Printf("updating the autoscaling group %v", *group.AutoScalingGroupARN)
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
	// update spotzero tags for the ASG
	return s.updateAutoScalingGroupTags(ctx, group)
}

func (s *asgUpdaterService) updateAutoScalingGroupTags(ctx context.Context, group *autoscaling.Group) error {
	log.Printf("updating tags for the autoscaling group %v", *group.AutoScalingGroupARN)
	input := &autoscaling.CreateOrUpdateTagsInput{
		Tags: []*autoscaling.Tag{
			{
				Key:               aws.String(spotzeroUpdatedTag),
				PropagateAtLaunch: aws.Bool(true),
				ResourceId:        group.AutoScalingGroupName,
				ResourceType:      aws.String("auto-scaling-group"),
				Value:             aws.String("true"),
			},
			{
				Key:               aws.String(spotzeroUpdatedTimeTag),
				PropagateAtLaunch: aws.Bool(true),
				ResourceId:        group.AutoScalingGroupName,
				ResourceType:      aws.String("auto-scaling-group"),
				Value:             aws.String(time.Now().String()),
			},
		},
	}
	output, err := s.asgsvc.CreateOrUpdateTagsWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("error updading tags for the autoscaling group: %v", err)
	}
	log.Printf("updated autoscaling group: %v", *output)
	return nil
}

func (s *asgUpdaterService) createLaunchTemplateOverrides(ctx context.Context, group *autoscaling.Group) ([]*autoscaling.LaunchTemplateOverrides, error) {
	instanceType := ""
	var err error
	if group.LaunchConfigurationName != nil {
		// TODO:get instance type from LaunchConfiguration getInstanceTypeFromLaunchConfiguration
	} else if group.LaunchTemplate != nil {
		// get LaunchTemplate from asg group
		instanceType, err = s.ec2svc.GetInstanceType(ctx, group.LaunchTemplate)
		if err != nil {
			return nil, fmt.Errorf("error getting instance type from launch template: %v", err)
		}
	} else if group.MixedInstancesPolicy != nil {
		// get LaunchTemplate from MixedInstancePolicy
		instanceType, err = s.ec2svc.GetInstanceType(ctx, group.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification)
		if err != nil {
			return nil, fmt.Errorf("error getting instance type from launch template: %v", err)
		}
	}
	if instanceType == "" {
		return nil, fmt.Errorf("failed to detect instance type for autoscaling group: %v", group.AutoScalingGroupARN)
	}
	// iterate over good candidates and add them with weights based on #vCPU
	candidates := ec2.GetSimilarTypes(instanceType, s.config.SimilarityConfig)
	ltOverrides := make([]*autoscaling.LaunchTemplateOverrides, len(candidates))
	for i, c := range candidates {
		// up to maximum number of instance types
		if i == maxAsgTypes {
			break
		}
		ltOverrides[i] = &autoscaling.LaunchTemplateOverrides{
			InstanceType:     aws.String(c.InstanceType),
			WeightedCapacity: aws.String(strconv.Itoa(c.Weight)),
		}
	}
	return ltOverrides, nil
}
