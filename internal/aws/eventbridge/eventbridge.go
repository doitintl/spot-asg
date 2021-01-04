package eventbridge

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/doitintl/spotzero/internal/math"

	"github.com/doitintl/spotzero/internal/aws/sts"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/eventbridge"
)

const (
	maxRecordsPerPutEvents = 10
)

type awsEventBridge interface {
	PutEventsWithContext(aws.Context, *eventbridge.PutEventsInput, ...request.Option) (*eventbridge.PutEventsOutput, error)
}

type ebService struct {
	svc         awsEventBridge
	eventBusArn string
}

// AsgPublisher interface
type AsgPublisher interface {
	PublishEvents(ctx context.Context, asgs []interface{}) error
}

// NewAsgPublisher create new ASG Publisher to publish discovered ASG into EventBridge
func NewAsgPublisher(role sts.AssumeRoleInRegion, eventBusArn string) AsgPublisher {
	return &ebService{
		svc:         eventbridge.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region)),
		eventBusArn: eventBusArn,
	}
}

// PublishEvents puslish events (serializable JSON) into eventbrige event bus
func (s *ebService) PublishEvents(ctx context.Context, asgs []interface{}) error {
	// publish ASG groups in batches
	for i := 0; i < len(asgs); i += maxRecordsPerPutEvents {
		batch := asgs[i:math.MinInt(i+maxRecordsPerPutEvents, len(asgs))]
		var entries []*eventbridge.PutEventsRequestEntry
		for _, asg := range batch {
			group, err := json.Marshal(asg)
			if err != nil {
				return errors.Wrapf(err, "error converting autoscaling group to JSON")
			}
			entries = append(entries, &eventbridge.PutEventsRequestEntry{
				Time:         aws.Time(time.Now()),
				Source:       aws.String("spotzero"),
				EventBusName: aws.String(s.eventBusArn),
				Detail:       aws.String(string(group)),
				DetailType:   aws.String("autoscaling-group"),
			})
		}
		if len(entries) > 0 {
			req := &eventbridge.PutEventsInput{
				Entries: entries,
			}
			res, err := s.svc.PutEventsWithContext(ctx, req)
			if err != nil {
				return errors.Wrap(err, "failed to send ASG to event bus")
			}
			if res.FailedEntryCount != nil && *res.FailedEntryCount > 0 {
				return errors.Errorf("failed to send %v ASG to event bus", *res.FailedEntryCount)
			}
		}
	}

	return nil
}
