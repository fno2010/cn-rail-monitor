# 12306 Train Ticket Monitor Agent

A Go-based monitoring agent that continuously fetches train ticket availability from 12306 (China Railway) for configured routes, exposing metrics via Prometheus and Telegraf.

## Features

- **12306 API Integration**: Queries real train tickets between configured origin and destination
- **Prometheus Metrics**: Exposes metrics at `/metrics` endpoint
- **Telegraf Support**: Outputs data in InfluxDB line protocol format
- **Multiple Routes**: Configure multiple origin-destination pairs
- **Configurable Polling**: Adjustable query interval and date range
- **Multiple Deployment Options**: Binary, Docker, docker-compose, or systemd
- **Graceful Shutdown**: Handles SIGINT/SIGTERM properly

## Quick Start

### Option 1: Binary

```bash
# Clone and build
git clone <repository-url>
cd cn-rail-monitor

# Copy configuration
cp config.yaml.example config.yaml

# Build
make build

# Run
./bin/cn-rail-monitor -config config.yaml
```

### Option 2: Docker

```bash
# Build and run
make docker-build
make docker-run
```

### Option 3: docker-compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f
```

## Configuration

Copy `config.yaml.example` to `config.yaml` and modify as needed:

```yaml
app:
  host: "0.0.0.0"
  port: 8080

query:
  interval: 300          # Polling interval in seconds
  days_ahead: 5         # Days ahead to query
  enable_price: false   # Enable price monitoring (experimental)
  train_types:
    - "G"               # High-speed
    - "D"               # Express
  routes:
    - name: "Beijing to Shanghai"
      from_station: "BJP"
      to_station: "SHH"

prometheus:
  enabled: true
  path: "/metrics"

telegraf:
  enabled: true
  output_mode: "stdout"
  output_path: "/var/log/telegraf/train_metrics.log"

log:
  level: "info"
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `query.interval` | Polling interval in seconds | 300 |
| `query.days_ahead` | Days ahead to query | 5 |
| `query.start_date` | Specific start date (YYYY-MM-DD) | - |
| `query.end_date` | Specific end date (YYYY-MM-DD) | - |
| `query.enable_price` | Enable price monitoring | false |
| `query.train_types` | Train type filter (G/D/K/T/Z) | all |
| `app.host` | Server binding address | 0.0.0.0 |
| `app.port` | Server port | 8080 |

### Station Codes

Common 12306 station codes:
- `BJP` - Beijing (北京)
- `SHH` - Shanghai (上海)
- `GZQ` - Guangzhou (广州)
- `SZP` - Shenzhen (深圳)
- `HZH` - Hangzhou (杭州)
- `XYY` - Xinyang (信阳)

## Deployment

### Binary Installation

```bash
# Build
make build

# Install to system
sudo make install

# Or install to user directory
make install-user
```

### Systemd Service (User Mode)

```bash
# Install service
make install-systemd-user

# Start service
make start-systemd-user

# View logs
make logs-systemd-user

# Stop service
make stop-systemd-user
```

### Docker

```bash
# Build image
make docker-build

# Run container
make docker-run

# Or use docker-compose
make docker-compose-up
```

## Prometheus Metrics

The agent exposes the following metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `train_ticket_query_total` | Counter | Total queries made |
| `train_ticket_query_errors_total` | Counter | Query failure count |
| `train_ticket_available_seats` | Gauge | Available seats per train/date/seat_type |
| `train_ticket_price` | Gauge | Price in CNY (currently 0) |

Example scrape config:

```yaml
scrape_configs:
  - job_name: 'cn-rail-monitor'
    static_configs:
      - targets: ['localhost:8080']
```

## Telegraf Integration

The agent outputs data in InfluxDB line protocol:

```
train_tickets,train_no=G531,train_type=G,from_station=北京南,to_station=上海虹桥,date=2026-02-20,seat_type=硬卧 available=13,price=0.00 1771047516901339760
```

Configure Telegraf to read from stdout or file:

```toml
# File input
[[inputs.tail]]
  files = ["/var/log/telegraf/train_metrics.log"]
  data_format = "influx"

# Stdin input
[[inputs.stdin]]
```

## HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `/metrics` | Prometheus metrics |
| `/health` | Health check |
| `/debug/metrics` | Debug ticket data |

## Makefile Commands

```bash
make build                  # Build binary
make build-linux           # Build for Linux
make build-darwin          # Build for macOS
make clean                 # Clean build artifacts
make install               # Install to system
make test                  # Run tests
make run                   # Run locally
make dev                   # Development mode

# Systemd
make install-systemd-user  # Install systemd service
make start-systemd-user    # Start service
make logs-systemd-user     # View logs

# Docker
make docker-build         # Build Docker image
make docker-compose-up     # Start with docker-compose

make help                 # Show all commands
```

## Project Structure

```
cn-rail-monitor/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── api/                 # 12306 API client
│   ├── config/              # Configuration loading
│   ├── metrics/             # Prometheus metrics
│   ├── output/              # Telegraf output
│   └── scheduler/           # Polling scheduler
├── systemd/
│   └── cn-rail-monitor.service  # Systemd service template
├── config.yaml.example      # Configuration template
├── Dockerfile               # Docker image
├── docker-compose.yml       # Docker compose
├── Makefile                # Build & deployment
└── README.md               # This file
```

## License

MIT
