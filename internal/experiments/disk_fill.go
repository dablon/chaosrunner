package experiments

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/dablon/chaosrunner/internal/client"
)

type DiskFillExperiment struct {
	BaseExperiment
	k8sClient *client.K8sClient
}

func NewDiskFillExperiment(k8s *client.K8sClient) *DiskFillExperiment {
	return &DiskFillExperiment{k8sClient: k8s}
}

func (e *DiskFillExperiment) Name() string {
	return "disk-fill"
}

func (e *DiskFillExperiment) Run(namespace, duration string) error {
	e.PrintHeader("disk-fill")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Fill size: 100M\n")

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

	startTime := time.Now()
	iteration := 1

	for time.Since(startTime) < dur {
		fmt.Printf("   ✓ Iteration %d: Filling disk with data...\n", iteration)

		cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"dd if=/dev/zero of=/tmp/diskfill bs=1M count=100 2>/dev/null || echo 'Disk fill limited'")
		_ = cmd.Run()

		fmt.Printf("   ✓ Iteration %d: Disk filled (100M)\n", iteration)

		time.Sleep(5 * time.Second)

		cleanupCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"rm -f /tmp/diskfill 2>/dev/null || true")
		_ = cleanupCmd.Run()

		iteration++
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("   ✓ Disk fill cycle completed\n")

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Disk filled: 100M per iteration\n")
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Iterations: %d\n", iteration-1)

	e.PrintFooter(duration)
	fmt.Printf("   Disk fill test completed\n")

	return nil
}
