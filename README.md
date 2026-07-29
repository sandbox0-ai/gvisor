# Sandbox0 gVisor Fork

![gVisor](g3doc/logo.png)

This repository is the production gVisor fork used by
[Sandbox0](https://github.com/sandbox0-ai/sandbox0). It tracks
[google/gvisor](https://github.com/google/gvisor) and carries a small set of
runtime changes that Sandbox0 cannot yet obtain from an upstream release.

This README is the inventory of those changes. Before replacing a Sandbox0
runtime with an official gVisor build, review the fork delta and complete the
replacement checks below.

The fork was last compared and synchronized with upstream `master` at
[`43396a1925`](https://github.com/google/gvisor/commit/43396a19255cdbaa230c4cfbffc2f8e2aff5f6b1)
on July 29, 2026. Neither runtime change listed below had an equivalent
implementation at that revision.

## Runtime Delta

| Change | Sandbox0 change | Why it is retained | Replacement impact |
| --- | --- | --- | --- |
| Direct containerd shim stats | [PR #1](https://github.com/sandbox0-ai/gvisor/pull/1), [`acdf789b6a`](https://github.com/sandbox0-ai/gvisor/commit/acdf789b6a1939040033933f2ec048237a54abd6) | Avoids starting one `runsc events --stats` process for every CRI stats request and sandbox container. | Upstream remains functionally usable, but replacing the fork reintroduces substantial CPU and fork/exec overhead on dense nodes. |
| Live resource-view updates | [PR #8](https://github.com/sandbox0-ai/gvisor/pull/8), [`43a3d25c8a`](https://github.com/sandbox0-ai/gvisor/commit/43a3d25c8ad3ac201b1cc0ef61f2d13e9ce8509d) | Makes CPU and memory changes from `runsc update` visible inside an already running sandbox. | Required for Sandbox0's idle-to-active in-place resize path. Without it, Kubernetes and the host cgroup resize while the guest continues to report its startup resources. |

Changes under `.github/` only support fork maintenance. They do not alter the
runtime binaries and do not block replacing this fork with upstream.

### Direct Containerd Shim Stats

Upstream `Runsc.Stats` starts a `runsc events --stats` subprocess on every call.
Kubelet CRI stats collection invokes this path for every sandbox container, so
the process cost becomes material on nodes with hundreds of sandboxes.

The fork:

- reads the existing runsc state while holding its advisory lock;
- caches sandbox-lifetime metadata, including the control socket and host
  cgroup;
- calls the existing `containerManager.Event` control RPC directly;
- applies the same host-cgroup CPU correction as the CLI path;
- invalidates the cache after a direct-path failure; and
- falls back to the upstream CLI implementation when the direct path is
  unavailable.

In the production comparison recorded in PR #1, the direct path reduced average
host CPU from 60.68% to 16.70%, CPU p95 from 100% to 36.70%, and forks from
271.30/s to 17.18/s. The baseline executed `runsc events --stats` 11,254 times
during the observation window; the direct path executed it zero times.

This is primarily a performance requirement. An upstream build can be considered
equivalent when its shim obtains the same stats without one subprocess per
container and preserves the existing CRI stats fields and CPU accounting.

### Live Resource-View Updates

Upstream `runsc update` updates the host cgroup and saved OCI spec. The Sentry's
application CPU count and reported total memory are initialized only when the
sandbox starts, so the values observed by applications do not follow a later
Kubernetes in-place resize.

The fork adds a Sentry resource-update RPC and refreshes:

- the application CPU count and CPU clock storage;
- existing task CPU affinity masks and assigned virtual CPUs;
- total memory reported through `/proc/meminfo` and `sysinfo(2)`;
- `/proc/cpuinfo`;
- `/sys/devices/system/cpu/{online,possible,present}`; and
- the sandbox-visible cgroup v1 CPU quota, CPU period, and memory limit at both
  the hierarchy root and the per-container child.

Explicit OCI update values take precedence over the parent cgroup snapshot.
This matters because kubelet may apply the pod parent cgroup before or after the
runtime call depending on resize direction. A new runsc binary also tolerates a
sandbox started by an older Sentry binary: if the new RPC is unavailable, it
logs a warning and preserves the original host-cgroup update behavior.

Current boundaries:

- CPU reporting retains gVisor's two-CPU minimum and never exceeds CPUs
  available on the host.
- Sandbox0 currently relies on the cgroup v1 guest view. This patch is not
  evidence that equivalent cgroup v2 views update correctly.
- CPU directory entries under `/sys/devices/system/cpu/cpuN` are created at
  boot. The online, possible, and present sets are dynamic.

This is a correctness requirement for Sandbox0. An upstream build is equivalent
only when a running sandbox observes both CPU and memory changes without being
restarted.

## Can Sandbox0 Replace This Fork?

Not yet without either losing the direct-stats optimization or breaking
guest-visible in-place resize. Do not decide from commit ancestry alone: an
upstream implementation may be equivalent without using the same code.

Before switching to an official gVisor release:

1. Compare the intended upstream release with the deployed Sandbox0 tag and
   review every runtime entry in the table above.
2. Confirm that repeated containerd CRI stats collection does not execute one
   `runsc events --stats` subprocess per sandbox container, or explicitly accept
   the measured performance regression.
3. Confirm that `runsc update` propagates CPU and memory changes into the
   running Sentry and all application-visible interfaces listed above.
4. Build `runsc`, `containerd-shim-runsc-v1`, and the expected sidecar binaries
   from the same upstream commit.
5. Run the focused tests and the remote Kubernetes smoke test below.
6. Update this README with the upstream commit or release that supersedes each
   fork change before removing it.

If both runtime rows are superseded and the smoke test passes, the `.github/`
fork-maintenance differences can be ignored and Sandbox0 can use the official
gVisor binaries directly.

## Replacement Smoke Test

Run this test on the same Kubernetes runtime class and node shape used by
Sandbox0 production:

1. Start a sandbox at `150m` CPU and `128Mi` memory.
2. Resize it in place to `900m` CPU and `2Gi` memory.
3. Resize it back to `300m` CPU and `256Mi` memory.
4. Verify after each transition:
   - Kubernetes spec and status contain the requested resources;
   - `restartCount` remains zero and resize conditions clear;
   - `getconf _NPROCESSORS_ONLN` reflects the effective CPU quota, subject to
     the two-CPU minimum and host CPU count;
   - `MemTotal` in `/proc/meminfo` reflects the requested memory;
   - `/proc/cpuinfo` and sysfs CPU sets agree with the effective CPU count; and
   - guest cgroup CPU quota/period and memory limit reflect the new values.
5. Exercise CRI sandbox stats at production-like sandbox density and verify
   both metric correctness and the absence of recurring stats subprocesses.

PR #8 was remotely validated with
`150m/128Mi -> 900m/2Gi -> 300m/256Mi` and `restartCount=0`. Memory and guest
cgroup values followed both resize directions. The validation node had two
physical CPUs, so dynamic growth beyond two CPUs is covered by unit tests and
must still be checked on production-sized nodes before an upstream replacement.

## Comparing with Upstream

For a clone whose `origin` points to this fork:

```sh
git remote add upstream https://github.com/google/gvisor.git
git fetch upstream master
git log --left-right --cherry-pick --oneline upstream/master...master
git diff --stat upstream/master...master
```

Compare against the exact upstream release intended for deployment as well as
upstream `master`. A fix on `master` is not available to Sandbox0 until it is
included in the pinned release or deliberately backported.

## Build and Test

gVisor uses Bazel, with `make` wrappers that run the canonical build container.
Build `runsc`, the containerd shim, and matched sidecar binaries from the same
commit:

```sh
mkdir -p bin
make copy TARGETS="//:release" DESTINATION=bin/
```

The output contains `runsc`, `containerd-shim-runsc-v1`, and the `gvisor-bin/`
sidecar directory. Keep them together when packaging the runtime; current
upstream runsc versions enforce matching sidecar releases.

Relevant focused tests for the fork delta include:

```sh
make test TARGETS="//pkg/shim/v1/runsccmd:runsccmd_test"
make test TARGETS="//pkg/sentry/usage:usage_test"
make test TARGETS="//pkg/sentry/kernel:kernel_test"
make test TARGETS="//pkg/sentry/fsimpl/sys:sys_integration_test"
make test TARGETS="//pkg/sentry/fsimpl/proc:proc_test" \
  OPTIONS="--test_filter=TestCPUInfoReflectsApplicationCores"
make test TARGETS="//pkg/sentry/control:control_test" \
  OPTIONS="--test_filter=TestResource.*"
make test TARGETS="//runsc/sandbox:sandbox_test" \
  OPTIONS="--test_filter=TestResource.*"
```

See the upstream
[build documentation](https://gvisor.dev/docs/user_guide/install/) and
[contribution guide](CONTRIBUTING.md) for the full build and test suites.

## About Upstream gVisor

[gVisor](https://gvisor.dev) is a userspace application kernel written in Go. It
provides an OCI runtime named `runsc` and reduces the host-kernel surface exposed
to sandboxed applications.

Upstream documentation, architecture, security reporting, and community
channels remain authoritative:

- [Documentation](https://gvisor.dev/docs/)
- [Source repository](https://github.com/google/gvisor)
- [Security policy](SECURITY.md)
- [Governance](GOVERNANCE.md)
- [Contributing](CONTRIBUTING.md)

This fork retains the upstream Apache 2.0 license. See [LICENSE](LICENSE).
