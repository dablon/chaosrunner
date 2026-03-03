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

func (e *PodKillExperiment) Run(namespace, duration string) error {
	e.PrintHeader("pod-kill")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Target: Random pods in namespace\n")

	dur, err := ParseDuration(duration)
	if err != nil {
		return err
	}

	fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

	initialPods, err := e.k8sClient.GetPods(namespace)
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	fmt.Printf("   ✓ Found %d pods in namespace:\n", len(initialPods))
	for _, p := range initialPods {
		fmt.Printf("      - %s [%s] Restarts: %d\n", p.Name, p.Phase, p.Restarts)
	}

	if len(initialPods) > 0 {
		stats, _ := e.k8sClient.GetPodStats(namespace, initialPods[0].Name)
		if stats != nil {
			fmt.Printf("      📊 Resources - CPU: %s, Memory: %s\n",
				initialPods[0].CPURequest, initialPods[0].MemoryRequest)
		}
	}

	fmt.Printf("\n⚙️  EXECUTION - Starting Chaos:\n")

	startTime := time.Now()
	iteration := 1
	totalKillTime := 0.0
	totalRecoveryTime := 0.0

	for time.Since(startTime) < dur {
		elapsed := time.Since(startTime)

		pods, err := e.k8sClient.GetPods(namespace)
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}

		running := 0
		for _, p := range pods {
			if p.Phase == "Running" {
				running++
			}
		}

		fmt.Printf("\n   📊 [%s] Iteration %d: %d/%d pods running\n",
			elapsed.Round(time.Second), iteration, running, len(pods))

		targetPod, err := e.k8sClient.GetRunningPod(namespace)
		if err != nil {
			fmt.Printf("   ⚠ No running pods found, waiting...\n")
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("   💀 Killing pod: %s\n", targetPod.Name)

		killStart := time.Now()
		err = e.k8sClient.DeletePod(namespace, targetPod.Name)
		if err != nil {
			fmt.Printf("   ⚠ Failed to delete pod: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		killTime := time.Since(killStart).Seconds()
		totalKillTime += killTime

		fmt.Printf("   ⏱  Kill time: %.2fs\n", killTime)
		fmt.Printf("   ⏳ Waiting for new pod to be Ready...")

		recoveryStart := time.Now()

		err = e.k8sClient.WaitForPodReady(namespace, targetPod.Name, 60*time.Second)
		recoveryTime := time.Since(recoveryStart).Seconds()
		totalRecoveryTime += recoveryTime

		if err != nil {
			fmt.Printf(" FAILED\n")
			fmt.Printf("   ⚠ Recovery timeout: %.2fs\n", recoveryTime)
		} else {
			fmt.Printf(" OK (%.2fs)\n", recoveryTime)
		}

		iteration++
		time.Sleep(2 * time.Second)
	}

	avgKillTime := totalKillTime / float64(iteration-1)
	avgRecoveryTime := totalRecoveryTime / float64(iteration-1)

	fmt.Printf("\n📈 RESULTS - Final Metrics:\n")
	fmt.Printf("   ✅ Total iterations: %d\n", iteration-1)
	fmt.Printf("   💀 Total pods killed: %d\n", iteration-1)
	fmt.Printf("   ⏱  Avg kill time: %.2fs\n", avgKillTime)
	fmt.Printf("   ⏳ Avg recovery time: %.2fs\n", avgRecoveryTime)
	fmt.Printf("   📊 Total duration: %s\n", duration)

	finalPods, _ := e.k8sClient.GetPods(namespace)
	runningFinal := 0
	for _, p := range finalPods {
		if p.Phase == "Running" {
			runningFinal++
		}
	}
	fmt.Printf("   ✓ Final state: %d/%d pods running\n", runningFinal, len(finalPods))

	e.PrintFooter(duration)
	fmt.Printf("   Recovery time: %.2fs (avg)\n", avgRecoveryTime)

	return nil
}
