# MicroOrchestrator - Lightweight Container Orchestration System

A container orchestration system built in Go, demonstrating distributed systems concepts, reliability patterns, and observability best practices.

## Features

- **Multi-Region Orchestration**: Simulates deployment across multiple regions with latency awareness
- **Intelligent Scheduling**: Multiple strategies (binpack, spread) with resource-aware placement
- **Health Monitoring**: HTTP health checks with circuit breaker pattern
- **Self-Healing**: Automatic container restarts with exponential backoff
- **gRPC API**: High-performance API for container management
- **Observability**: Prometheus metrics and Grafana dashboards
- **99.99% Success Rate Tracking**: Real-time deployment success monitoring

## Architecture

```
Control Plane (Go)
├── Scheduler (Binpack/Spread strategies)
├── Health Monitor (Circuit breakers)
├── gRPC API Server
└── Metrics Collector (Prometheus)
Node Agents (Simulated)
├── Region: us-east-1
├── Region: eu-west-1
└── Region: ap-southeast-1
```

## Tech Stack

- **Language**: Go 1.24+
- **RPC Framework**: gRPC with Protocol Buffers
- **Observability**: Prometheus + Grafana
- **Container Runtime**: Docker (simulated)

## Installation & Development

```bash
# Clone the repository
git clone https://github.com/yourusername/micro-orchestrator.git
cd micro-orchestrator

# Install dependencies
go mod download

# Generate protobuf code
make proto

# Build the orchestrator
make build
```


## Demo

```bash
# start the servers and deploy some test containers
make demo

# Orchestrator API: http://localhost:50051
# Metrics: http://localhost:8080/metrics
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000

# deploy more containers
docker-compose exec orchestrator /app/cli \
                -server=localhost:50051 \
                -action=deploy \
                -name=demo-app \
                -image=nginx:latest \
                -replicas=2 \
                -cpu=2 \
                -memory=1024
```


Built with ❤️ for the Koyeb engineering team
