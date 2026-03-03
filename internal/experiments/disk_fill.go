package experiments

import (
	"fmt"
	"os/exec"
	"strconv"
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

func (e *DiskFillExperiment) Run(namespace, duration string, opts *ExperimentOptions) error {
	diskSize := opts.DiskSize
	if diskSize == "" {
		diskSize = "100M"
	}

	if IsTextOutput(opts) {
		e.PrintHeader("disk-fill", opts)
		fmt.Printf("   Namespace: %s\n", namespace)
		fmt.Printf("   Duration: %s\n", duration)
		fmt.Printf("   Fill size: %s per iteration\n", diskSize)
	}

	result := &ExperimentResult{
		Experiment: "disk-fill",
		Namespace:  namespace,
		Duration:   duration,
		Success:    true,
		Metrics:    make(map[string]interface{}),
	}

	dur, err := ParseDuration(duration)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return err
	}

	targetPod, err := e.k8sClient.GetRunningPod(opts.Ctx, namespace, opts.Selector)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return err
	}

	createdFiles := make([]string, 0)
	cleanupFiles := func() {
		for _, f := range createdFiles {
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
				fmt.Sprintf("rm -f %s 2>/dev/null || true", f))
			cmd.Run()
		}
		if IsTextOutput(opts) {
			fmt.Printf("   ✓ Cleaned up %d temporary files\n", len(createdFiles))
		}
	}
	defer cleanupFiles()

	if IsTextOutput(opts) {
		fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

		statsBefore, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
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

		podsBefore, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

		fmt.Printf("\n⚙️  EXECUTION - Filling Disk:\n")
	}

	startTime := time.Now()
	iteration := 1
	var lastReport time.Duration

	for time.Since(startTime) < dur {
		select {
		case <-opts.Ctx.Done():
			if IsTextOutput(opts) {
				fmt.Printf("\n⚠ Experiment cancelled\n")
			}
			result.Metrics["cancelled"] = true
			result.Metrics["iterations_completed"] = iteration - 1
			return nil
		default:
		}

		elapsed := time.Since(startTime)

		if elapsed-lastReport >= 30*time.Second || elapsed >= dur {
			if IsTextOutput(opts) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
					"df -h / | tail -1 | awk '{print $5}'")
				diskUsage, _ := cmd.CombinedOutput()
				fmt.Printf("   📊 [%s elapsed] Iteration %d - Disk usage: %s", elapsed.Round(time.Second), iteration, string(diskUsage))
			}
			lastReport = elapsed
		}

		filename := fmt.Sprintf("/tmp/diskfill_%d", iteration)
		createdFiles = append(createdFiles, filename)

		diskSizeNum := diskSize
		if diskSizeNum == "" {
			diskSizeNum = "100M"
		}

		if IsTextOutput(opts) {
			fmt.Printf("   ✓ Iteration %d: Filling disk (%s)...\n", iteration, diskSizeNum)
		}

		cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			fmt.Sprintf("dd if=/dev/zero of=%s bs=1M count=%s 2>/dev/null || echo 'limited'", filename, diskSizeNum))
		_ = cmd.Run()

		cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			fmt.Sprintf("ls -lh %s 2>/dev/null || echo 'file not found'", filename))
		verify, _ := cmd.CombinedOutput()
		if len(verify) > 0 && string(verify) != "file not found\n" {
			if IsTextOutput(opts) {
				fmt.Printf("      ✓ File created: %s\n", string(verify))
			}
		}

		iteration++
		time.Sleep(2 * time.Second)
	}

	if IsTextOutput(opts) {
		fmt.Printf("   ✅ Disk fill cycles completed\n")

		fmt.Printf("\n📈 RESULTS - After Disk Fill Test:\n")

		cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"df -h / | tail -1")
		diskAfter, _ := cmd.CombinedOutput()
		fmt.Printf("   📊 Disk after: %s\n", string(diskAfter))

		statsAfter, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
		if err == nil {
			fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
			fmt.Printf("      Status: %s\n", statsAfter.Phase)
			fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, 0)
			if statsAfter.Restarts > 0 {
				fmt.Printf("      ⚠ Pod was restarted due to disk pressure!\n")
			}
		}

		podsAfter, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		running := 0
		for _, p := range podsAfter {
			if p.Phase == "Running" {
				running++
			}
		}
		fmt.Printf("   ✓ Final state: %d/%d pods running\n", running, len(podsAfter))
		fmt.Printf("   ✓ Total iterations: %d\n", iteration-1)
		fmt.Printf("   ✓ Duration: %s\n", duration)

		e.PrintFooter(duration, opts)
		fmt.Printf("   Disk fill test completed\n")
	}

	result.Metrics["iterations"] = iteration - 1
	result.Metrics["disk_size"] = diskSize
	result.Metrics["files_created"] = len(createdFiles)

	diskUsageBefore, _ := strconv.Atoi(string(diskSize[:len(diskSize)-1]))
	diskUsageAfter := diskUsageBefore * (iteration - 1)
	result.Metrics["approx_disk_written"] = fmt.Sprintf("%dM", diskUsageAfter)

	if opts.Output == "json" {
		result.PrintJSON()
	}

	return nil
}
