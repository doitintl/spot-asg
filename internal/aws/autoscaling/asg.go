package autoscaling

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws/request"

	"github.com/doitintl/spotzero/internal/math"

	"github.com/doitintl/spotzero/internal/aws/sts"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

const (
	maxRecordsReturnedByAPI = 100
	maxAsgNamesPerDescribe  = 50
)

// define interface for used methods only (simplify testing)
type awsAutoScaling interface {
	DescribeTagsPagesWithContext(aws.Context, *autoscaling.DescribeTagsInput, func(*autoscaling.DescribeTagsOutput, bool) bool, ...request.Option) error
	DescribeAutoScalingGroupsPagesWithContext(aws.Context, *autoscaling.DescribeAutoScalingGroupsInput, func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool, ...request.Option) error
}

type asgService struct {
	svc awsAutoScaling
}

// Lister ASG Lister interface
type Lister interface {
	List(ctx context.Context, tags map[string]string) ([]*autoscaling.Group, error)
}

// NewLister create new ASG Lister
func NewLister(role sts.AssumeRoleInRegion) Lister {
	return &asgService{svc: autoscaling.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region))}
}

func (s *asgService) List(ctx context.Context, tags map[string]string) ([]*autoscaling.Group, error) {
	var asgs []*autoscaling.Group
	log.Printf("listing autoscaling groups matching tags: %v", tags)
	// asgNamesSet idiomatic Go way to implement set of strings
	asgNamesSet := make(map[string]bool)
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
		req := &autoscaling.DescribeTagsInput{
			Filters:    asFilters,
			MaxRecords: aws.Int64(maxRecordsReturnedByAPI),
		}

		err := s.svc.DescribeTagsPagesWithContext(ctx, req, func(p *autoscaling.DescribeTagsOutput, lastPage bool) bool {
			for _, t := range p.Tags {
				switch *t.ResourceType {
				case "auto-scaling-group":
					// add asg name to set of strings
					asgNamesSet[*t.ResourceId] = true
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

	if len(asgNamesSet) != 0 {
		// copy names to array
		i := 0
		asgNames := make([]*string, len(asgNamesSet))
		for k := range asgNamesSet {
			asgNames[i] = aws.String(k)
			i++
		}
		for i := 0; i < len(asgNamesSet); i += maxAsgNamesPerDescribe {
			batch := asgNames[i:math.MinInt(i+maxAsgNamesPerDescribe, len(asgNames))]
			req := &autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: batch,
				MaxRecords:            aws.Int64(maxAsgNamesPerDescribe),
			}
			err := s.svc.DescribeAutoScalingGroupsPagesWithContext(ctx, req, func(p *autoscaling.DescribeAutoScalingGroupsOutput, lastPage bool) bool {
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
