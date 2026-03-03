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
		return fields[1]
	}
	return strings.TrimSpace(string(out))
}

func (e *CpuStressExperiment) Name() string {
	return "cpu-stress"
}

func (e *CpuStressExperiment) Run(namespace, duration string, opts *ExperimentOptions) error {
	workers := opts.Workers
	if workers <= 0 {
		workers = 4
	}

	if IsTextOutput(opts) {
		e.PrintHeader("cpu-stress", opts)
		fmt.Printf("   Namespace: %s\n", namespace)
		fmt.Printf("   Duration: %s\n", duration)
		fmt.Printf("   Stress workers: %d\n", workers)
	}

	result := &ExperimentResult{
		Experiment: "cpu-stress",
		Namespace:  namespace,
		Duration:   duration,
		Success:    true,
		Metrics:    make(map[string]interface{}),
	}

	dur, err := ParseDuration(duration)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return err
	}

	targetPod, err := e.k8sClient.GetRunningPod(opts.Ctx, namespace, opts.Selector)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return err
	}

	if IsTextOutput(opts) {
		fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

		statsBefore, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
		if err == nil {
			fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
			fmt.Printf("      Status: %s\n", statsBefore.Phase)
			fmt.Printf("      Restarts: %d\n", statsBefore.Restarts)
			fmt.Printf("      Age: %s\n", statsBefore.Age)
		}

		podsBefore, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

		fmt.Printf("\n⚙️  EXECUTION - Starting CPU Stress:\n")

		baselineCPU := getCPUMetric(targetPod.Name, namespace)
		fmt.Printf("   📊 CPU before stress: %s\n", baselineCPU)

		stressDuration := int(dur.Seconds())
		fmt.Printf("   🚀 Starting stress-ng (%d workers, %ds)...\n\n", workers, stressDuration)
	}

	stressDuration := int(dur.Seconds())

	stressScript := func(secs int, wkrs int) string {
		workers := fmt.Sprintf("%d", wkrs)
		return fmt.Sprintf("stress-ng --cpu %s --timeout %ds 2>/dev/null || stress -c %s -t %ds 2>/dev/null || (apt-get update -qq && apt-get install -y -qq stress-ng && stress-ng --cpu %s --timeout %ds) 2>/dev/null || (apk add --no-cache stress-ng && stress-ng --cpu %s --timeout %ds) 2>/dev/null || (yum install -y -q stress-ng && stress-ng --cpu %s --timeout %ds) 2>/dev/null || (for i in $(seq 1 %s); do while :; do :; done & done; sleep %d; kill 0)",
			workers, secs, workers, secs, workers, secs, workers, secs, workers, secs, workers, secs)
	}

	startStress := func(secs int) *exec.Cmd {
		return exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c", stressScript(secs, workers))
	}

	stressCmd := startStress(stressDuration)
	stressDone := make(chan error, 1)
	if err := stressCmd.Start(); err != nil {
		errMsg := fmt.Sprintf("failed to start stress: %v", err)
		result.Error = errMsg
		result.Success = false
		return fmt.Errorf("%s", errMsg)
	}
	go func() { stressDone <- stressCmd.Wait() }()

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

	if IsTextOutput(opts) {
		printProgress(0, getCPUMetric(targetPod.Name, namespace))
	}

	stressRunning := true
	retries := 0
	maxRetries := 2

	for time.Since(startTime) < dur {
		select {
		case <-opts.Ctx.Done():
			if stressRunning {
				_ = stressCmd.Process.Kill()
			}
			if IsTextOutput(opts) {
				fmt.Printf("\n⚠ Experiment cancelled\n")
			}
			result.Metrics["cancelled"] = true
			break
		case <-stressDone:
			if stressRunning {
				stressRunning = false
				elapsed := time.Since(startTime)
				remaining := dur - elapsed
				if remaining > 5*time.Second && retries < maxRetries {
					retries++
					if IsTextOutput(opts) {
						fmt.Printf("\n   ⚠ Stress exited early, restarting (%d/%d) for %s...\n", retries, maxRetries, remaining.Round(time.Second))
					}
					stressCmd = startStress(int(remaining.Seconds()))
					stressDone = make(chan error, 1)
					if err := stressCmd.Start(); err == nil {
						stressRunning = true
						go func() { stressDone <- stressCmd.Wait() }()
					}
				} else if retries >= maxRetries {
					if IsTextOutput(opts) {
						fmt.Printf("\n   ⚠ Stress tool unavailable in container. Monitoring pod for remaining %s...\n", remaining.Round(time.Second))
					}
				}
			}
		case <-ticker.C:
			elapsed := time.Since(startTime)
			if elapsed >= dur {
				break
			}
			if IsTextOutput(opts) {
				cpuNow := getCPUMetric(targetPod.Name, namespace)
				printProgress(elapsed, cpuNow)
			}
		}
	}

	if IsTextOutput(opts) {
		cpuFinal := getCPUMetric(targetPod.Name, namespace)
		printProgress(dur, cpuFinal)
		fmt.Println()

		if stressRunning {
			_ = stressCmd.Process.Kill()
		}

		fmt.Printf("\n   ✅ CPU stress completed\n")

		time.Sleep(2 * time.Second)

		fmt.Printf("\n📈 RESULTS - After Stress:\n")

		statsAfter, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
		if err == nil {
			fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
			fmt.Printf("      Status: %s\n", statsAfter.Phase)
			fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, 0)
			if statsAfter.Restarts > 0 {
				fmt.Printf("      ⚠ Pod was restarted during stress!\n")
			}
		}

		cpuAfter := getCPUMetric(targetPod.Name, namespace)
		fmt.Printf("   📊 CPU after stress: %s\n", cpuAfter)

		result.Metrics["cpu_before"] = getCPUMetric(targetPod.Name, namespace)
		result.Metrics["cpu_after"] = cpuAfter

		podsAfter, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		running := 0
		for _, p := range podsAfter {
			if p.Phase == "Running" {
				running++
			}
		}
		fmt.Printf("   ✓ Final state: %d/%d pods running\n", running, len(podsAfter))
		fmt.Printf("   ✓ Duration: %s\n", duration)

		e.PrintFooter(duration, opts)
		fmt.Printf("   CPU stress test completed\n")
	}

	if opts.Output == "json" {
		result.PrintJSON()
	}

	return nil
}
