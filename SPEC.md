# 12306 Train Ticket Monitor Agent

## 1. Project Overview

- **Project Name**: cn-rail-monitor
- **Type**: Go-based monitoring agent
- **Core Functionality**: Continuously fetch train ticket availability and pricing from 12306 for configured routes, exposing metrics via Prometheus and Telegraf
- **Target Users**: DevOps engineers, data analysts, and anyone needing historical tracking of train ticket prices and availability

## 2. Functionality Specification

### 2.1 Core Features

1. **12306 API Integration**
   - Query train tickets between configured origin and destination
   - Support date range queries (query multiple dates)
   - Parse train types: G (High-Speed), D (Express), K (Fast), etc.
   - Extract seat availability per train
   - Price monitoring (configurable via `enable_price` - currently not available in public API)

2. **Configuration Management**
   - YAML-based configuration file
   - Configure multiple routes (origin → destination pairs)
   - Configure query date range (how many days ahead to query)
   - Configure polling interval
   - Optional price monitoring (`enable_price: false` by default)

2. **Configuration Management**
   - YAML-based configuration file
   - Configure multiple routes (origin → destination pairs)
   - Configure query date range (how many days ahead to query)
   - Configure polling interval

3. **Prometheus Metrics Exporter**
   - HTTP endpoint (`/metrics`) for Prometheus scraping
   - Metrics:
     - `train_ticket_available_seats` - Available seat count per train/date/seat_type
     - `train_ticket_price` - Price per train/date/seat_type
     - `train_ticket_query_total` - Total queries made
     - `train_ticket_query_errors_total` - Query failure count

4. **Telegraf Data Output**
   - Output InfluxDB line protocol to file or stdout
   - Support configurable output mode (file/stdout)
   - Include timestamp, train info, seat type, count, price

5. **Polling & Scheduling**
   - Configurable polling interval (default: 5 minutes)
   - Graceful shutdown
   - Error handling with retry logic

### 2.2 Configuration Structure (config.yaml)

```yaml
app:
  host: "0.0.0.0"
  port: 8080

query:
  # Polling interval in seconds (default: 300 = 5 minutes)
  interval: 300
  
  # Days ahead to query tickets
  days_ahead: 5
  
  # Enable price monitoring (default: false)
  # Note: 12306 public API does not return prices in list query
  enable_price: false
  
  # Train types to filter (optional)
  train_types:
    - "G"  # High-speed
    - "D"  # Express
    
  # Routes to monitor
  routes:
    - name: "Beijing to Shanghai"
      from_station: "BJP"      # Beijing station code
      to_station: "SHH"        # Shanghai station code
      
    - name: "Xinyang to Beijing"
      from_station: "XYY"
      to_station: "BJP"

# Prometheus settings
prometheus:
  enabled: true
  path: "/metrics"

# Telegraf output settings
telegraf:
  enabled: true
  output_mode: "stdout"  # "file" or "stdout"
  output_path: "/var/log/telegraf/train_metrics.log"
  
# Logging
log:
  level: "info"
  file: ""
```

### 2.3 API Details

12306 API Implementation:
- Init URL: `https://kyfw.12306.cn/otn/leftTicket/init` (for cookie)
- Query URL: `https://kyfw.12306.cn/otn/leftTicket/query` → redirects to `queryG`
- Parameters: `leftTicketDTO.train_date`, `leftTicketDTO.from_station`, `leftTicketDTO.to_station`, `purpose_codes`
- Authentication: Cookie-based (JSESSIONID, route)
- Response format: JSON with base64-encoded + pipe-delimited train data

**Note**: 12306 does not provide prices in the public list query API. Prices are only available after selecting a specific train (requires login). The `enable_price` config option is a placeholder for future implementation.

### 2.4 Data Model

```go
// TrainInfo represents a train
type TrainInfo struct {
    TrainNo       string    // Train number (e.g., G1)
    TrainType     string    // G/D/K/etc
    FromStation   string    // Station name
    ToStation     string    // Station name
    DepartureTime string    // Departure time
    ArrivalTime   string    // Arrival time
    Duration      string    // Duration
}

// TicketInfo represents seat availability
type TicketInfo struct {
    TrainInfo
    Date         string    // Travel date
    SeatType     string    // Seat type name
    Price        float64   // Price in CNY
    Available    int       // Available seats (0 = sold out)
    Status       string    // "有" (available), "无" (sold out), or number
}
```

## 3. Technical Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    cn-rail-monitor                      │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────┐ │
│  │   Config    │  │   Scheduler │  │  12306 Client  │ │
│  │   Loader    │  │   (Polling) │  │    (HTTP)      │ │
│  └─────────────┘  └─────────────┘  └────────────────┘ │
│         │                │                  │          │
│         └────────────────┼──────────────────┘          │
│                          ▼                              │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Metrics Collector                  │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│         ┌────────────────┼────────────────┐            │
│         ▼                ▼                ▼            │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────┐ │
│  │  Prometheus │  │   Telegraf │  │    Logger      │ │
│  │  Exporter   │  │   Output   │  │                │ │
│  └─────────────┘  └─────────────┘  └────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## 4. Acceptance Criteria

1. **Configuration Loading**: Agent successfully loads config from YAML file
2. **API Query**: Successfully fetches train ticket data from 12306
3. **Prometheus Metrics**: `/metrics` endpoint returns valid Prometheus format
4. **Telegraf Output**: Outputs valid InfluxDB line protocol
5. **Multiple Routes**: Supports querying multiple origin-destination pairs
6. **Error Handling**: Handles network errors gracefully without crashing
7. **Graceful Shutdown**: Handles SIGINT/SIGTERM properly

## 5. Dependencies

- `github.com/prometheus/client_golang/prometheus` - Prometheus metrics
- `github.com/prometheus/client_golang/prometheus/promhttp` - HTTP handler
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/go-resty/resty/v2` - HTTP client
