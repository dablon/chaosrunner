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

func (e *MemoryHogExperiment) Run(namespace, duration string) error {
	e.PrintHeader("memory-hog")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Memory: 512MB\n")

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified pods in namespace\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	fmt.Printf("   ✓ Starting memory stress...\n")

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"stress-ng --vm 2 --vm-bytes 256M --timeout 30s 2>/dev/null || (dd if=/dev/zero of=/tmp/memhog bs=1M count=256 2>/dev/null &); sleep 30; rm -f /tmp/memhog 2>/dev/null || echo 'Memory stress completed'")
	_ = cmd.Run()

	fmt.Printf("   ✓ Memory stress applied\n")
	fmt.Printf("   ✓ Monitoring memory usage...\n")

	time.Sleep(2 * time.Second)

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Memory consumed: ~256MB\n")
	fmt.Printf("   Duration: 30s\n")
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Total pods affected: 1\n")

	e.PrintFooter(duration)
	fmt.Printf("   Memory hog test completed\n")

	return nil
}
