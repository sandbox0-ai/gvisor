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

package usage

import "testing"

func TestSetTotalMemoryLimit(t *testing.T) {
	SetTotalMemoryLimit(0)
	t.Cleanup(func() {
		SetTotalMemoryLimit(0)
	})

	const hostMemory = uint64(64 << 30)
	if got, want := TotalMemory(hostMemory, 0), hostMemory; got != want {
		t.Fatalf("TotalMemory() without limit = %d, want %d", got, want)
	}

	for _, limit := range []uint64{128 << 20, 2 << 30, 16 << 30} {
		SetTotalMemoryLimit(limit)
		if got := TotalMemory(hostMemory, 0); got != limit {
			t.Fatalf("TotalMemory() after SetTotalMemoryLimit(%d) = %d, want %d", limit, got, limit)
		}
		if got := TotalMemoryLimit(); got != limit {
			t.Fatalf("TotalMemoryLimit() = %d, want %d", got, limit)
		}
	}

	SetTotalMemoryLimit(0)
	if got, want := TotalMemory(1<<30, 0), uint64(defaultMinimumTotalMemoryBytes); got != want {
		t.Fatalf("TotalMemory() after clearing limit = %d, want %d", got, want)
	}
}
