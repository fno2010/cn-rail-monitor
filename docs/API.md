# 12306 API 数据获取流程文档

## 概述

本文档详细解释 cn-rail-monitor 如何从 12306官网获取火车票余票数据。

## 整体流程

```
1. 获取 Cookie
   |
   v
2. 查询车次列表 (leftTicket/queryG)
   |
   v
3. 处理响应数据
   |
   v
4. 解析余票信息
```

## 站点代码缓存机制

### 为什么需要缓存

12306 官网提供了 3000+ 个火车站站点，每个站点都有唯一的 3 位字母代码（如北京=BJP，上海=SHH）。每次查询都需要将用户友好的站点名称转换为 12306 API 要求的代码。

### 自动获取

程序在启动时会自动从 12306 官方获取最新的站点代码：

**数据源**: `https://kyfw.12306.cn/otn/resources/js/framework/station_name.js`

**请求示例**:
```bash
curl -s "https://kyfw.12306.cn/otn/resources/js/framework/station_name.js"
```

**响应格式**:
```javascript
var station_names = '@bjb|北京北|VAP|beijingbei|bjb|0|...@bji北京|BJP|beijing|bj|2|...'
```

### 缓存策略

| 阶段 | 行为 |
|------|------|
| 程序启动 | 尝试读取缓存文件 |
| 缓存不存在 | 自动从网络获取 |
| 网络失败 | 使用内置备用站点列表 |
| 缓存存在 | 直接加载（秒级启动） |

### 默认缓存路径

程序按以下优先级查找缓存文件：

1. **配置指定**: `config.yaml` 中 `station.cache_path` 配置的路径
2. **可执行文件目录**: 与 `cn-rail-monitor` 二进制文件同目录的 `station_codes.json`
3. **当前工作目录**: 运行程序时所在目录的 `station_codes.json`

> **注意**: 如果使用 systemd 服务运行，默认路径是二进制文件所在目录（通常是 `~/.local/bin/station_codes.json`）。建议显式配置 `station.cache_path` 以确保缓存文件位置可控。

### 配置缓存路径

```json
{
  "codes": [
    {
      "code": "BJP",
      "name": "北京",
      "pinyin": "beijing",
      "short_name": "bj"
    },
    ...
  ],
  "updated": "2026-02-16T10:06:58+08:00"
}
```

### 配置缓存路径

在 `config.yaml` 中添加：

```yaml
station:
  cache_path: "/path/to/custom/station_codes.json"
```

### 手动刷新缓存

可以在程序运行时删除缓存文件，下次启动时会自动重新获取最新站点数据。

### 备用站点列表

如果无法从网络获取站点代码，程序内置了常用站点作为备用：

| 代码 | 城市 |
|------|------|
| BJP | 北京 |
| SHH | 上海 |
| GZQ | 广州 |
| SZP | 深圳 |
| HZH | 杭州 |
| CDW | 成都 |
| WHH | 武汉 |
| XAY | 西安 |
| NJH | 南京 |
| TJP | 天津 |
| CQW | 重庆 |
| XUN | 信阳 |

---

## 详细步骤

### 1. 获取 Cookie

12306 API 需要有效的 Cookie 才能访问。程序在启动时和每次查询前都会刷新 Cookie。

**API 端点**: `GET https://kyfw.12306.cn/otn/

**请求示例**:
```bash
curl -v "https://kyfw.12306.cn/otn/" \
  -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
```

**响应**: 返回 Set-Cookie 头，包含:
- `route`: 路由标识
- `JSESSIONID`: 会话 ID
- `BIGipServerotn`: 负载均衡标识

**代码位置**: `internal/api/client.go` - `refreshCookie()` 函数

---

### 2. 查询车次列表

使用获取的 Cookie 查询指定日期和路线的车次信息。

**API 端点**: `GET https://kyfw.12306.cn/otn/leftTicket/queryG`

**请求参数**:
| 参数 | 说明 | 示例 |
|------|------|------|
| leftTicketDTO.train_date | 出发日期 (YYYY-MM-DD) | 2026-02-25 |
| leftTicketDTO.from_station | 出发站代码 (3位字母) | XUN (信阳) |
| leftTicketDTO.to_station | 到达站代码 (3位字母) | BJP (北京) |
| purpose_codes | 票种 (ADULT=成人) | ADULT |

**完整请求示例**:
```bash
curl -s -L "https://kyfw.12306.cn/otn/leftTicket/queryG?\
leftTicketDTO.train_date=2026-02-25&\
leftTicketDTO.from_station=XUN&\
leftTicketDTO.to_station=BJP&\
purpose_codes=ADULT" \
  -H "Cookie: route=xxx; JSESSIONID=xxx; BIGipServerotn=xxx" \
  -H "User-Agent: Mozilla/5.0 ..."
```

