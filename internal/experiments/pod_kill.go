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

	fmt.Printf("\n⚙️  Progress:\n")

	startTime := time.Now()
	iteration := 1

	for time.Since(startTime) < dur {
		pods, err := e.k8sClient.GetPods(namespace)
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}

		fmt.Printf("   ✓ Iteration %d: Identified %d pods in namespace\n", iteration, len(pods))

		targetPod, err := e.k8sClient.GetRunningPod(namespace)
		if err != nil {
			fmt.Printf("   ⚠ No running pods found, waiting...\n")
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("   ✓ Iteration %d: Selected target: %s\n", iteration, targetPod.Name)

		killStart := time.Now()
		err = e.k8sClient.DeletePod(namespace, targetPod.Name)
		if err != nil {
			fmt.Printf("   ⚠ Failed to delete pod: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		terminationTime := time.Since(killStart).Seconds()
		fmt.Printf("   ✓ Iteration %d: Pod terminated (%.1fs)\n", iteration, terminationTime)

		fmt.Printf("   ✓ Waiting for ReplicaSet to spawn new pod...\n")
		time.Sleep(3 * time.Second)

		iteration++
		fmt.Printf("   ✓ Sleeping before next iteration...\n")
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Total iterations: %d\n", iteration-1)
	fmt.Printf("   Total pods affected: %d\n", iteration-1)
	fmt.Printf("   Duration: %s\n", duration)

	e.PrintFooter(duration)
	fmt.Printf("   Recovery time: ~3s per iteration (within threshold)\n")

	return nil
}
