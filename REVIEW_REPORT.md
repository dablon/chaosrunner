# ChaosRunner - Full Tool Review Report

**Date:** 2026-03-02
**Tested against:** Minikube cluster, namespace `chaos-test`
**Target workload:** nginx-test Deployment (3 replicas)
**Reviewer:** Claude (Automated QA Review)

---

## Executive Summary

ChaosRunner is a lightweight Kubernetes chaos engineering CLI tool written in Go. It provides 5 experiment types to test cluster resilience. After running every experiment against a live minikube cluster, the tool **works** but has **significant issues** in code quality, test coverage, reliability, and real-world usefulness.

**Verdict: Functional MVP, not production-ready. Needs serious work to be useful beyond demos.**

| Category | Score | Notes |
|----------|-------|-------|
| Does it work? | 7/10 | 4 of 5 experiments execute; network-latency silently fails |
| Code Quality | 4/10 | Bugs, duplicated logic, shell-out antipatterns |
| Test Coverage | 1/10 | ~3 trivial tests, 0% experiment coverage |
| Usefulness | 5/10 | Nice output but lacks configurability and actionable insights |
| UX/Output | 8/10 | Clean, emoji-based progress output with real-time metrics |
| Error Handling | 4/10 | Errors silently swallowed in multiple places |
| Security | 3/10 | Shells out via kubectl exec with unsanitized inputs |

---

## 1. Live Test Results

### 1.1 Basic Commands

| Command | Result | Notes |
|---------|--------|-------|
| `chaosrunner list` | PASS | Lists all 5 experiments |
| `chaosrunner version` | PASS | Prints "1.0.0" |
| `chaosrunner run` (no args) | PASS | Shows usage error correctly |
| `chaosrunner run invalid-name` | PASS | Clean error with suggestion to use `list` |
| `chaosrunner --help` | PASS | Shows cobra-generated help |

### 1.2 pod-kill (30s, chaos-test namespace)

**Result: PASS - Works well**

- Killed 15 pods in 30 seconds across 15 iterations
- Average kill time: 0.01s
- Average recovery time: 0.00s (misleading - see bugs below)
- Correctly shows real-time iteration progress
- Pods were actually deleted and recreated by the ReplicaSet

**Issues found:**
- Recovery time reports 0.00s because `WaitForPodReady` waits for the **deleted** pod name, which immediately errors and returns. It should wait for a **new** pod to appear.
- The tool killed and re-killed pods it had just created, creating a churning loop. With 3 replicas and 2s sleep, it created 15 kills in 30s which is extremely aggressive.
- "2/3 pods running" during iterations shows the system was constantly degraded.

### 1.3 cpu-stress (30s, chaos-test namespace)

**Result: PASS - Works with fallback chain**

- Successfully installed stress-ng inside the nginx container via apk
- Real-time progress bar with CPU metrics (0m -> 18m)
- Clean before/after comparison
- Pod survived without restarts

**Issues found:**
- CPU went to 18m (millicores) - barely noticeable. For nginx with no CPU limits, 4 workers on a minikube VM is not meaningful stress.
- The fallback chain installs packages into the running container (apt/apk/yum). This modifies the container filesystem, which is a side effect that outlasts the experiment.
- Hardcoded 4 workers with no configurability.

### 1.4 memory-hog (30s, chaos-test namespace)

**Result: PARTIAL PASS - Blocking behavior issue**

- Memory went from 265Mi to 758Mi (noticeable impact)
- No OOM kill (no limits set on nginx pods)
- Before/after comparison works

**Issues found:**
- The `memCmd.Run()` call on line 71 is **blocking**. The monitoring loop (lines 76-91) runs **after** the stress finishes, making real-time monitoring useless. The stress and monitoring should run concurrently like cpu-stress does.
- Header says "Memory: 512MB" but the actual stress command uses 256MB. Misleading.
- Duration capped at 300s silently - user gets no warning.
- The raw `kubectl top` output is printed including the pod name column, making it ugly: `nginx-test-566dbd78d4-5njck   1001m   758Mi`

### 1.5 network-latency (20s, chaos-test namespace)

**Result: FAIL - Silently fails**

- The tool reports "Network latency applied (5s delay)" even though `tc` is not installed in the nginx container
- Output shows: `sh: 1: tc: not found` but the tool ignores the error and continues
- Claims success even though zero network latency was actually injected
- Cleanup also silently fails (nothing to clean up)

**This experiment is useless on standard containers without iproute2/tc installed.** The tool should detect this and warn the user.

### 1.6 disk-fill (20s, chaos-test namespace)

**Result: PASS - Works correctly**

