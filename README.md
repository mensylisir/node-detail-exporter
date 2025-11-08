# Node Prober

Node Prober is a monitoring tool designed to collect a variety of system and network metrics from a host and expose them in a format that can be scraped by Prometheus.

## Features

- **System Metrics:** Collects statistics on CPU, memory, I/O, and per-process resource usage.
- **Network Probing:** Performs network diagnostics like ping and MTR to measure network latency and packet loss.
- **Kubernetes Auditing:** Can tail Kubernetes API server audit logs to provide insights into cluster activity.
- **Extensible:** Designed with a collector-based architecture that is easy to extend.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Docker (for containerized deployment)
- `mtr` command-line tool (if using the MTR collector)

### Configuration

Node Prober is configured via a YAML file. An example configuration file is provided at `configs/config.example.yaml`.

To enable or disable a collector, simply set its corresponding key to `true` or `false` in the `collectors` section of the configuration file.

### Building and Running from Source

You can build and run Node Prober from source using the provided `Makefile`:

```bash
# Build the binary
make build

# Run the prober with the default configuration
./node-prober --config.file=configs/config.example.yaml
```

### Building and Running as a Docker Container

A `Dockerfile` is provided for building and running Node Prober in a container:

```bash
# Build the Docker image
make docker-build

# Run the container (assuming the image was tagged with version 'latest')
docker run -p 8080:8080 -v $(pwd)/configs:/configs node-prober:latest --config.file=/configs/config.example.yaml
```

## Exposed Metrics

Node Prober exposes metrics on port `8080` at the `/metrics` endpoint. These metrics are designed to be scraped by a Prometheus server.
