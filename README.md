# ChaosRunner

Kubernetes chaos engineering tool for injecting faults into containerized applications.

## Installation

```bash
# Linux
curl -L -o chaosrunner https://github.com/dablon/chaosrunner/releases/download/v1.3/chaosrunner-linux-amd64
chmod +x chaosrunner
sudo mv chaosrunner /usr/local/bin/

# macOS
curl -L -o chaosrunner https://github.com/dablon/chaosrunner/releases/download/v1.3/chaosrunner-darwin-amd64
chmod +x chaosrunner
sudo mv chaosrunner /usr/local/bin/

# Windows
# Download from https://github.com/dablon/chaosrunner/releases/tag/v1.3
```

## Usage

```bash
chaosrunner --help
```

## Commands

### list

List available chaos experiments.

```bash
$ chaosrunner list
Available experiments:
   - pod-kill
   - network-latency
   - cpu-stress
   - memory-hog
   - disk-fill
```

### run

Run a chaos experiment.

```bash
$ chaosrunner run pod-kill --namespace production
Running experiment: pod-kill
Namespace: production
```

**Flags:**
- `-n, --namespace` - Target namespace (default: "default")
- `-d, --duration` - Experiment duration (default: "5m")

### version

Print version.

```bash
$ chaosrunner version
1.0.0
```

## Examples

```bash
# List experiments
chaosrunner list

# Run pod-kill in production namespace
chaosrunner run pod-kill -n production

# Run cpu-stress for 10 minutes
chaosrunner run cpu-stress -n staging -d 10m
```

## License

MIT
