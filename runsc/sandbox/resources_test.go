// Copyright 2026 The gVisor Authors.
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

package sandbox

import (
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func TestApplicationCoresFromQuota(t *testing.T) {
	tests := []struct {
		name      string
		cpuNum    int
		quota     int64
		period    int64
		fromQuota bool
		want      int
	}{
		{name: "disabled", cpuNum: 8, quota: 400000, period: 100000, want: 8},
		{name: "unlimited", cpuNum: 8, quota: -1, period: 100000, fromQuota: true, want: 8},
		{name: "fractional minimum", cpuNum: 8, quota: 15000, period: 100000, fromQuota: true, want: 2},
		{name: "four CPUs", cpuNum: 8, quota: 400000, period: 100000, fromQuota: true, want: 4},
		{name: "quota above cpuset", cpuNum: 8, quota: 1200000, period: 100000, fromQuota: true, want: 8},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := applicationCoresFromQuota(test.cpuNum, test.quota, test.period, test.fromQuota); got != test.want {
				t.Fatalf("applicationCoresFromQuota() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestResourceValuesFromUpdate(t *testing.T) {
	const (
		hostMemory    = uint64(64 << 30)
		currentMemory = uint64(128 << 20)
	)
	currentQuota := int64(15000)
	currentPeriod := int64(100000)

	quota := int64(400000)
	period := uint64(100000)
	memory := int64(16 << 30)
	res := &specs.LinuxResources{
		CPU:    &specs.LinuxCPU{Quota: &quota, Period: &period},
		Memory: &specs.LinuxMemory{Limit: &memory},
	}
	gotQuota, gotPeriod, gotMemory := resourceValuesFromUpdate(currentQuota, currentPeriod, hostMemory, currentMemory, res)
	if gotQuota != quota || gotPeriod != int64(period) || gotMemory != uint64(memory) {
		t.Fatalf("resourceValuesFromUpdate() = (%d, %d, %d), want (%d, %d, %d)", gotQuota, gotPeriod, gotMemory, quota, period, memory)
	}

	unlimitedQuota := int64(-1)
	unlimitedMemory := int64(-1)
	res = &specs.LinuxResources{
		CPU:    &specs.LinuxCPU{Quota: &unlimitedQuota},
		Memory: &specs.LinuxMemory{Limit: &unlimitedMemory},
	}
	gotQuota, gotPeriod, gotMemory = resourceValuesFromUpdate(currentQuota, currentPeriod, hostMemory, currentMemory, res)
	if gotQuota != unlimitedQuota || gotPeriod != currentPeriod || gotMemory != hostMemory {
		t.Fatalf("resourceValuesFromUpdate() for unlimited resources = (%d, %d, %d), want (%d, %d, %d)", gotQuota, gotPeriod, gotMemory, unlimitedQuota, currentPeriod, hostMemory)
	}

	zeroQuota := int64(0)
	zeroPeriod := uint64(0)
	zeroMemory := int64(0)
	res = &specs.LinuxResources{
		CPU:    &specs.LinuxCPU{Quota: &zeroQuota, Period: &zeroPeriod},
		Memory: &specs.LinuxMemory{Limit: &zeroMemory},
	}
	gotQuota, gotPeriod, gotMemory = resourceValuesFromUpdate(currentQuota, currentPeriod, hostMemory, currentMemory, res)
	if gotQuota != currentQuota || gotPeriod != currentPeriod || gotMemory != currentMemory {
		t.Fatalf("resourceValuesFromUpdate() for zero values = (%d, %d, %d), want (%d, %d, %d)", gotQuota, gotPeriod, gotMemory, currentQuota, currentPeriod, currentMemory)
	}
}
