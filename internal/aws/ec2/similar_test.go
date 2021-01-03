package ec2

import (
	"testing"
)

//nolint:funlen
func Test_GetSimilarTypes(t *testing.T) {
	type args struct {
		instanceType string
		config       SimilarityConfig
	}
	tests := []struct {
		name string
		args args
		want []InstanceTypeWeight
	}{
		{
			"get candidates for m5.4xlarge: general purpose 16vCPU",
			args{
				"m5.4xlarge",
				SimilarityConfig{
					IgnoreFamily:        false,
					IgnoreGeneration:    false,
					MultiplyFactorUpper: 2,
					MultiplyFactorLower: 2,
				},
			},
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
			"get candidates for m5.4xlarge: general purpose 16vCPU tuned config",
			args{
				"m5.4xlarge",
				SimilarityConfig{
					IgnoreFamily:        false,
					IgnoreGeneration:    false,
					MultiplyFactorUpper: 1,
					MultiplyFactorLower: 2,
				},
			},
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
				{"m5.2xlarge", 8},
				{"m5dn.2xlarge", 8},
				{"m5zn.2xlarge", 8},
				{"m5a.2xlarge", 8},
				{"m5zn.3xlarge", 12},
			},
		},
		{
			"get candidates for t3.large: burstable 2 vPCU",
			args{
				"t3.large",
				SimilarityConfig{
					IgnoreFamily:        false,
					IgnoreGeneration:    false,
					MultiplyFactorUpper: 2,
					MultiplyFactorLower: 2,
				},
			},
			[]InstanceTypeWeight{
				{"t3.large", 2},
				{"t3.medium", 2},
				{"t2.large", 2},
				{"t2.medium", 2},
				{"t3a.large", 2},
				{"t3a.medium", 2},
				{"t3.nano", 2},
				{"t3a.nano", 2},
				{"t3a.small", 2},
				{"t3a.micro", 2},
				{"t3.small", 2},
				{"t3.micro", 2},
				{"t2.micro", 1},
				{"t2.nano", 1},
				{"t2.small", 1},
				{"t2.xlarge", 4},
				{"t3.xlarge", 4},
				{"t3a.xlarge", 4},
			},
		},
		{
			"get candidates for c5g.xlarge: graviron2 arm 4 vPCU",
			args{
				"c6g.xlarge",
				SimilarityConfig{
					IgnoreFamily:        false,
					IgnoreGeneration:    false,
					MultiplyFactorUpper: 2,
					MultiplyFactorLower: 2,
				},
			},
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
			got := GetSimilarTypes(tt.args.instanceType, tt.args.config)
			if len(got) != len(tt.want) {
				t.Errorf("GetSimilarTypes() result size = %v, want %v", len(got), len(tt.want))
				return
			}
			// check sort
			for i := range got {
				if got[i].Weight != tt.want[i].Weight {
					t.Errorf("GetSimilarTypes() sorted weight = %v, want %v", got[i].Weight, tt.want[i].Weight)
				}
			}
		})
	}
}

func Test_isSimilarGPU(t *testing.T) {
	type args struct {
		oGPU int
		nGPU int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"no GPU",
			args{0, 0},
			true,
		},
		{
			"fail: no GPU",
			args{0, 1},
			false,
		},
		{
			"same number of GPU",
			args{2, 2},
			true,
		},
		{
			"smaller number of GPU",
			args{2, 4},
			true,
		},
		{
			"fail: bigger number of GPU",
			args{2, 1},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSimilarGPU(tt.args.oGPU, tt.args.nGPU); got != tt.want {
				t.Errorf("isSimilarGPU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSimilarCPU(t *testing.T) {
	type args struct {
		oCPU      int
		nCPU      int
		oArch     []string
		nArch     []string
		factorUp  int
		factorLow int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same everything",
			args: args{2, 2, []string{"x86_64"}, []string{"x86_64"}, 2, 2},
			want: true,
		},
		{
			name: "subset architecture",
			args: args{2, 2, []string{"x86_64"}, []string{"i386", "x86_64"}, 2, 2},
			want: true,
		},
		{
			name: "different architecture",
			args: args{2, 2, []string{"x86_64"}, []string{"arm64"}, 2, 2},
			want: false,
		},
		{
			name: "cpu in range",
			args: args{2, 4, []string{"arm64"}, []string{"arm64"}, 2, 2},
			want: true,
		},
		{
			name: "cpu out of range",
			args: args{4, 1, []string{"arm64"}, []string{"arm64"}, 2, 2},
			want: false,
		},
		{
			name: "cpu out upper range",
			args: args{4, 16, []string{"arm64"}, []string{"arm64"}, 4, 1},
			want: true,
		},
		{
			name: "cpu out lower range",
			args: args{4, 1, []string{"arm64"}, []string{"arm64"}, 1, 4},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSimilarCPU(tt.args.oCPU, tt.args.nCPU, tt.args.oArch, tt.args.nArch, tt.args.factorUp, tt.args.factorLow); got != tt.want {
				t.Errorf("isSimilarCPU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSimilarKind(t *testing.T) {
	type args struct {
		oFamily          string
		oType            string
		oGeneration      string
		nFamily          string
		nType            string
		nGeneration      string
		ignoreFamily     bool
		ignoreGeneration bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "similar kind",
			args: args{"General Purpose", "t3.2xlarge", "current",
				"General Purpose", "t3.4xlarge", "current", false, false},
			want: true,
		},
		{
			name: "different generations",
			args: args{"General Purpose", "t3.2xlarge", "current",
				"General Purpose", "t2.2xlarge", "previous", false, false},
			want: false,
		},
		{
			name: "ignore different generations",
			args: args{"General Purpose", "t3.2xlarge", "current",
				"General Purpose", "t2.2xlarge", "previous", false, true},
			want: true,
		},
		{
			name: "different families",
			args: args{"General Purpose", "m5.2xlarge", "current",
				"Compute Optimized", "c5.2xlarge", "current", false, false},
			want: false,
		},
		{
			name: "ignore different families",
			args: args{"General Purpose", "m5.2xlarge", "current",
				"Compute Optimized", "c5.2xlarge", "current", true, false},
			want: true,
		},
		{
			name: "different types",
			args: args{"General Purpose", "t3.2xlarge", "current",
				"General Purpose", "m5.2xlarge", "current", false, false},
			want: false,
		},
		{
			name: "both metal",
			args: args{"General Purpose", "m5d.metal", "current",
				"General Purpose", "m6g.metal", "current", false, false},
			want: true,
		},
		{
			name: "fail: one non-metal",
			args: args{"General Purpose", "m5.medium", "current",
				"General Purpose", "m6g.metal", "current", false, false},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSimilarKind(tt.args.oFamily, tt.args.oType, tt.args.oGeneration, tt.args.nFamily, tt.args.nType, tt.args.nGeneration, tt.args.ignoreFamily, tt.args.ignoreGeneration); got != tt.want {
				t.Errorf("isSimilarKind() = %v, want %v", got, tt.want)
			}
		})
	}
}
