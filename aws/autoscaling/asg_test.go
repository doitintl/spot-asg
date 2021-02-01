package autoscaling

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/doitintl/spotzero/mocks"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/stretchr/testify/mock"

	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func testNamedDescribeAutoScalingGroupsOutput(groupName string, desiredCap int64, instanceIds ...string) *autoscaling.DescribeAutoScalingGroupsOutput {
	var instances []*autoscaling.Instance
	for _, id := range instanceIds {
		instances = append(instances, &autoscaling.Instance{
			InstanceId:       aws.String(id),
			AvailabilityZone: aws.String("us-east-1a"),
		})
	}
	return &autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: []*autoscaling.Group{
			{
				AutoScalingGroupName: aws.String(groupName),
				DesiredCapacity:      aws.Int64(desiredCap),
				MinSize:              aws.Int64(1),
				MaxSize:              aws.Int64(5),
				Instances:            instances,
				AvailabilityZones:    aws.StringSlice([]string{"us-east-1a"}),
			},
		},
	}
}

func Test_asgService_ListGroups(t *testing.T) {
	type args struct {
		ctx  context.Context
		tags map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    []*autoscaling.Group
		wantErr bool
	}{
		{
			name: "list asg groups",
			args: args{context.TODO(), map[string]string{"test-tag": "test-value"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAsgSvc := new(mocks.AwsAutoScaling)
			s := &asgService{
				svc: mockAsgSvc,
			}
			mockAsgSvc.On("DescribeTagsPagesWithContext",
				tt.args.ctx,
				&autoscaling.DescribeTagsInput{
					Filters: []*autoscaling.Filter{
						{Name: aws.String("key"), Values: aws.StringSlice([]string{"test-tag"})},
						{Name: aws.String("value"), Values: aws.StringSlice([]string{"test-value"})},
					},
					MaxRecords: aws.Int64(maxRecordsReturnedByAPI),
				},
				mock.AnythingOfType("func(*autoscaling.DescribeTagsOutput, bool) bool"),
			).Run(func(args mock.Arguments) {
				fn := args.Get(2).(func(*autoscaling.DescribeTagsOutput, bool) bool)
				fn(&autoscaling.DescribeTagsOutput{
					Tags: []*autoscaling.TagDescription{
						{ResourceId: aws.String("auto-asg"), ResourceType: aws.String("auto-scaling-group")},
					}}, false)
			}).Return(nil).Once()
			mockAsgSvc.On("DescribeAutoScalingGroupsPagesWithContext",
				tt.args.ctx,
				&autoscaling.DescribeAutoScalingGroupsInput{
					AutoScalingGroupNames: aws.StringSlice([]string{"auto-asg"}),
					MaxRecords:            aws.Int64(maxAsgNamesPerDescribe),
				},
				mock.AnythingOfType("func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool"),
			).Run(func(args mock.Arguments) {
				fn := args.Get(2).(func(*autoscaling.DescribeAutoScalingGroupsOutput, bool) bool)
				fn(testNamedDescribeAutoScalingGroupsOutput("auto-asg", 1, "test-instance-id"), false)
			}).Return(nil)

			got, err := s.List(tt.args.ctx, tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List() got = %v, want %v", got, tt.want)
			}
			// assert mock
			mockAsgSvc.AssertExpectations(t)
		})
	}
}

// generate N `*autoscaling.TagDescription`
func createNTagDescriptions(n int) []*autoscaling.TagDescription {
	tags := make([]*autoscaling.TagDescription, n)
	for i := range tags {
		tags[i] = &autoscaling.TagDescription{
			Key:   aws.String(fmt.Sprint("key:", i)),
			Value: aws.String(fmt.Sprint("value:", i)),
		}
	}
	return tags
}

func Test_matchesAsgTags(t *testing.T) {
	type args struct {
		tags   map[string]string
		actual []*autoscaling.TagDescription
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"test exact match",
			args{
				map[string]string{"key:1": "value:1", "key:3": "value:3"},
				createNTagDescriptions(5),
			},
			true,
		},
		{
			"empty tags should match",
			args{
				map[string]string{},
				createNTagDescriptions(5),
			},
			true,
		},
		{
			"test exact match should fail",
			args{
				map[string]string{"key:1": "value:1", "key:X": "value:X"},
				createNTagDescriptions(5),
			},
			false,
		},
		{
			fmt.Sprint(spotzeroUpdatedTag, "=true should fail"),
			args{
				map[string]string{spotzeroUpdatedTag: "true"},
				createNTagDescriptions(5),
			},
			false,
		},
		{
			"asg without tags should fail",
			args{
				map[string]string{"key:1": "value:1", "key:3": "value:3"},
				createNTagDescriptions(0),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesAsgTags(tt.args.tags, tt.args.actual); got != tt.want {
				t.Errorf("matchesAsgTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
