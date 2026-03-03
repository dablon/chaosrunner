package experiments

import (
	"fmt"
	"time"

	"github.com/dablon/chaosrunner/internal/client"
)

type PodKillExperiment struct {
	BaseExperiment
	k8sClient *client.K8sClient
}

func NewPodKillExperiment(k8s *client.K8sClient) *PodKillExperiment {
	return &PodKillExperiment{k8sClient: k8s}
}

func (e *PodKillExperiment) Name() string {
	return "pod-kill"
}

func (e *PodKillExperiment) Run(namespace, duration string, opts *ExperimentOptions) error {
	if IsTextOutput(opts) {
		e.PrintHeader("pod-kill", opts)
		fmt.Printf("   Namespace: %s\n", namespace)
		fmt.Printf("   Duration: %s\n", duration)
		if opts.Selector != "" {
			fmt.Printf("   Target: Pods matching selector: %s\n", opts.Selector)
		} else {
			fmt.Printf("   Target: Random pods in namespace\n")
		}
	}

	dur, err := ParseDuration(duration)
	if err != nil {
		return err
	}

	result := &ExperimentResult{
		Experiment: "pod-kill",
		Namespace:  namespace,
		Duration:   duration,
		Success:    true,
		Metrics:    make(map[string]interface{}),
	}

	if IsTextOutput(opts) {
		fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")
	}

	initialPods, err := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
	if err != nil {
		errMsg := fmt.Sprintf("failed to list pods: %v", err)
		result.Error = errMsg
		result.Success = false
		if IsTextOutput(opts) {
			fmt.Printf("   ✗ Failed to list pods: %v\n", err)
		}
		return fmt.Errorf("%s", errMsg)
	}

	if IsTextOutput(opts) {
		fmt.Printf("   ✓ Found %d pods in namespace:\n", len(initialPods))
		for _, p := range initialPods {
			readyStatus := "NotReady"
			if p.Ready {
				readyStatus = "Ready"
			}
			fmt.Printf("      - %s [%s] Restarts: %d\n", p.Name, readyStatus, p.Restarts)
		}

		if len(initialPods) > 0 {
			stats, _ := e.k8sClient.GetPodStats(opts.Ctx, namespace, initialPods[0].Name)
			if stats != nil {
				fmt.Printf("      📊 Resources - CPU: %s, Memory: %s\n",
					initialPods[0].CPURequest, initialPods[0].MemoryRequest)
			}
		}

		fmt.Printf("\n⚙️  EXECUTION - Starting Chaos:\n")
	}

	startTime := time.Now()
	iteration := 1
	var totalKillTime float64
	var totalRecoveryTime float64

	for time.Since(startTime) < dur {
		select {
		case <-opts.Ctx.Done():
			if IsTextOutput(opts) {
				fmt.Printf("\n⚠ Experiment cancelled\n")
			}
			result.Metrics["cancelled"] = true
			result.Metrics["iterations_completed"] = iteration - 1
			break
		default:
		}

		elapsed := time.Since(startTime)

		pods, err := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}

		running := 0
		for _, p := range pods {
			if p.Phase == "Running" {
				running++
			}
		}

		if IsTextOutput(opts) {
			fmt.Printf("\n   📊 [%s] Iteration %d: %d/%d pods running\n",
				elapsed.Round(time.Second), iteration, running, len(pods))
		}

		targetPod, err := e.k8sClient.GetRunningPod(opts.Ctx, namespace, opts.Selector)
		if err != nil {
			if IsTextOutput(opts) {
				fmt.Printf("   ⚠ No running pods found, waiting...\n")
			}
			time.Sleep(5 * time.Second)
			continue
		}

		if IsTextOutput(opts) {
			fmt.Printf("   💀 Killing pod: %s\n", targetPod.Name)
		}

		killStart := time.Now()
		err = e.k8sClient.DeletePod(opts.Ctx, namespace, targetPod.Name)
		if err != nil {
			if IsTextOutput(opts) {
				fmt.Printf("   ⚠ Failed to delete pod: %v\n", err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		killTime := time.Since(killStart).Seconds()
		totalKillTime += killTime

		if IsTextOutput(opts) {
			fmt.Printf("   ⏱  Kill time: %.2fs\n", killTime)
			fmt.Printf("   ⏳ Waiting for new pod to be Ready...")
		}

		recoveryStart := time.Now()

		err = e.k8sClient.WaitForNewPodReady(opts.Ctx, namespace, opts.Selector, targetPod.Name, 60*time.Second)
		recoveryTime := time.Since(recoveryStart).Seconds()
		totalRecoveryTime += recoveryTime

		if err != nil {
			if IsTextOutput(opts) {
				fmt.Printf(" FAILED\n")
				fmt.Printf("   ⚠ Recovery timeout: %.2fs\n", recoveryTime)
			}
		} else {
			if IsTextOutput(opts) {
				fmt.Printf(" OK (%.2fs)\n", recoveryTime)
			}
		}

		iteration++
		time.Sleep(2 * time.Second)
	}

	iterationsCompleted := iteration - 1
	if iterationsCompleted < 1 {
		iterationsCompleted = 1
	}
	avgKillTime := totalKillTime / float64(iterationsCompleted)
	avgRecoveryTime := totalRecoveryTime / float64(iterationsCompleted)

	result.Metrics["iterations"] = iterationsCompleted
	result.Metrics["pods_killed"] = iterationsCompleted
	result.Metrics["avg_kill_time"] = fmt.Sprintf("%.2fs", avgKillTime)
	result.Metrics["avg_recovery_time"] = fmt.Sprintf("%.2fs", avgRecoveryTime)

	if IsTextOutput(opts) {
		fmt.Printf("\n📈 RESULTS - Final Metrics:\n")
		fmt.Printf("   ✅ Total iterations: %d\n", iterationsCompleted)
		fmt.Printf("   💀 Total pods killed: %d\n", iterationsCompleted)
		fmt.Printf("   ⏱  Avg kill time: %.2fs\n", avgKillTime)
		fmt.Printf("   ⏳ Avg recovery time: %.2fs\n", avgRecoveryTime)
		fmt.Printf("   📊 Total duration: %s\n", duration)

		finalPods, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		runningFinal := 0
		for _, p := range finalPods {
			if p.Phase == "Running" {
				runningFinal++
			}
		}
		fmt.Printf("   ✓ Final state: %d/%d pods running\n", runningFinal, len(finalPods))

		e.PrintFooter(duration, opts)
		fmt.Printf("   Recovery time: %.2fs (avg)\n", avgRecoveryTime)
	}

	if opts.Output == "json" {
		result.PrintJSON()
	}

	return nil
}
