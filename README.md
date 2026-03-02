# ChaosRunner

Kubernetes chaos engineering tool for injecting faults into containerized applications.

## Overview

ChaosRunner helps you test system resilience by injecting various types of failures into your Kubernetes clusters. It's designed for chaos engineering experiments and disaster recovery testing.

## Why ChaosRunner?

- **Multiple Failure Types**: Pod kill, network latency, CPU stress, memory hog, disk fill
- **Safe Defaults**: All experiments run in a controlled manner
- **Namespace Isolation**: Target specific namespaces
- **Duration Control**: Set experiment duration
- **Rollback**: Automatic recovery after experiments

## Quick Start

### Install

```bash
# Binary
curl -L -o chaosrunner https://github.com/dablon/chaosrunner/releases/latest/download/chaosrunner-linux-amd64
chmod +x chaosrunner
sudo mv chaosrunner /usr/local/bin/

# Docker
docker pull dablon/chaosrunner:latest
```

### Usage

```bash
# List available experiments
chaosrunner list

# Run pod-kill experiment
chaosrunner run pod-kill --namespace production

# Run network latency experiment  
chaosrunner run network-latency --namespace staging --delay 500ms
```

## Available Experiments

| Experiment | Description | Impact |
|------------|-------------|--------|
| `pod-kill` | Randomly terminate pods | Medium |
| `network-latency` | Inject network delay | Low |
| `cpu-stress` | Stress CPU to 100% | High |
| `memory-hog` | Consume available memory | High |
| `disk-fill` | Fill disk to 95% | Critical |

## Commands

### list

List all available chaos experiments.

```bash
chaosrunner list
```

### run

Run a chaos experiment.

```bash
chaosrunner run <experiment> [flags]
```

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace` | Target namespace | default |
| `--duration` | Experiment duration | 5m |
| `--delay` | Network delay (for latency) | 100ms |

## Examples

### Kill Random Pods

```bash
# Kill random pods in production namespace
chaosrunner run pod-kill --namespace production
```

### Inject Network Latency

```bash
# Inject 500ms latency for 10 minutes
chaosrunner run network-latency --namespace staging --delay 500ms --duration 10m
```

### CPU Stress Test

```bash
# Stress CPU for 5 minutes
chaosrunner run cpu-stress --namespace default --duration 5m
```

## Safety

⚠️ **Warning**: These experiments can disrupt your services. Always:

1. Notify team before running experiments
2. Run during maintenance windows
3. Have rollback plans ready
4. Monitor system during experiments
5. Start with low-impact experiments

## License

MIT
