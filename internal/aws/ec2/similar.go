package ec2

import (
	"log"
	"reflect"
	"sort"

	ec2instancesinfo "github.com/cristim/ec2-instances-info"
)

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

// InstanceTypeWeight EC2 details (type, weight)
type InstanceTypeWeight struct {
	InstanceType string // instance type name
	Weight       int    // Weight by # of vCPU
	// spotPrice    float32 // spot price
}

func GetSimilarTypes(instanceType string) []InstanceTypeWeight {
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
			if original.Arch[0] == nt.Arch[0] &&
				// same number of GPU
				original.GPU == nt.GPU &&
				// CPU/2 <= similar CPU <= CPU*2
				(nt.VCPU <= original.VCPU*2 && nt.VCPU >= original.VCPU/2) &&
				// similar family: general, memory, compute, storage accelerated
				original.Family == nt.Family &&
				original.InstanceType[:1] == nt.InstanceType[:1] {
				candidates = append(candidates, InstanceTypeWeight{nt.InstanceType, nt.VCPU})
			}
		}
		// sort candidates by Weight
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
