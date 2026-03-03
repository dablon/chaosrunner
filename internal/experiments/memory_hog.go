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

	fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

	statsBefore, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsBefore.Phase)
		fmt.Printf("      Restarts: %d\n", statsBefore.Restarts)
		fmt.Printf("      Memory Request: %s\n", targetPod.MemoryRequest)
		fmt.Printf("      Memory Limit: %s\n", targetPod.MemoryLimit)
	}

	podsBefore, _ := e.k8sClient.GetPods(namespace)
	fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

	fmt.Printf("\n⚙️  EXECUTION - Starting Memory Stress:\n")

	cmd := exec.Command("kubectl", "top", "pod", targetPod.Name, "-n", namespace, "--no-headers")
	output, _ := cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("   📊 Memory before stress: %s\n", string(output))
	}

	memDuration := int(dur.Seconds())
	if memDuration > 300 {
		memDuration = 300
	}

	fmt.Printf("   🚀 Starting memory stress (256MB, %ds)...\n", memDuration)

	memCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		fmt.Sprintf("stress-ng --vm 2 --vm-bytes 256M --timeout %ds 2>/dev/null || (dd if=/dev/zero of=/tmp/memhog bs=1M count=256 &); sleep %ds; rm -f /tmp/memhog 2>/dev/null || true", memDuration, memDuration))
	memCmd.Run()

	startTime := time.Now()
	var lastReport time.Duration

	for time.Since(startTime) < dur {
		elapsed := time.Since(startTime)

		if elapsed-lastReport >= 30*time.Second || elapsed >= dur {
			cmd := exec.Command("kubectl", "top", "pod", targetPod.Name, "-n", namespace, "--no-headers")
			output, _ := cmd.CombinedOutput()
			memInfo := "N/A"
			if len(output) > 0 {
				memInfo = string(output)
			}
			fmt.Printf("   📊 [%s elapsed] Memory: %s\n", elapsed.Round(time.Second), memInfo)
			lastReport = elapsed
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("   ✅ Memory stress completed\n")

	time.Sleep(2 * time.Second)

	fmt.Printf("\n📈 RESULTS - After Stress:\n")

	statsAfter, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsAfter.Phase)
		fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, statsBefore.Restarts)
		if statsAfter.Restarts > statsBefore.Restarts {
			fmt.Printf("      ⚠ Pod was restarted due to OOM!\n")
		}
	}

	cmd = exec.Command("kubectl", "top", "pod", targetPod.Name, "-n", namespace, "--no-headers")
	output, _ = cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("   📊 Memory after stress: %s\n", string(output))
	}

	podsAfter, _ := e.k8sClient.GetPods(namespace)
	running := 0
	for _, p := range podsAfter {
		if p.Phase == "Running" {
			running++
		}
	}
	fmt.Printf("   ✓ Final state: %d/%d pods running\n", running, len(podsAfter))
	fmt.Printf("   ✓ Duration: %s\n", duration)

	e.PrintFooter(duration)
	fmt.Printf("   Memory hog test completed\n")

	return nil
}
