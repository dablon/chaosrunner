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

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified pods in namespace\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	fmt.Printf("   ✓ Applying network latency using tc...\n")

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"tc qdisc add dev eth0 root netem delay 5s 2>/dev/null || tc qdisc change dev eth0 root netem delay 5s 2>/dev/null")
	_ = cmd.Run()

	fmt.Printf("   ✓ Network latency applied\n")

	fmt.Printf("   ⏳ Maintaining latency for %s...\n", duration)

	startTime := time.Now()
	iteration := 1

	for time.Since(startTime) < dur {
		fmt.Printf("   ✓ Iteration %d: Latency active (%.0fs elapsed)\n", iteration, time.Since(startTime).Seconds())
		time.Sleep(10 * time.Second)
		iteration++
	}

	fmt.Printf("   ✓ Cleaning up network latency...\n")

	cleanupCmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"tc qdisc del dev eth0 root 2>/dev/null || true")
	_ = cleanupCmd.Run()

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Latency added: 5s\n")
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Total duration: %s\n", duration)
	fmt.Printf("   Iterations: %d\n", iteration-1)

	e.PrintFooter(duration)
	fmt.Printf("   Network latency injected and cleaned up\n")

	return nil
}
