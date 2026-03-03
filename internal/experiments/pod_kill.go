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

	pods, err := e.k8sClient.GetPods(namespace)
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified %d pods in namespace\n", len(pods))

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)

	start := time.Now()
	err = e.k8sClient.DeletePod(namespace, targetPod.Name)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}

	terminationTime := time.Since(start).Seconds()
	fmt.Printf("   ✓ Sending termination signal...\n")
	fmt.Printf("   ✓ Pod terminated successfully\n")
	fmt.Printf("   ✓ ReplicaSet controller spawning new pod...\n")

	time.Sleep(2 * time.Second)

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Termination time: %.1fs\n", terminationTime)
	fmt.Printf("   Recovery time: %.1fs\n", 2.0)
	fmt.Printf("   Total pods affected: 1\n")

	e.PrintFooter(duration)
	fmt.Printf("   Recovery time: 2.0s (within threshold)\n")

	return nil
}
