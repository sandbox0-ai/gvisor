// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kernel

import (
	"testing"

	"gvisor.dev/gvisor/pkg/sentry/kernel/sched"
)

func TestTaskCPU(t *testing.T) {
	for _, test := range []struct {
		mask sched.CPUSet
		tid  ThreadID
		cpu  int32
	}{
		{
			mask: []byte{0xff},
			tid:  1,
			cpu:  1,
		},
		{
			mask: []byte{0xff},
			tid:  10,
			cpu:  2,
		},
		{
			// more than 8 cpus.
			mask: []byte{0xff, 0xff},
			tid:  10,
			cpu:  10,
		},
		{
			// missing the first cpu.
			mask: []byte{0xfe},
			tid:  1,
			cpu:  2,
		},
		{
			mask: []byte{0xfe},
			tid:  10,
			cpu:  4,
		},
		{
			// missing the fifth cpu.
			mask: []byte{0xef},
			tid:  10,
			cpu:  3,
		},
		{
			// only the fifth cpu.
			mask: []byte{0x10},
			tid:  10,
			cpu:  4,
		},
	} {
		assigned := assignCPU(test.mask, test.tid)
		if test.cpu != assigned {
			t.Errorf("assignCPU(%v, %v) got %v, want %v", test.mask, test.tid, assigned, test.cpu)
		}
	}

}

func TestResizeCPUSet(t *testing.T) {
	tests := []struct {
		name     string
		mask     sched.CPUSet
		oldCores uint
		newCores uint
		wantCPUs []uint
	}{
		{
			name:     "grow full mask",
			mask:     sched.NewFullCPUSet(2),
			oldCores: 2,
			newCores: 4,
			wantCPUs: []uint{0, 1, 2, 3},
		},
		{
			name:     "grow explicit affinity",
			mask:     cpuSetOf(2, 1),
			oldCores: 2,
			newCores: 4,
			wantCPUs: []uint{1},
		},
		{
			name:     "shrink full mask",
			mask:     sched.NewFullCPUSet(4),
			oldCores: 4,
			newCores: 2,
			wantCPUs: []uint{0, 1},
		},
		{
			name:     "shrink removes entire affinity",
			mask:     cpuSetOf(4, 3),
			oldCores: 4,
			newCores: 2,
			wantCPUs: []uint{0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resizeCPUSet(test.mask, test.oldCores, test.newCores)
			var gotCPUs []uint
			got.ForEachCPU(func(cpu uint) {
				gotCPUs = append(gotCPUs, cpu)
			})
			if len(gotCPUs) != len(test.wantCPUs) {
				t.Fatalf("resizeCPUSet() CPUs = %v, want %v", gotCPUs, test.wantCPUs)
			}
			for i := range gotCPUs {
				if gotCPUs[i] != test.wantCPUs[i] {
					t.Fatalf("resizeCPUSet() CPUs = %v, want %v", gotCPUs, test.wantCPUs)
				}
			}
		})
	}
}

func cpuSetOf(size uint, cpus ...uint) sched.CPUSet {
	set := sched.NewCPUSet(size)
	for _, cpu := range cpus {
		set.Set(cpu)
	}
	return set
}
