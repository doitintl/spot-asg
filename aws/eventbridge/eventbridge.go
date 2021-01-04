package eventbridge

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/doitintl/spotzero/internal/math"

	"github.com/doitintl/spotzero/aws/sts"

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

// Publisher interface
type Publisher interface {
	PublishEvents(ctx context.Context, events []interface{}, eventType string) error
}

// NewPublisher create new ASG Publisher to publish discovered ASG into EventBridge
func NewPublisher(role sts.AssumeRoleInRegion, eventBusArn string) Publisher {
	return &ebService{
		svc:         eventbridge.New(sts.MustAwsSession(role.Arn, role.ExternalID, role.Region)),
		eventBusArn: eventBusArn,
	}
}

// PublishEvents publish events (serializable JSON) into EventBridge event bus
func (s *ebService) PublishEvents(ctx context.Context, events []interface{}, eventType string) error {
	// publish ASG groups in batches
	for i := 0; i < len(events); i += maxRecordsPerPutEvents {
		batch := events[i:math.MinInt(i+maxRecordsPerPutEvents, len(events))]
		var entries []*eventbridge.PutEventsRequestEntry
		for _, events := range batch {
			jsonEvent, err := json.Marshal(events)
			if err != nil {
				return errors.Wrapf(err, "error converting autoscaling group to JSON")
			}
			entries = append(entries, &eventbridge.PutEventsRequestEntry{
				Time:         aws.Time(time.Now()),
				Source:       aws.String("spotzero"),
				EventBusName: aws.String(s.eventBusArn),
				Detail:       aws.String(string(jsonEvent)),
				DetailType:   aws.String(eventType),
			})
		}
		if len(entries) > 0 {
			req := &eventbridge.PutEventsInput{
				Entries: entries,
			}
			res, err := s.svc.PutEventsWithContext(ctx, req)
			if err != nil {
				return errors.Wrapf(err, "failed to send %v to event bus", eventType)
			}
			if res.FailedEntryCount != nil && *res.FailedEntryCount > 0 {
				return errors.Errorf("failed to send %v %v to event bus", *res.FailedEntryCount, eventType)
			}
		}
	}

	return nil
}
