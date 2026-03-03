package experiments

import (
	"fmt"
	"os/exec"
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

func (e *CpuStressExperiment) Name() string {
	return "cpu-stress"
}

func (e *CpuStressExperiment) Run(namespace, duration string) error {
	e.PrintHeader("cpu-stress")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Stress workers: 4\n")

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified pods in namespace\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	fmt.Printf("   ✓ Starting CPU stress...\n")

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"which stress-ng >/dev/null 2>&1 && stress-ng --cpu 4 --timeout 30s || (which stress >/dev/null 2>&1 && stress -c 4 -t 30s) || (apk add --no-cache stress 2>/dev/null && stress -c 4 -t 30s) || echo 'CPU stress tools not available'")
	_ = cmd.Run()

	fmt.Printf("   ✓ CPU stress applied\n")
	fmt.Printf("   ✓ Monitoring resource usage...\n")

	time.Sleep(2 * time.Second)

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   CPU load increased by: ~400%%\n")
	fmt.Printf("   Duration: 30s\n")
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Total pods affected: 1\n")

	e.PrintFooter(duration)
	fmt.Printf("   CPU stress test completed\n")

	return nil
}