**响应格式** (JSON):
```json
{
  "httpstatus": 200,
  "data": {
    "flag": "1",
    "map": {
      "BJP": "北京",
      "XUN": "信阳"
    },
    "result": [
      "url_encoded_string|预订|车次ID|车次号|..."
    ]
  },
  "status": true
}
```

---

### 3. 处理重定向

12306 API 可能会返回 HTTP 302 重定向。程序使用 Go HTTP 客户端的默认重定向处理，自动跟随重定向并保留 Cookie。

**处理逻辑**:
- 客户端默认自动跟随 302/301 重定向
- 自动维护请求头中的 Cookie
- 最终到达最终页面获取数据

---

### 4. 解析余票数据

响应中的 `result` 字段包含 URL 编码的列车数据，需要解码后解析。

**数据格式** (以 `|` 分割):
```
[0] secretStr     - 加密字符串
[1] buttonText    - 按钮文字 ("预订")
[2] trainNo       - 内部车次ID
[3] stationTrainCode - 车次号 (如 G672)
[4] fromStationTelecode - 出发站电报码
[5] toStationTelecode   - 到达站电报码
[6] fromStationName    - 出发站名称
[7] toStationName      - 到达站名称
[8] startTime         - 出发时间 (HH:MM)
[9] arriveTime        - 到达时间 (HH:MM)
[10] duration         - 历时
[11] dayDifference    - 天数差
[12] - 座位信息编码
[13] 出发日期
[14] 座位数量
[15-28] 各座位类型信息
```

**座位类型编码**:
| 编码 | 座位类型 |
|------|----------|
| M | 一等座 |
| O | 二等座 |
| WZ | 无座 |
| YZ | 硬座 |
| YW | 硬卧 |
| RW | 软卧 |
| SR | 商务座 |
| TZ | 特等座 |

**座位信息格式** (第15-28位):
```
座位类型代码|剩余数量|票价|...
```

---

## 站点代码

### 获取站点代码

程序会自动从 12306 官方获取最新的站点代码，并缓存到本地文件。

**站点数据源**: `https://kyfw.12306.cn/otn/resources/js/framework/station_name.js`

**缓存文件**: `station_codes.json` (与可执行文件同目录)

**缓存更新**:
- 程序启动时自动加载缓存
- 如果缓存文件不存在，自动从网络获取
- 可以调用 `RefreshStationCodes()` 手动更新

### 常用站点代码

| 代码 | 城市 |
|------|------|
| BJP | 北京 |
| SHH | 上海 |
| GZQ | 广州 |
| SZP | 深圳 |
| HZH | 杭州 |
| CDW | 成都 |
| WHH | 武汉 |
| XAY | 西安 |
| NJH | 南京 |
| TJP | 天津 |
| CQW | 重庆 |
| XUN | 信阳 |

---

## 错误处理

### 常见错误

1. **HTTP 302/301**: 正常重定向，由 HTTP 客户端自动处理

2. **HTTP 404**: 路线不存在或日期不可售

3. **HTTP 403**: Cookie 无效或过期，需要刷新

4. **空结果**: API 返回 `result: []`，可能是:
   - 站点代码错误
   - 日期无票
   - API 限流

### 容错机制

当 API 调用失败时，程序会:
1. 记录错误日志
2. 使用模拟数据作为后备
3. 继续查询其他路线

---

## API 调用示例

### 完整请求流程 (Python)

```python
import requests
import urllib.parse

def query_tickets(from_station, to_station, date):
    base_url = "https://kyfw.12306.cn/otn"
    
    # Step 1: Get Cookie
    session = requests.Session()
    session.get(f"{base_url}/")
    cookies = session.cookies.get_dict()
    
    # Step 2: Query trains
    url = f"{base_url}/leftTicket/queryG?" \
          f"leftTicketDTO.train_date={date}&" \
          f"leftTicketDTO.from_station={from_station}&" \
          f"leftTicketDTO.to_station={to_station}&" \
          f"purpose_codes=ADULT"
    
    response = session.get(url)
    data = response.json()
    
    # Step 3: Parse results
    for item in data['data'].get('result', []):
        parts = urllib.parse.unquote(item).split('|')
        train_no = parts[3]  # 车次号
        # ... 解析座位信息
```

---

## 参考资料

- 12306 官网: https://www.12306.cn/
- 车站代码查询: https://www.12306.cn/mormhweb/zxdy/index.html
