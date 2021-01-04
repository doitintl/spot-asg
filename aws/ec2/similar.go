package ec2

import (
	"log"
	"reflect"
	"sort"
	"strings"

	ec2instancesinfo "github.com/cristim/ec2-instances-info"
)

const metal = "metal"

var (
	ec2data *ec2instancesinfo.InstanceData
)

func init() {
	data, err := ec2instancesinfo.Data() // load data only once
	if err != nil {
		log.Fatalln("failed to load binary serialized JSON sourced from ec2instances.info")
	}
	ec2data = data
}

// InstanceTypeWeight EC2 instance type record that contains type name and weight (equal to the VCPU number)
type InstanceTypeWeight struct {
	// InstanceType instance type name, like `m5.4xlarge`
	InstanceType string
	// Weight instance weight for MixedInstancePolicy; equal to the VCPU number
	Weight int
	// spotPrice    float32 // spot price
}

// A Config is used for tuning the EC2 similarity algorithm
type Config struct {
	// IgnoreFamily ignore instance family
	IgnoreFamily bool
	// IgnoreGeneration ignore instance generation
	IgnoreGeneration bool
	// MultiplyFactorUpper a multiplier for the upper limit of the VCPU size
	MultiplyFactorUpper int
	// MultiplyFactorUpper a multiplier for the lower limit of the VCPU size
	MultiplyFactorLower int
}

// GetSimilarTypes find EC2 instances, that are similar to the specified EC2 instance type.
// The algorithm compares between instance types over multiple dimensions.
// It returns a list of "similar" EC2 instance types (with weights)
func GetSimilarTypes(instanceType string, config Config) []InstanceTypeWeight {
	var candidates []InstanceTypeWeight
	for _, it := range *ec2data {
		if it.InstanceType != instanceType {
			continue
		}
		// found original instance type
		original := it
		// find similar instances
		for _, nt := range *ec2data {
			// skip original instance type, it will be added later as a 1st element
			if reflect.DeepEqual(original, nt) {
				continue
			}
			if isSimilarGPU(original.GPU, nt.GPU) &&
				isSimilarCPU(original.VCPU, nt.VCPU, original.Arch, nt.Arch, config.MultiplyFactorUpper, config.MultiplyFactorLower) &&
				isSimilarKind(
					original.Family, original.InstanceType, original.Generation,
					nt.Family, nt.InstanceType, nt.Generation,
					config.IgnoreFamily, config.IgnoreGeneration) {
				candidates = append(candidates, InstanceTypeWeight{nt.InstanceType, nt.VCPU})
			}
		}
		// sort candidates by Weight, keep original weight first
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].Weight == original.VCPU {
				return true
			} else if candidates[j].Weight == original.VCPU {
				return false
			}
			return candidates[i].Weight < candidates[j].Weight
		})
		// prepend 1st element
		candidates = append([]InstanceTypeWeight{{original.InstanceType, original.VCPU}}, candidates...)
		// no need to continue
		break
	}

	return candidates
}

// if using GPU at least the same number of GPUs
func isSimilarGPU(oGPU, nGPU int) bool {
	return (oGPU == 0 && nGPU == 0) || (oGPU > 0 && oGPU <= nGPU)
}

// CPU/2 <= similar CPU <= CPU*2
// and the same VCPU architecture
func isSimilarCPU(oCPU, nCPU int, oArch, nArch []string, factorUp, factorLow int) bool {
	// original support more architecture platforms than new
	if len(oArch) > len(nArch) {
		return false
	}
	// check if original CPU architecture is a subset of new CPU architecture
	// no need to optimize, mostly 2 elements in slice
	for _, a := range oArch {
		subset := false
		for _, n := range nArch {
			if a == n {
				subset = true
				break
			}
		}
		if !subset {
			return false
		}
	}
	// last check: compare number of VPCU within allowed range
	return nCPU <= oCPU*factorUp && nCPU >= oCPU/factorLow
}

// similar kind
// 1. the same instance family
// 2. the same instance type
// 3. the same instance generation
func isSimilarKind(oFamily, oType, oGeneration, nFamily, nType, nGeneration string, ignoreFamily, ignoreGeneration bool) bool {
	if oFamily != nFamily && !ignoreFamily {
		return false
	}
	// analyze instance type
	oTypeInfo := strings.Split(oType, ".")
	nTypeInfo := strings.Split(nType, ".")
	// every instance type is composed from 2 dot separated strings
	if len(oTypeInfo) != 2 || len(nTypeInfo) != 2 {
		return false
	}
	// for metal instance type similar type should be metal
	if (oTypeInfo[1] == metal || nTypeInfo[1] == metal) && oTypeInfo[1] != nTypeInfo[1] {
		return false
	}
	// compare first instance type character: `t` and `m` both "General Purpose" but `t` is burstable
	if oTypeInfo[0][:1] != nTypeInfo[0][:1] && !ignoreFamily {
		return false
	}
	// the same generation
	if oGeneration != nGeneration && !ignoreGeneration {
		return false
	}
	// OK: similar kind
	return true
}