- Created 5 iterations of 100MB files
- Files properly created and cleaned up each iteration
- Before/after disk comparison works (18% in both cases since cleanup happens per iteration)

**Issues found:**
- It creates a file, then deletes it, then creates it again. Disk is never actually filled - it oscillates between 100MB over and baseline. The experiment name is "disk-fill" but it never actually fills anything.
- No configurable fill size.
- 100MB on a 1TB filesystem is meaningless (0.01% impact).

---

## 2. Code Quality Issues (Bugs)

### BUG 1: Restart count double-counted in GetPodStats

[k8s.go:123-136](internal/client/k8s.go#L123-L136):
```go
for _, cs := range pod.Status.ContainerStatuses {
    if cs.State.Running != nil {
        stats.Restarts = cs.RestartCount  // Set here
    }
}
if len(pod.Status.ContainerStatuses) > 0 {
    for _, cs := range pod.Status.ContainerStatuses {
        stats.Restarts += cs.RestartCount  // Added AGAIN
    }
}
```
Restarts are counted twice for running containers. If a container has 2 restarts, it'll report 4.

### BUG 2: handler init() clears KUBECONFIG

[handler.go:92-94](internal/handler/handler.go#L92-L94):
```go
func init() {
    os.Setenv("KUBECONFIG", "")
}
```
This **clears the KUBECONFIG environment variable** on startup. The tool works anyway because `clientcmd.NewDefaultClientConfigLoadingRules()` falls back to `~/.kube/config`, but if someone has a custom KUBECONFIG path set, this destroys it. This is almost certainly a debugging leftover.

### BUG 3: ParseDuration prepends "0"

[experiment.go:27](internal/experiments/experiment.go#L27):
```go
d = "0" + d  // "30s" becomes "030s"
```
This works by accident (Go's parser handles "030s") but it's a hack. "0" + "1h30m" = "01h30m" which also works. But "0" + "" = "0" which is valid 0 duration. This masks invalid input.

### BUG 4: pod-kill WaitForPodReady waits for deleted pod

[pod_kill.go:104](internal/experiments/pod_kill.go#L104):
```go
err = e.k8sClient.WaitForPodReady(namespace, targetPod.Name, 60*time.Second)
```
The pod was just deleted. A replacement pod will have a **different name**. This waits for a pod that no longer exists, gets errors on each poll, and times out or exits early giving a fake "0.00s recovery time."

### BUG 5: CaptureResources only captures last container

[k8s.go:183-188](internal/client/k8s.go#L183-L188):
```go
for _, c := range pod.Status.ContainerStatuses {
    p.Restarts = c.RestartCount  // Overwritten each iteration
}
```
In multi-container pods, only the last container's restart count is kept.

### BUG 6: K8sPod.Ready field type conflict

`K8sPod.Ready` is a `string` ("1/1" or "0/1"), but `PodStats.Ready` is a `bool`. Two different types for the same concept.

---

## 3. Architectural Issues

### 3.1 Shelling out to kubectl instead of using the K8s API

The tool initializes a proper `kubernetes.Clientset` but then uses `exec.Command("kubectl", ...)` for most actual work:
- CPU metrics: `kubectl top pod`
- Stress injection: `kubectl exec ... stress-ng`
- Disk operations: `kubectl exec ... dd/df`
- Network config: `kubectl exec ... tc`

This means:
- kubectl must be installed and configured separately
- Two authentication paths (clientset + kubectl) that could diverge
- No structured error handling from kubectl output
- Shell injection risk if namespace/pod names contain special characters

### 3.2 No experiment isolation

All experiments pick the first running pod (`GetRunningPod` returns the first match). There's no way to:
- Target a specific pod
- Target a specific deployment/statefulset
- Exclude system pods
- Run against multiple pods simultaneously
- Use label selectors

### 3.3 No cleanup on Ctrl+C

If the user interrupts during network-latency or cpu-stress, there's no signal handler. The tc qdisc or stress process stays in the container. Only the happy path has cleanup.

### 3.4 No structured output

Everything is `fmt.Printf`. There's no way to:
- Export results as JSON/YAML
- Pipe results to monitoring systems
- Parse results programmatically
- Store historical results

---

## 4. What a User Would Actually Need (Gap Analysis)

| Feature | ChaosRunner Has It? | Industry Standard (Litmus/Chaos Mesh) |
|---------|---------------------|---------------------------------------|
| Pod kill | Yes | Yes |
| CPU stress | Yes (with fallbacks) | Yes (via sidecar, no container mutation) |
| Memory stress | Partially (blocking bug) | Yes |
| Network latency | Broken on most containers | Yes (via network policies/sidecars) |
| Disk pressure | Fake (creates/deletes in loop) | Yes (actual pressure) |
| Target by label selector | No | Yes |
| Target by deployment name | No | Yes |
| Configurable intensity | No (all hardcoded) | Yes |
| Scheduled chaos | No | Yes (CronJob-like) |
| Rollback/abort | No (no Ctrl+C handler) | Yes |
| JSON/YAML output | No | Yes |
| Metrics export (Prometheus) | No (README claims it, code doesn't) | Yes |
| RBAC/permissions check | No | Yes |
| Dry-run mode | No | Yes |
| Concurrent experiments | No | Yes |
| Web UI/Dashboard | No | Yes |
| Webhook notifications | No | Yes |
| Container-native injection | No (modifies containers via exec) | Yes (sidecar injection) |

---

## 5. Test Coverage Analysis

```
Packages with tests:     2 / 5 (40%)
Total test functions:    3
Test lines of code:      ~24
Estimated coverage:      <5%
```

What's tested:
- `config.Default()` returns port 8080 (1 assertion)
- `handler.New()` returns non-nil (1 assertion)
- `handler.initK8s()` with nil client (basically tests that Init() runs)

What's NOT tested:
- Zero experiment logic
- Zero K8s client operations
- Zero CLI command parsing
- Zero duration parsing
- Zero error paths
- No mocking of K8s API
- No integration tests

---

## 6. Security Concerns

1. **Shell injection via kubectl exec**: Pod names and namespaces are passed directly into shell commands. A malicious pod name like `pod; rm -rf /` would be dangerous.
2. **Container mutation**: Installing packages (apt-get/apk/yum) inside running containers violates immutability principles and could trigger security scanners.
3. **No RBAC validation**: Tool doesn't check if the service account has delete/exec permissions before attempting operations.
4. **handler init() clears KUBECONFIG**: Side effect that could affect other tools in the same process.

---

## 7. What Works Well

1. **UX is genuinely good**: The emoji-based output with progress bars, before/after comparisons, and structured phases (DIAGNOSIS -> EXECUTION -> RESULTS) is clean and readable. This is better than many CLI tools.
2. **pod-kill core logic is solid**: Despite the recovery time bug, the actual killing and iteration loop works correctly and the ReplicaSet recovery is properly validated.
3. **CPU stress fallback chain is clever**: Trying stress-ng -> stress -> install -> shell loop with auto-restart is resilient.
4. **Binary size is reasonable**: ~63MB for a Go binary with K8s client libraries is normal.
5. **CLI structure is clean**: Cobra is well-used, flags make sense, help text is good.
6. **Error messages for invalid input are helpful**: "Use 'chaosrunner list'" is a nice touch.

---

## 8. Verdict: Is This Tool Useful?

**For learning/demos:** Yes. The output looks professional, it runs against a real cluster, and it demonstrates chaos engineering concepts clearly.

**For actual chaos engineering:** No. The bugs (fake recovery times, silent failures, non-functional network latency, fake disk fill) mean the data it produces is unreliable. A team using this to validate their resilience would get false confidence.

**For a portfolio project:** It needs the bugs fixed, real tests added, and at minimum: label selectors, configurable intensity, JSON output, and signal handling. In its current state, any senior engineer reviewing this would spot the issues immediately.

---

## 9. Priority Fix List

### Critical (Tool gives wrong/misleading results):
1. Fix pod-kill recovery time measurement (wait for NEW pod, not deleted one)
2. Fix network-latency to detect missing `tc` and fail explicitly
3. Fix restart count double-counting in GetPodStats
4. Remove `handler.init()` that clears KUBECONFIG

### High (Limits real usefulness):
5. Make memory-hog stress run concurrently with monitoring (non-blocking)
6. Fix disk-fill to actually accumulate files instead of create/delete loop
7. Add Ctrl+C signal handler with cleanup
8. Add label selector flag for pod targeting
9. Fix header saying "512MB" when actually using 256MB

### Medium (Code quality):
10. Add unit tests for experiments (mock K8s client)
11. Add JSON output mode
12. Make stress parameters configurable (workers, memory size, latency amount)
13. Sanitize inputs passed to shell commands
14. Remove `ParseDuration` hack (prepending "0")

### Low (Nice to have):
15. Add dry-run mode
16. Add RBAC pre-check
17. Use K8s API exec instead of shelling out to kubectl
18. Add Prometheus metrics endpoint
19. Support multi-container pods properly

---

*Report generated from live testing against minikube with 3 nginx replicas in the chaos-test namespace. All experiments were run with short durations (20-30s) to validate core functionality.*
