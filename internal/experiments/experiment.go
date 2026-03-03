package experiments

import "fmt"

type Experiment interface {
	Name() string
	Run(namespace, duration string) error
}

type BaseExperiment struct{}

func (b *BaseExperiment) PrintHeader(name string) {
	fmt.Println("🔥 Running chaos experiment:", name)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n📋 Experiment: %s\n", name)
}

func (b *BaseExperiment) PrintFooter(duration string) {
	fmt.Printf("\n✅ Experiment completed successfully\n")
	fmt.Printf("   Duration: %s\n", duration)
}
