package experiments

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dablon/chaosrunner/internal/client"
)

type NetworkLatencyExperiment struct {
	BaseExperiment
	k8sClient *client.K8sClient
}

func NewNetworkLatencyExperiment(k8s *client.K8sClient) *NetworkLatencyExperiment {
	return &NetworkLatencyExperiment{k8sClient: k8s}
}

func (e *NetworkLatencyExperiment) Name() string {
	return "network-latency"
}

func (e *NetworkLatencyExperiment) Run(namespace, duration string, opts *ExperimentOptions) error {
	delay := opts.Delay
	if delay == "" {
		delay = "100ms"
	}

	if IsTextOutput(opts) {
		e.PrintHeader("network-latency", opts)
		fmt.Printf("   Namespace: %s\n", namespace)
		fmt.Printf("   Duration: %s\n", duration)
		fmt.Printf("   Latency: %s\n", delay)
	}

	result := &ExperimentResult{
		Experiment: "network-latency",
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

	cleanup := func() {
		cleanupCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"tc qdisc del dev eth0 root 2>/dev/null || true")
		cleanupCmd.Run()
		if IsTextOutput(opts) {
			fmt.Printf("   ✓ Network latency cleaned up\n")
		}
	}
	defer cleanup()

	if IsTextOutput(opts) {
		fmt.Printf("\n🔍 DIAGNOSIS - Initial State:\n")

		statsBefore, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
		if err == nil {
			fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
			fmt.Printf("      Status: %s\n", statsBefore.Phase)
			fmt.Printf("      Restarts: %d\n", statsBefore.Restarts)
			fmt.Printf("      Age: %s\n", statsBefore.Age)
		}

		podsBefore, _ := e.k8sClient.GetPods(opts.Ctx, namespace, opts.Selector)
		fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

		fmt.Printf("\n⚙️  EXECUTION - Injecting Network Latency:\n")
		fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	}

	checkTcCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c", "which tc")
	checkOutput, checkErr := checkTcCmd.CombinedOutput()
	if checkErr != nil || len(checkOutput) == 0 {
		if IsTextOutput(opts) {
			fmt.Printf("   ℹ tc not found, attempting to install iproute2...\n")
		}

		installCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"apk add --no-cache iproute2 2>/dev/null || apt-get update -qq && apt-get install -y -qq iproute2 2>/dev/null || yum install -y -q iproute2 2>/dev/null || true")
		installOutput, _ := installCmd.CombinedOutput()

		checkTcCmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c", "which tc")
		checkOutput, checkErr = checkTcCmd.CombinedOutput()

		if checkErr != nil || len(checkOutput) == 0 {
			errMsg := "tc command not found in container. Tried to install iproute2 but failed."
			result.Error = errMsg
			result.Success = false
			if IsTextOutput(opts) {
				fmt.Printf("   ✗ %s\n", errMsg)
				fmt.Printf("   ℹ Install iproute2 in your container image or use a different base image\n")
				fmt.Printf("   ℹ Installation output: %s\n", string(installOutput))
			}
			return fmt.Errorf("%s", errMsg)
		}

		if IsTextOutput(opts) {
			fmt.Printf("   ✅ Successfully installed iproute2\n")
		}
	}

	latencyCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		fmt.Sprintf("tc qdisc add dev eth0 root netem delay %s 2>&1 || tc qdisc change dev eth0 root netem delay %s 2>&1 || (ip link show && tc qdisc add dev eth0 root netem delay %s)", delay, delay, delay))
	latencyOutput, latencyErr := latencyCmd.CombinedOutput()

	outputStr := string(latencyOutput)
	if latencyErr != nil || strings.Contains(outputStr, "not found") || strings.Contains(outputStr, "Operation not permitted") || strings.Contains(outputStr, "RTNETLINK answers") {
		errMsg := fmt.Sprintf("failed to apply network latency. Output: %s, Error: %v", outputStr, latencyErr)
		result.Error = errMsg
		result.Success = false
		if IsTextOutput(opts) {
			fmt.Printf("   ✗ %s\n", errMsg)
			if strings.Contains(outputStr, "Operation not permitted") || strings.Contains(outputStr, "permission") {
				fmt.Printf("   ℹ The container needs NET_ADMIN capability or must run as privileged\n")
				fmt.Printf("   ℹ Try: kubectl patch deployment %s -p '{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"*\",\"securityContext\":{\"privileged\":true}}]}}}'\n", targetPod.Name)
			}
		}
		return fmt.Errorf("%s", errMsg)
	}

	if IsTextOutput(opts) {
		fmt.Printf("   ✓ Network latency applied (%s delay)\n", delay)

		cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
			"tc qdisc show dev eth0")
		output, _ := cmd.CombinedOutput()
		fmt.Printf("   📊 Current qdisc config:\n%s\n", string(output))

		fmt.Printf("   ⏳ Maintaining latency for %s...\n", duration)
	}

	startTime := time.Now()
	var lastReport time.Duration

	for time.Since(startTime) < dur {
		select {
		case <-opts.Ctx.Done():
			if IsTextOutput(opts) {
				fmt.Printf("\n⚠ Experiment cancelled\n")
			}
			result.Metrics["cancelled"] = true
			return nil
		default:
		}

		elapsed := time.Since(startTime)

		if elapsed-lastReport >= 30*time.Second || elapsed >= dur {
			if IsTextOutput(opts) {
				cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
					"tc qdisc show dev eth0 | grep delay")
				output, _ := cmd.CombinedOutput()
				latencyInfo := "active"
				if len(output) > 0 {
					latencyInfo = string(output)
				}
				fmt.Printf("   📊 [%s elapsed] Network latency: %s\n", elapsed.Round(time.Second), latencyInfo)
			}
			lastReport = elapsed
		}

		time.Sleep(5 * time.Second)
	}

	if IsTextOutput(opts) {
		fmt.Printf("\n📈 RESULTS - After Latency Test:\n")

		fmt.Printf("   🚀 Cleaning up network latency...\n")

		statsAfter, err := e.k8sClient.GetPodStats(opts.Ctx, namespace, targetPod.Name)
		if err == nil {
			fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
			fmt.Printf("      Status: %s\n", statsAfter.Phase)
			fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, 0)
			if statsAfter.Restarts > 0 {
				fmt.Printf("      ⚠ Pod was restarted!\n")
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
		fmt.Printf("   ✓ Duration: %s\n", duration)

		e.PrintFooter(duration, opts)
		fmt.Printf("   Network latency test completed\n")
	}

	result.Metrics["delay_applied"] = delay

	if opts.Output == "json" {
		result.PrintJSON()
	}

	return nil
}
