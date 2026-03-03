package experiments

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/dablon/chaosrunner/internal/client"
)

type MemoryHogExperiment struct {
	BaseExperiment
	k8sClient *client.K8sClient
}

func NewMemoryHogExperiment(k8s *client.K8sClient) *MemoryHogExperiment {
	return &MemoryHogExperiment{k8sClient: k8s}
}

func (e *MemoryHogExperiment) Name() string {
	return "memory-hog"
}

func (e *MemoryHogExperiment) Run(namespace, duration string, opts *ExperimentOptions) error {
	memSize := opts.Memory
	if memSize == "" {
		memSize = "256M"
	}

	if IsTextOutput(opts) {
		e.PrintHeader("memory-hog", opts)
		fmt.Printf("   Namespace: %s\n", namespace)
		fmt.Printf("   Duration: %s\n", duration)
		fmt.Printf("   Memory: %s\n", memSize)
	}

	result := &ExperimentResult{
		Experiment: "memory-hog",
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
			fmt.Printf("      Memory Request: %s\n", targetPod.MemoryRequest)
			fmt.Printf("      Memory Limit: %s\n", targetPod.MemoryLimit)
		}

		podsBefore, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

		fmt.Printf("\n⚙️  EXECUTION - Starting Memory Stress:\n")

		cmd := exec.Command("kubectl", "top", "pod", targetPod.Name, "-n", namespace, "--no-headers")
		output, _ := cmd.CombinedOutput()
		if len(output) > 0 {
			fmt.Printf("   📊 Memory before stress: %s\n", string(output))
		}

		memDuration := int(dur.Seconds())
		if memDuration > 300 {
			memDuration = 300
		}

		fmt.Printf("   🚀 Starting memory stress (%s, %ds)...\n", memSize, memDuration)
	}

	memDuration := int(dur.Seconds())
	if memDuration > 300 {
		memDuration = 300
	}

	memCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		fmt.Sprintf("stress-ng --vm 2 --vm-bytes %s --timeout %ds 2>/dev/null || (dd if=/dev/zero of=/tmp/memhog bs=1M count=%s &); sleep %ds; rm -f /tmp/memhog 2>/dev/null || true",
			memSize, memDuration, memSize, memDuration))

	if err := memCmd.Start(); err != nil {
		errMsg := fmt.Sprintf("failed to start memory stress: %v", err)
		result.Error = errMsg
		result.Success = false
		return fmt.Errorf("%s", errMsg)
	}

	go func() { memCmd.Wait() }()

	startTime := time.Now()
	var lastReport time.Duration

	stressDone := make(chan struct{})
	go func() {
		memCmd.Wait()
		close(stressDone)
	}()

	if IsTextOutput(opts) {
		for time.Since(startTime) < dur {
			select {
			case <-opts.Ctx.Done():
				_ = memCmd.Process.Kill()
				fmt.Printf("\n⚠ Experiment cancelled\n")
				result.Metrics["cancelled"] = true
				break
			case <-stressDone:
				if IsTextOutput(opts) {
					fmt.Printf("   ⚠ Memory stress completed early\n")
				}
			default:
			}

			elapsed := time.Since(startTime)

			if elapsed-lastReport >= 30*time.Second || elapsed >= dur {
				cmd := exec.Command("kubectl", "top", "pod", targetPod.Name, "-n", namespace, "--no-headers")
				output, _ := cmd.CombinedOutput()
				memInfo := "N/A"
				if len(output) > 0 {
					memInfo = string(output)
				}
				fmt.Printf("   📊 [%s elapsed] Memory: %s\n", elapsed.Round(time.Second), memInfo)
				lastReport = elapsed
			}

			time.Sleep(5 * time.Second)
		}

		_ = memCmd.Process.Kill()

		fmt.Printf("   ✅ Memory stress completed\n")

		time.Sleep(2 * time.Second)

		fmt.Printf("\n📈 RESULTS - After Stress:\n")

		statsAfter, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
		if err == nil {
			fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
			fmt.Printf("      Status: %s\n", statsAfter.Phase)
			fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, 0)
			if statsAfter.Restarts > 0 {
				fmt.Printf("      ⚠ Pod was restarted due to OOM!\n")
			}
		}

		cmd := exec.Command("kubectl", "top", "pod", targetPod.Name, "-n", namespace, "--no-headers")
		output, _ := cmd.CombinedOutput()
		if len(output) > 0 {
			fmt.Printf("   📊 Memory after stress: %s\n", string(output))
		}

		result.Metrics["memory_size"] = memSize

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
		fmt.Printf("   Memory hog test completed\n")
	}

	if opts.Output == "json" {
		result.PrintJSON()
	}

	return nil
}
