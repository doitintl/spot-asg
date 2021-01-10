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
	// refresh instances configuration
	minHealthyPercentage = 90  // 90%
	instanceWarmup       = 300 // 5 minutes
	// spotzero updated tags
	spotzeroUpdatedTag     = "spotzero:updated"
	spotzeroUpdatedTimeTag = "spotzero:updated:time"
)

type awsAsgUpdater interface {
	CreateOrUpdateTagsWithContext(aws.Context, *autoscaling.CreateOrUpdateTagsInput, ...request.Option) (*autoscaling.CreateOrUpdateTagsOutput, error)
	UpdateAutoScalingGroupWithContext(aws.Context, *autoscaling.UpdateAutoScalingGroupInput, ...request.Option) (*autoscaling.UpdateAutoScalingGroupOutput, error)
	StartInstanceRefreshWithContext(aws.Context, *autoscaling.StartInstanceRefreshInput, ...request.Option) (*autoscaling.StartInstanceRefreshOutput, error)
}

type asgUpdaterService struct {
	asgsvc awsAsgUpdater
	ec2svc ec2.InstanceDescriber
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
		ec2svc: ec2.NewInstanceDescriber(role),
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
	// get LT from group
	template, err := s.getLaunchTemplateSpec(group)
	if err != nil {
		return nil, fmt.Errorf("failed to get launch template: %v", err)
	}
	// prepare request
	mixedInstancePolicy := &autoscaling.MixedInstancesPolicy{
		InstancesDistribution: &autoscaling.InstancesDistribution{
			OnDemandBaseCapacity:                aws.Int64(s.config.OnDemandBaseCapacity),
			OnDemandPercentageAboveBaseCapacity: aws.Int64(s.config.OnDemandPercentageAboveBaseCapacity),
			SpotAllocationStrategy:              aws.String(spotAllocationStrategy),
		},
		LaunchTemplate: &autoscaling.LaunchTemplate{
			LaunchTemplateSpecification: &autoscaling.LaunchTemplateSpecification{
				LaunchTemplateId: template.LaunchTemplateId,
				Version:          template.Version,
			},
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
	err = s.updateAutoScalingGroupTags(ctx, group)
	if err != nil {
		return err
	}
	// refresh instances for the ASG
	return s.startInstanceRefresh(ctx, group)
}

func (s *asgUpdaterService) startInstanceRefresh(ctx context.Context, group *autoscaling.Group) error {
	log.Printf("starting instance refresh for the autoscaling group %v", *group.AutoScalingGroupARN)
	input := &autoscaling.StartInstanceRefreshInput{
		AutoScalingGroupName: group.AutoScalingGroupName,
		Preferences: &autoscaling.RefreshPreferences{
			InstanceWarmup:       aws.Int64(instanceWarmup),
			MinHealthyPercentage: aws.Int64(minHealthyPercentage),
		},
	}
	output, err := s.asgsvc.StartInstanceRefreshWithContext(ctx, input)
	if err != nil {
		return fmt.Errorf("error starting instance refresh for the autoscaling group: %v", err)
	}
	log.Printf("started instance refresh autoscaling group: %v", *output)
	return nil
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

// get LT spec from ASG or MixedInstancePolicy
func (s *asgUpdaterService) getLaunchTemplateSpec(group *autoscaling.Group) (*autoscaling.LaunchTemplateSpecification, error) {
	if group == nil {
		return nil, errors.New("error autoscaling group is nil")
	}
	if group.LaunchTemplate != nil {
		return group.LaunchTemplate, nil
	}
	if group.MixedInstancesPolicy != nil && group.MixedInstancesPolicy.LaunchTemplate != nil {
		return group.MixedInstancesPolicy.LaunchTemplate.LaunchTemplateSpecification, nil
	}
	return nil, fmt.Errorf("failed to find launch template attached to the autoscaling group: %v", group.AutoScalingGroupARN)
}

func (s *asgUpdaterService) createLaunchTemplateOverrides(ctx context.Context, group *autoscaling.Group) ([]*autoscaling.LaunchTemplateOverrides, error) {
	// get Launch Template from ASG
	lts, err := s.getLaunchTemplateSpec(group)
	if err != nil {
		return nil, fmt.Errorf("failed to get launch template: %v", err)
	}
	// get instance details from LaunchTemplate
	instance, err2 := s.ec2svc.GetInstanceDetails(ctx, lts)
	if err2 != nil {
		return nil, fmt.Errorf("failed to detect instance type for autoscaling group: %v", err2)
	}
	// check if LaunchTemplate is requesting Spot instances in configuration
	if instance.MarketType == ec2.SpotMarketType {
		return nil, errors.New("incompatible launch template: already requesting for spot instances")
	}
	// iterate over good candidates and add them with weights based on #vCPU
	candidates := ec2.GetSimilarTypes(instance.TypeName, s.config.SimilarityConfig)
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
