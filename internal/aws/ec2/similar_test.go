package ec2

import (
	"reflect"
	"testing"
)

func Test_getGoodCandidates(t *testing.T) {
	type args struct {
		instanceType string
	}
	tests := []struct {
		name string
		args args
		want []InstanceTypeWeight
	}{
		{
			"get candidates for m5.4xlarge: general purpose 16vCPU",
			args{"m5.4xlarge"},
			[]InstanceTypeWeight{
				{"m5.4xlarge", 16},
				{"m4.4xlarge", 16},
				{"m5n.4xlarge", 16},
				{"m5ad.4xlarge", 16},
				{"m5dn.4xlarge", 16},
				{"m5d.4xlarge", 16},
				{"m5a.4xlarge", 16},
				{"m4.2xlarge", 8},
				{"m5n.2xlarge", 8},
				{"m5ad.2xlarge", 8},
				{"m5d.2xlarge", 8},
				{"m3.2xlarge", 8},
				{"m5.2xlarge", 8},
				{"m5dn.2xlarge", 8},
				{"m5zn.2xlarge", 8},
				{"m5a.2xlarge", 8},
				{"m5zn.3xlarge", 12},
				{"m5zn.6xlarge", 24},
				{"m5dn.8xlarge", 32},
				{"m5d.8xlarge", 32},
				{"m5a.8xlarge", 32},
				{"m5ad.8xlarge", 32},
				{"m5.8xlarge", 32},
				{"m5n.8xlarge", 32},
			},
		},
		{
			"get candidates for t3.large: burstable 2 vPCU",
			args{"t3.large"},
			[]InstanceTypeWeight{
				{"t3.large", 2},
				{"t3.medium", 2},
				{"t2.large", 2},
				{"t3a.large", 2},
				{"t3a.medium", 2},
				{"t3.nano", 2},
				{"t3a.nano", 2},
				{"t3a.small", 2},
				{"t3a.micro", 2},
				{"t3.small", 2},
				{"t3.micro", 2},
				{"t2.xlarge", 4},
				{"t3.xlarge", 4},
				{"t3a.xlarge", 4},
			},
		},
		{
			"get candidates for c5g.xlarge: graviron2 arm 4 vPCU",
			args{"c6g.xlarge"},
			[]InstanceTypeWeight{
				{"c6g.xlarge", 4},
				{"c6gn.xlarge", 4},
				{"c6gd.xlarge", 4},
				{"c6g.large", 2},
				{"c6gd.large", 2},
				{"c6gn.large", 2},
				{"c6gn.2xlarge", 8},
				{"c6g.2xlarge", 8},
				{"c6gd.2xlarge", 8},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSimilarTypes(tt.args.instanceType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getGoodCandidates() = %v, want %v", got, tt.want)
			}
		})
	}
}
