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

import "testing"

func TestResourceCgroupCPUValues(t *testing.T) {
	tests := []struct {
		name       string
		quota      int64
		period     int64
		wantQuota  string
		wantPeriod string
	}{
		{
			name:       "limited",
			quota:      400000,
			period:     100000,
			wantQuota:  "400000",
			wantPeriod: "100000",
		},
		{
			name:       "unlimited defaults period",
			quota:      -1,
			wantQuota:  "-1",
			wantPeriod: "100000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			quota, period := cgroupCPUValues(test.quota, test.period)
			if quota != test.wantQuota || period != test.wantPeriod {
				t.Fatalf("cgroupCPUValues(%d, %d) = (%q, %q), want (%q, %q)", test.quota, test.period, quota, period, test.wantQuota, test.wantPeriod)
			}
		})
	}
}

func TestResourceCgroupPaths(t *testing.T) {
	for _, test := range []struct {
		path string
		want []string
	}{
		{path: "", want: []string{"/"}},
		{path: "/", want: []string{"/"}},
		{path: "/container", want: []string{"/", "/container"}},
	} {
		got := resourceCgroupPaths(test.path)
		if len(got) != len(test.want) {
			t.Fatalf("resourceCgroupPaths(%q) = %v, want %v", test.path, got, test.want)
		}
		for i := range got {
			if got[i] != test.want[i] {
				t.Fatalf("resourceCgroupPaths(%q) = %v, want %v", test.path, got, test.want)
			}
		}
	}
}
