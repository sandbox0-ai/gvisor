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

package control

import (
	"fmt"
	"strconv"

	"gvisor.dev/gvisor/pkg/log"
	"gvisor.dev/gvisor/pkg/sentry/kernel"
	"gvisor.dev/gvisor/pkg/sentry/usage"
)

// Resources contains the state for sandbox resource control commands.
type Resources struct {
	Kernel *kernel.Kernel
}

// ResourcesUpdateArgs contains sandbox-wide resource values visible to
// applications.
type ResourcesUpdateArgs struct {
	ApplicationCores uint   `json:"application_cores"`
	TotalMemoryBytes uint64 `json:"total_memory_bytes"`
	CPUQuotaMicros   int64  `json:"cpu_quota_micros"`
	CPUPeriodMicros  int64  `json:"cpu_period_micros"`
	CgroupPath       string `json:"cgroup_path"`
}

// Update changes the CPU and memory values reported inside the sandbox.
func (r *Resources) Update(args *ResourcesUpdateArgs, _ *struct{}) error {
	if args == nil {
		return fmt.Errorf("resources update arguments are required")
	}
	if err := r.Kernel.SetApplicationCores(args.ApplicationCores); err != nil {
		return err
	}
	usage.SetTotalMemoryLimit(args.TotalMemoryBytes)
	r.updateCgroupView(args)
	return nil
}

func (r *Resources) updateCgroupView(args *ResourcesUpdateArgs) {
	ctx := r.Kernel.SupervisorContext()
	registry := r.Kernel.CgroupRegistry()

	for _, path := range resourceCgroupPaths(args.CgroupPath) {
		if cpu, err := registry.FindCgroup(ctx, kernel.CgroupControllerCPU, path); err == nil {
			quota, period := cgroupCPUValues(args.CPUQuotaMicros, args.CPUPeriodMicros)
			periodErr := cpu.WriteControl(ctx, "cpu.cfs_period_us", period)
			quotaErr := cpu.WriteControl(ctx, "cpu.cfs_quota_us", quota)
			if periodErr != nil || quotaErr != nil {
				log.Warningf("Unable to update sandbox-visible CPU cgroup values at %q: cpu.cfs_period_us: %v, cpu.cfs_quota_us: %v", path, periodErr, quotaErr)
			}
		}

		if memory, err := registry.FindCgroup(ctx, kernel.CgroupControllerMemory, path); err == nil {
			value := strconv.FormatUint(args.TotalMemoryBytes, 10)
			if err := memory.WriteControl(ctx, "memory.limit_in_bytes", value); err != nil {
				log.Warningf("Unable to update sandbox-visible memory cgroup value at %q: %v", path, err)
			}
		}
	}
}

func resourceCgroupPaths(path string) []string {
	if path == "" || path == "/" {
		return []string{"/"}
	}
	return []string{"/", path}
}

func cgroupCPUValues(quota, period int64) (string, string) {
	if period <= 0 {
		period = 100000
	}
	periodValue := strconv.FormatInt(period, 10)
	if quota <= 0 {
		return "-1", periodValue
	}
	return strconv.FormatInt(quota, 10), periodValue
}
