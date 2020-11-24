// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package autoscaling

import (
	context "context"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	mock "github.com/stretchr/testify/mock"
)

// MockAsgLister is an autogenerated mock type for the AsgLister type
type MockAsgLister struct {
	mock.Mock
}

// ListGroups provides a mock function with given fields: ctx, tags
func (_m *MockAsgLister) ListGroups(ctx context.Context, tags map[string]string) ([]*autoscaling.Group, error) {
	ret := _m.Called(ctx, tags)

	var r0 []*autoscaling.Group
	if rf, ok := ret.Get(0).(func(context.Context, map[string]string) []*autoscaling.Group); ok {
		r0 = rf(ctx, tags)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*autoscaling.Group)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, map[string]string) error); ok {
		r1 = rf(ctx, tags)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}