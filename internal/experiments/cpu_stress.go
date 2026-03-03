package experiments

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dablon/chaosrunner/internal/client"
)

type CpuStressExperiment struct {
	BaseExperiment
	k8sClient *client.K8sClient
}

func NewCpuStressExperiment(k8s *client.K8sClient) *CpuStressExperiment {
	return &CpuStressExperiment{k8sClient: k8s}
}

func getCPUMetric(podName, namespace string) string {
	cmd := exec.Command("kubectl", "top", "pod", podName, "-n", namespace, "--no-headers")
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) == 0 {
		return "N/A"
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) >= 2 {
		return fields[1] // CPU column
	}
	return strings.TrimSpace(string(out))
}

func (e *CpuStressExperiment) Name() string {
	return "cpu-stress"
}

func (e *CpuStressExperiment) Run(namespace, duration string) error {
	e.PrintHeader("cpu-stress")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Stress workers: 4\n")

	dur, err := ParseDuration(duration)
	if err != nil {
		return err
	}

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

	statsBefore, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsBefore.Phase)
		fmt.Printf("      Restarts: %d\n", statsBefore.Restarts)
		fmt.Printf("      Age: %s\n", statsBefore.Age)
	}

	podsBefore, _ := e.k8sClient.GetPods(namespace)
	fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

	fmt.Printf("\n⚙️  EXECUTION - Starting CPU Stress:\n")

	baselineCPU := getCPUMetric(targetPod.Name, namespace)
	fmt.Printf("   📊 CPU before stress: %s\n", baselineCPU)

	stressDuration := int(dur.Seconds())

	fmt.Printf("   🚀 Starting stress-ng (4 workers, %ds)...\n\n", stressDuration)

	// Install + run stress with fallbacks for Debian, Alpine, and RHEL-based images
	stressScript := func(secs int) string {
		return fmt.Sprintf(strings.Join([]string{
			"stress-ng --cpu 4 --timeout %[1]ds 2>/dev/null",
			"|| stress -c 4 -t %[1]ds 2>/dev/null",
			"|| (apt-get update -qq && apt-get install -y -qq stress-ng && stress-ng --cpu 4 --timeout %[1]ds) 2>/dev/null",
			"|| (apk add --no-cache stress-ng && stress-ng --cpu 4 --timeout %[1]ds) 2>/dev/null",
			"|| (yum install -y -q stress-ng && stress-ng --cpu 4 --timeout %[1]ds) 2>/dev/null",
			// Last resort: pure shell CPU burn
			"|| (for i in 1 2 3 4; do while :; do :; done & done; sleep %[1]d; kill 0)",
		}, " "), secs)
	}

	startStress := func(secs int) *exec.Cmd {
		return exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c", stressScript(secs))
	}

	// Run stress in background so we can monitor in real-time
	stressCmd := startStress(stressDuration)
	stressDone := make(chan error, 1)
	if err := stressCmd.Start(); err != nil {
		return fmt.Errorf("failed to start stress: %w", err)
	}
	go func() { stressDone <- stressCmd.Wait() }()

	// Real-time progress monitoring
	startTime := time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	printProgress := func(elapsed time.Duration, cpu string) {
		pct := int(elapsed.Seconds() / dur.Seconds() * 100)
		if pct > 100 {
			pct = 100
		}
		barLen := 30
		filled := barLen * pct / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)
		fmt.Printf("\r   [%s] %3d%% | %s/%s | CPU: %-20s",
			bar, pct,
			elapsed.Round(time.Second),
			dur.Round(time.Second),
			cpu)
	}

	// Initial progress line
	printProgress(0, baselineCPU)

	stressRunning := true
	retries := 0
	maxRetries := 2
	for time.Since(startTime) < dur {
		select {
		case <-stressDone:
			if stressRunning {
				stressRunning = false
				elapsed := time.Since(startTime)
				remaining := dur - elapsed
				if remaining > 5*time.Second && retries < maxRetries {
					retries++
					fmt.Printf("\n   ⚠ Stress exited early, restarting (%d/%d) for %s...\n", retries, maxRetries, remaining.Round(time.Second))
					stressCmd = startStress(int(remaining.Seconds()))
					stressDone = make(chan error, 1)
					if err := stressCmd.Start(); err == nil {
						stressRunning = true
						go func() { stressDone <- stressCmd.Wait() }()
					}
				} else if retries >= maxRetries {
					fmt.Printf("\n   ⚠ Stress tool unavailable in container. Monitoring pod for remaining %s...\n", remaining.Round(time.Second))
				}
			}
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed >= dur {
				break
			}
			cpuNow := getCPUMetric(targetPod.Name, namespace)
			printProgress(elapsed, cpuNow)
		}
	}

	// Final progress update
	cpuFinal := getCPUMetric(targetPod.Name, namespace)
	printProgress(dur, cpuFinal)
	fmt.Println()

	// Kill stress if still running past duration
	if stressRunning {
		_ = stressCmd.Process.Kill()
	}

	fmt.Printf("\n   ✅ CPU stress completed\n")

	time.Sleep(2 * time.Second)

	fmt.Printf("\n📈 RESULTS - After Stress:\n")

	statsAfter, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsAfter.Phase)
		fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, statsBefore.Restarts)
		if statsAfter.Restarts > statsBefore.Restarts {
			fmt.Printf("      ⚠ Pod was restarted during stress!\n")
		}
	}

	cpuAfter := getCPUMetric(targetPod.Name, namespace)
	fmt.Printf("   📊 CPU after stress: %s\n", cpuAfter)

	podsAfter, _ := e.k8sClient.GetPods(namespace)
	running := 0
	for _, p := range podsAfter {
		if p.Phase == "Running" {
			running++
		}
	}
	fmt.Printf("   ✓ Final state: %d/%d pods running\n", running, len(podsAfter))
	fmt.Printf("   ✓ Duration: %s\n", duration)

	e.PrintFooter(duration)
	fmt.Printf("   CPU stress test completed\n")

	return nil
}
