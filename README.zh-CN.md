# 12306 车票监控 Agent

一个基于 Go 语言开发的监控代理，持续从 12306 中国铁路查询配置路线的车票余票情况，并通过 Prometheus 和 Telegraf 暴露指标数据。

## 功能特性

- **12306 API 集成**: 查询配置出发地和目的地之间的真实列车票务信息
- **Prometheus 指标**: 在 `/metrics` 端点暴露指标数据
- **Telegraf 支持**: 以 InfluxDB 行协议格式输出数据
- **多路线支持**: 可配置多个出发地-目的地组合
- **可配置轮询**: 可调整查询间隔和日期范围
- **多种部署方式**: 二进制、Docker、docker-compose 或 systemd
- **优雅关闭**: 正确处理 SIGINT/SIGTERM 信号

## 快速开始

### 方式一：二进制运行

```bash
# 克隆并构建
git clone <仓库地址>
cd cn-rail-monitor

# 复制配置文件
cp config.yaml.example config.yaml

# 构建
make build

# 运行
./bin/cn-rail-monitor -config config.yaml
```

### 方式二：Docker

```bash
# 构建并运行
make docker-build
make docker-run
```

### 方式三：docker-compose

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

## 配置说明

将 `config.yaml.example` 复制为 `config.yaml` 并根据需要修改：

```yaml
app:
  host: "0.0.0.0"
  port: 8080

query:
  interval: 300          # 轮询间隔（秒）
  days_ahead: 5         # 提前查询天数
  enable_price: false   # 启用票价监控（实验性功能）
  train_types:
    - "G"               # 高铁
    - "D"               # 动车
  routes:
    - name: "北京到上海"
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

### 配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `query.interval` | 轮询间隔（秒） | 300 |
| `query.days_ahead` | 提前查询天数 | 5 |
| `query.start_date` | 指定开始日期 (YYYY-MM-DD) | - |
| `query.end_date` | 指定结束日期 (YYYY-MM-DD) | - |
| `query.enable_price` | 启用票价监控 | false |
| `query.train_types` | 列车类型过滤 (G/D/K/T/Z) | 全部 |
| `app.host` | 服务器绑定地址 | 0.0.0.0 |
| `app.port` | 服务器端口 | 8080 |

### 车站代码

常用 12306 车站代码：
- `BJP` - 北京 (Beijing)
- `SHH` - 上海 (Shanghai)
- `GZQ` - 广州 (Guangzhou)
- `SZP` - 深圳 (Shenzhen)
- `HZH` - 杭州 (Hangzhou)
- `XYY` - 信阳 (Xinyang)

## 部署方式

### 二进制安装

```bash
# 构建
make build

# 安装到系统
sudo make install

# 或安装到用户目录
make install-user
```

### Systemd 服务（用户模式）

```bash
# 安装服务
make install-systemd-user

# 启动服务
make start-systemd-user

# 查看日志
make logs-systemd-user

# 停止服务
make stop-systemd-user
```

### Docker

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run

# 或使用 docker-compose
make docker-compose-up
```

## Prometheus 指标

代理暴露以下指标：

| 指标 | 类型 | 说明 |
|------|------|------|
| `train_ticket_query_total` | Counter | 总查询次数 |
| `train_ticket_query_errors_total` | Counter | 查询失败次数 |
| `train_ticket_available_seats` | Gauge | 各车次/日期/座位类型的余票数 |
| `train_ticket_price` | Gauge | 票价（单位：元，目前为 0） |

采集配置示例：

```yaml
scrape_configs:
  - job_name: 'cn-rail-monitor'
    static_configs:
      - targets: ['localhost:8080']
```

## Telegraf 集成

代理以 InfluxDB 行协议格式输出数据：

```
train_tickets,train_no=G531,train_type=G,from_station=北京南,to_station=上海虹桥,date=2026-02-20,seat_type=硬卧 available=13,price=0.00 1771047516901339760
```

配置 Telegraf 从 stdout 或文件读取：

```toml
# 文件输入
[[inputs.tail]]
  files = ["/var/log/telegraf/train_metrics.log"]
  data_format = "influx"

# 标准输入
[[inputs.stdin]]
```

## HTTP 端点

| 端点 | 说明 |
|------|------|
| `/metrics` | Prometheus 指标 |
| `/health` | 健康检查 |
| `/debug/metrics` | 调试票据数据 |

## Makefile 命令

```bash
make build                  # 构建二进制
make build-linux           # 构建 Linux 版本
make build-darwin          # 构建 macOS 版本
make clean                 # 清理构建产物
make install               # 安装到系统
make test                  # 运行测试
make run                   # 本地运行
make dev                   # 开发模式

# Systemd
make install-systemd-user  # 安装 systemd 服务
make start-systemd-user    # 启动服务
make logs-systemd-user     # 查看日志

# Docker
make docker-build         # 构建 Docker 镜像
make docker-compose-up    # 使用 docker-compose 启动

make help                 # 显示所有命令
```

## 项目结构

```
cn-rail-monitor/
├── cmd/
│   └── main.go              # 应用入口
├── internal/
│   ├── api/                 # 12306 API 客户端
│   ├── config/              # 配置加载
│   ├── metrics/             # Prometheus 指标
│   ├── output/              # Telegraf 输出
│   └── scheduler/           # 轮询调度器
├── systemd/
│   └── cn-rail-monitor.service  # Systemd 服务模板
├── config.yaml.example      # 配置模板
├── Dockerfile               # Docker 镜像
├── docker-compose.yml       # Docker Compose
├── Makefile                # 构建和部署
└── README.md               # 本文件
```

## 许可证

MIT
