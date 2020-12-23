package eventbridge

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/doitintl/spot-asg/mocks"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"

	"github.com/aws/aws-sdk-go/service/eventbridge"

	"github.com/stretchr/testify/mock"

	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func testGenerateAsgGroups(num int) []interface{} {
	asgs := make([]interface{}, num)
	for i := 0; i < num; i++ {
		name := fmt.Sprintf("test-asg-%v", i)
		arn := fmt.Sprintf("arn:aws:autoscaling:.../%v", name)
		asgs[i] = &autoscaling.Group{
			AutoScalingGroupARN:  &arn,
			AutoScalingGroupName: &name,
		}
	}
	return asgs
}

func Test_ebService_PublishAsgGroups(t *testing.T) {
	type fields struct {
		eventBusArn string
	}
	type args struct {
		ctx             context.Context
		groups          int
		calls           int
		filedEntryCount int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "get 1 asg",
			fields: fields{"eventbus:test:arn"},
			args: args{
				context.TODO(),
				1,
				1,
				0,
			},
		},
		{
			name:   "get 10 asg",
			fields: fields{"eventbus:test:arn"},
			args: args{
				context.TODO(),
				10,
				1,
				0,
			},
		},
		{
			name:   "get 91 asg",
			fields: fields{"eventbus:test:arn"},
			args: args{
				context.TODO(),
				91,
				10,
				0,
			},
		},
		{
			name:   "fail to post event",
			fields: fields{"eventbus:test:arn"},
			args: args{
				context.TODO(),
				1,
				1,
				0,
			},
			wantErr: true,
		},
		{
			name:   "fail to post some events",
			fields: fields{"eventbus:test:arn"},
			args: args{
				context.TODO(),
				2,
				1,
				1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEbSvc := new(mocks.AwsEventBridge)
			s := &ebService{
				svc:         mockEbSvc,
				eventBusArn: tt.fields.eventBusArn,
			}
			mockEbSvc.On("PutEventsWithContext",
				tt.args.ctx,
				mock.AnythingOfType("*eventbridge.PutEventsInput"),
			).Return(
				func(aws.Context, *eventbridge.PutEventsInput, ...request.Option) *eventbridge.PutEventsOutput {
					return &eventbridge.PutEventsOutput{
						FailedEntryCount: &tt.args.filedEntryCount,
					}
				},
				func(aws.Context, *eventbridge.PutEventsInput, ...request.Option) error {
					if tt.args.filedEntryCount == 0 && tt.wantErr {
						return errors.New("error")
					}
					return nil
				}).Times(tt.args.calls)
			asgs := testGenerateAsgGroups(tt.args.groups)
			if err := s.PublishEvents(tt.args.ctx, asgs); (err != nil) != tt.wantErr {
				t.Errorf("PublishEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
			// assert mock
			mockEbSvc.AssertExpectations(t)
		})
	}
}
