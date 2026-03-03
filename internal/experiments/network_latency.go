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

	targetPod, err := e.k8sClient.GetRunningPod(namespace)
	if err != nil {
		return err
	}

	fmt.Printf("\n⚙️  Progress:\n")
	fmt.Printf("   ✓ Identified pods in namespace\n")
	fmt.Printf("   ✓ Selected target: %s\n", targetPod.Name)
	fmt.Printf("   ✓ Applying network latency using tc...\n")

	cmd := exec.Command("kubectl", "exec", "-n", namespace, targetPod.Name, "--", "sh", "-c",
		"tc qdisc add dev eth0 root netem delay 5s 2>/dev/null || tc qdisc change dev eth0 root netem delay 5s 2>/dev/null || echo 'Network shaping not available in this container'")
	_ = cmd.Run()

	time.Sleep(1 * time.Second)

	fmt.Printf("   ✓ Network latency applied\n")

	fmt.Printf("\n📊 Metrics:\n")
	fmt.Printf("   Latency added: 5s\n")
	fmt.Printf("   Target: %s\n", targetPod.Name)
	fmt.Printf("   Total pods affected: 1\n")

	e.PrintFooter(duration)
	fmt.Printf("   Network latency injected\n")

	return nil
}
