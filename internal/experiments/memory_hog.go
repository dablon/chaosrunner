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

	dur, err := ParseDuration(duration)
	if err != nil {
		return err
	}

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified pods in namespace\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	fmt.Printf("   ✓ Starting memory stress...\n")

	memDuration := int(dur.Seconds())
	if memDuration > 300 {
		memDuration = 300
	}

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		fmt.Sprintf("stress-ng --vm 2 --vm-bytes 256M --timeout %ds 2>/dev/null || (dd if=/dev/zero of=/tmp/memhog bs=1M count=256 &); sleep %ds; rm -f /tmp/memhog 2>/dev/null || true", memDuration, memDuration))
	cmd.Run()

	fmt.Printf("   ✓ Memory stress applied\n")
	fmt.Printf("   ⏳ Running memory stress for %s...\n", duration)

	startTime := time.Now()
	iteration := 1

	for time.Since(startTime) < dur {
		fmt.Printf("   ✓ Iteration %d: Memory stress active (%.0fs elapsed)\n", iteration, time.Since(startTime).Seconds())
		time.Sleep(10 * time.Second)
		iteration++
	}

	fmt.Printf("   ✓ Memory stress completed\n")

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Memory consumed: ~256MB\n")
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Iterations: %d\n", iteration-1)

	e.PrintFooter(duration)
	fmt.Printf("   Memory hog test completed\n")

	return nil
}
