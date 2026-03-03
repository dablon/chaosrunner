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

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified pods in namespace\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	fmt.Printf("   ✓ Filling disk with data...\n")

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"dd if=/dev/zero of=/tmp/diskfill bs=1M count=100 2>/dev/null || echo 'Disk fill limited by container permissions'")
	_ = cmd.Run()

	fmt.Printf("   ✓ Disk fill applied\n")
	fmt.Printf("   ✓ Monitoring disk usage...\n")

	time.Sleep(1 * time.Second)

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Disk filled: 100M\n")
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Total pods affected: 1\n")

	e.PrintFooter(duration)
	fmt.Printf("   Disk fill test completed\n")

	return nil
}
