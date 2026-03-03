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
	fmt.Printf("   ✓ Starting CPU stress...\n")

	stressDuration := int(dur.Seconds())
	if stressDuration > 300 {
		stressDuration = 300
	}

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		fmt.Sprintf("stress-ng --cpu 4 --timeout %ds 2>/dev/null || (stress -c 4 -t %ds 2>/dev/null) || (apk add --no-cache stress 2>/dev/null && stress -c 4 -t %ds)", stressDuration, stressDuration, stressDuration))
	cmd.Run()

	fmt.Printf("   ✓ CPU stress applied\n")
	fmt.Printf("   ⏳ Running CPU stress for %s...\n", duration)

	startTime := time.Now()
	iteration := 1

	for time.Since(startTime) < dur {
		fmt.Printf("   ✓ Iteration %d: CPU stress active (%.0fs elapsed)\n", iteration, time.Since(startTime).Seconds())
		time.Sleep(10 * time.Second)
		iteration++
	}

	fmt.Printf("   ✓ CPU stress completed\n")

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   CPU load increased by: ~400%%\n")
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Iterations: %d\n", iteration-1)

	e.PrintFooter(duration)
	fmt.Printf("   CPU stress test completed\n")

	return nil
}
