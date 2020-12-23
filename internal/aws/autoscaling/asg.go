package autoscaling

import (
	"context"
	"fmt"
	"log"

	"github.com/doitintl/spot-asg/internal/math"

	"github.com/doitintl/spot-asg/internal/aws/sts"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

const (
	maxRecordsReturnedByAPI = 100
	maxAsgNamesPerDescribe  = 50
)

// define interface for used methods only (simplify testing)
type awsAutoScaling interface {
	DescribeTagsPages(input *autoscaling.DescribeTagsInput, fn func(*autoscaling.DescribeTagsOutput, bool) bool) error
	DescribeAutoScalingGroupsPages(input *autoscaling.DescribeAutoScalingGroupsInput, fn func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool) error
}

type asgService struct {
	svc awsAutoScaling
}

// AsgLister ASG Lister interface
type AsgLister interface {
	ListGroups(ctx context.Context, tags map[string]string) ([]*autoscaling.Group, error)
}

// NewAsgLister create new ASG Lister
func NewAsgLister(role sts.AssumeRoleInRegion) AsgLister {
	return &asgService{svc: autoscaling.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region))}
}

func (s *asgService) ListGroups(ctx context.Context, tags map[string]string) ([]*autoscaling.Group, error) {
	var asgs []*autoscaling.Group
	log.Printf("listing autoscaling groups matching tags: %v", tags)
	var asgNames []*string
	{
		var asFilters []*autoscaling.Filter
		for k, v := range tags {
			// Not an exact match, but likely the best we can do
			asFilters = append(asFilters,
				&autoscaling.Filter{
					Name:   aws.String("key"),
					Values: []*string{aws.String(k)},
				},
				&autoscaling.Filter{
					Name:   aws.String("value"),
					Values: []*string{aws.String(v)},
				},
			)
		}
		request := &autoscaling.DescribeTagsInput{
			Filters:    asFilters,
			MaxRecords: aws.Int64(maxRecordsReturnedByAPI),
		}

		err := s.svc.DescribeTagsPages(request, func(p *autoscaling.DescribeTagsOutput, lastPage bool) bool {
			for _, t := range p.Tags {
				switch *t.ResourceType {
				case "auto-scaling-group":
					asgNames = append(asgNames, t.ResourceId)
				default:
					log.Printf("unexpected resource type: %v", *t.ResourceType)
				}
			}
			return true
		})
		if err != nil {
			return nil, fmt.Errorf("error listing autoscaling cluster tags: %v", err)
		}
	}

	if len(asgNames) != 0 {
		for i := 0; i < len(asgNames); i += maxAsgNamesPerDescribe {
			batch := asgNames[i:math.MinInt(i+maxAsgNamesPerDescribe, len(asgNames))]
			request := &autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: batch,
				MaxRecords:            aws.Int64(maxAsgNamesPerDescribe),
			}
			err := s.svc.DescribeAutoScalingGroupsPages(request, func(p *autoscaling.DescribeAutoScalingGroupsOutput, lastPage bool) bool {
				for _, asg := range p.AutoScalingGroups {
					if !matchesAsgTags(tags, asg.Tags) {
						// We used an inexact filter above
						continue
					}
					// Check for "Delete in progress" (the only use of .Status)
					if asg.Status != nil {
						log.Printf("skipping ASG %v (which matches tags): %v", *asg.AutoScalingGroupARN, *asg.Status)
						continue
					}
					asgs = append(asgs, asg)
				}
				return true
			})
			if err != nil {
				return nil, fmt.Errorf("error listing autoscaling groups: %v", err)
			}
		}
	}

	return asgs, nil
}

// matchesAsgTags is used to filter an asg by tags
func matchesAsgTags(tags map[string]string, actual []*autoscaling.TagDescription) bool {
	for k, v := range tags {
		found := false
		for _, a := range actual {
			if aws.StringValue(a.Key) == k {
				if aws.StringValue(a.Value) == v {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}
