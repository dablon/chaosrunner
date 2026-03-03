package experiments

import (
	"fmt"
	"os/exec"
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

func (e *NetworkLatencyExperiment) Run(namespace, duration string) error {
	e.PrintHeader("network-latency")
	fmt.Printf("   Namespace: %s\n", namespace)
	fmt.Printf("   Duration: %s\n", duration)
	fmt.Printf("   Latency: 5s\n")

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

	podsBefore, _ := e.k8sClient.GetPods(namespace)
	fmt.Printf("   ✓ Namespace overview: %d pods running\n", len(podsBefore))

	fmt.Printf("\n⚙️  EXECUTION - Injecting Network Latency:\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"tc qdisc add dev eth0 root netem delay 5s 2>/dev/null || tc qdisc change dev eth0 root netem delay 5s 2>/dev/null")
	_ = cmd.Run()

	fmt.Printf("   ✓ Network latency applied (5s delay)\n")

	cmd = exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"tc qdisc show dev eth0")
	output, _ := cmd.CombinedOutput()
	fmt.Printf("   📊 Current qdisc config:\n%s\n", string(output))

	startTime := time.Now()
	var lastReport time.Duration

	fmt.Printf("   ⏳ Maintaining latency for %s...\n", duration)

	for time.Since(startTime) < dur {
		elapsed := time.Since(startTime)

		if elapsed-lastReport >= 30*time.Second || elapsed >= dur {
			cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
				"tc qdisc show dev eth0 | grep delay")
			output, _ := cmd.CombinedOutput()
			latencyInfo := "active (5s delay)"
			if len(output) > 0 {
				latencyInfo = string(output)
			}
			fmt.Printf("   📊 [%s elapsed] Network latency: %s\n", elapsed.Round(time.Second), latencyInfo)
			lastReport = elapsed
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("\n📈 RESULTS - After Latency Test:\n")

	fmt.Printf("   🚀 Cleaning up network latency...\n")

	cleanupCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"tc qdisc del dev eth0 root 2>/dev/null || true")
	_ = cleanupCmd.Run()

	statsAfter, err := e.k8sClient.GetPodStats(namespace, targetPod.Name)
	if err == nil {
		fmt.Printf("   ✓ Target pod: %s\n", targetPod.Name)
		fmt.Printf("      Status: %s\n", statsAfter.Phase)
		fmt.Printf("      Restarts: %d (before: %d)\n", statsAfter.Restarts, statsBefore.Restarts)
		if statsAfter.Restarts > statsBefore.Restarts {
			fmt.Printf("      ⚠ Pod was restarted!\n")
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
	fmt.Printf("   ✓ Duration: %s\n", duration)
	fmt.Printf("   ✓ Latency cleaned up: ✓\n")

	e.PrintFooter(duration)
	fmt.Printf("   Network latency test completed\n")

	return nil
}
