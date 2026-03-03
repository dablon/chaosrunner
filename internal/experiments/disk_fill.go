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

	fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

	statsBefore, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsBefore.Phase)
		fmt.Printf("      Restarts: %d\n", statsBefore.Restarts)
		fmt.Printf("      Age: %s\n", statsBefore.Age)
	}

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"df -h / | tail -1")
	diskBefore, _ := cmd.CombinedOutput()
	fmt.Printf("   📊 Disk before: %s\n", string(diskBefore))

	podsBefore, _ := e.k8sClient.GetPods(namespace)
	fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

	fmt.Printf("\n⚙️  EXECUTION - Filling Disk:\n")

	startTime := time.Now()
	iteration := 1
	var lastReport time.Duration

	for time.Since(startTime) < dur {
		elapsed := time.Since(startTime)

		if elapsed-lastReport >= 30*time.Second || elapsed >= dur {
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
				"df -h / | tail -1 | awk '{print $5}'")
			diskUsage, _ := cmd.CombinedOutput()
			fmt.Printf("   📊 [%s elapsed] Iteration %d - Disk usage: %s", elapsed.Round(time.Second), iteration, string(diskUsage))
			lastReport = elapsed
		}

		fmt.Printf("   ✓ Iteration %d: Filling disk (100M)...\n", iteration)

		cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"dd if=/dev/zero of=/tmp/diskfill bs=1M count=100 2>/dev/null || echo 'limited'")
		_ = cmd.Run()

		cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"ls -lh /tmp/diskfill 2>/dev/null || echo 'file not found'")
		verify, _ := cmd.CombinedOutput()
		if len(verify) > 0 && string(verify) != "file not found\n" {
			fmt.Printf("      ✓ File created: %s\n", string(verify))
		}

		time.Sleep(2 * time.Second)

		cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"rm -f /tmp/diskfill 2>/dev/null || true")
		_ = cmd.Run()

		iteration++
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("   ✅ Disk fill cycles completed\n")

	fmt.Printf("\n📈 RESULTS - After Disk Fill Test:\n")

	cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"df -h / | tail -1")
	diskAfter, _ := cmd.CombinedOutput()
	fmt.Printf("   📊 Disk after: %s\n", string(diskAfter))

	statsAfter, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsAfter.Phase)
		fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, statsBefore.Restarts)
		if statsAfter.Restarts > statsBefore.Restarts {
			fmt.Printf("      ⚠ Pod was restarted due to disk pressure!\n")
		}
	}

	podsAfter, _ := e.k8sClient.GetPods(namespace)
	running := 0
	for _, p := range podsAfter {
		if p.Phase == "Running" {
			running++
		}
	}
	fmt.Printf("   ✓ Final state: %d/%d pods running\n", running, len(podsAfter))
	fmt.Printf("   ✓ Total iterations: %d\n", iteration-1)
	fmt.Printf("   ✓ Duration: %s\n", duration)

	e.PrintFooter(duration)
	fmt.Printf("   Disk fill test completed\n")

	return nil
}
