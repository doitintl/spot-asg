package autoscaling

import (
	"context"
	"reflect"
	"testing"

	"github.com/doitintl/spot-asg/mocks"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/stretchr/testify/mock"

	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func testNamedDescribeAutoScalingGroupsOutput(groupName string, desiredCap int64, instanceIds ...string) *autoscaling.DescribeAutoScalingGroupsOutput {
	instances := []*autoscaling.Instance{}
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

			got, err := s.ListGroups(tt.args.ctx, tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListGroups() got = %v, want %v", got, tt.want)
			}
			// assert mock
			mockAsgSvc.AssertExpectations(t)
		})
	}
}
