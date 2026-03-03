package experiments

import (
	"fmt"
	"time"
)

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

func ParseDuration(d string) (time.Duration, error) {
	d = "0" + d
	duration, err := time.ParseDuration(d)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format '%s'. Use format like 5m, 30s, 1h", d)
	}
	return duration, nil
}
