## okex-buddy

### Local Development

### Prerequisites

- Go 1.20+
- Redis 6.x (running on localhost:6379)
- Python 3.9+ (for Bytewax, later milestones)


### M2 - WebSocket Client Setup

#### Proxy Configuration

The WebSocket client supports SOCKS5 proxy for local development:

- **Development (local)**: Enable proxy in `config/app.dev.env`
  ```bash
  USE_PROXY=true
  PROXY_ADDR=127.0.0.1:4781
  ```

- **Production (Hong Kong server)**: Disable proxy in `config/app.prod.env`
  ```bash
  USE_PROXY=false
  PROXY_ADDR=
  ```

The client will automatically use SOCKS5 proxy when `USE_PROXY=true` is set.

#### Setup Steps

1. **Start Redis** (required for M2)
   ```bash
   redis-server
   ```

2. **Configure trading pairs in Redis**
   ```bash
   # Add trading pairs to monitor (use SWAP contracts for real-time data)
   redis-cli SADD trading_pairs:active BTC-USDT-SWAP ETH-USDT-SWAP DOGE-USDT-SWAP
   
   # Verify configuration
   redis-cli SMEMBERS trading_pairs:active
   ```

3. **Start WebSocket Client**
   ```bash
   cd /Users/anthony/Documents/github/okex-buddy
   
   # Load environment variables
   export $(grep -v '^#' config/app.dev.env | xargs)
   
   # Run WebSocket client
   cd backend/go
   go run ./cmd/websocket_client
   ```

   **Expected output:**
   ```
   2026/01/01 10:10:55 Subscription confirmed: BTC-USDT-SWAP
   2026/01/01 10:10:55 Subscription confirmed: ETH-USDT-SWAP
   ```

4. **Verify subscription success**
   
   After the WebSocket client starts and subscriptions are confirmed, verify that order book data is being received:
   
   ```bash
   # Check order book snapshot (first 25 lines show metadata and first few price levels)
   redis-cli -h localhost -p 6379 HGETALL orderbook:BTC-USDT-SWAP | head -25
   
   # Expected output includes:
   # - instrument_id: BTC-USDT-SWAP
   # - checksum: <int32 value>
   # - asks: [array of ask price levels]
   # - bids: [array of bid price levels]
   # - timestamp: <unix timestamp>
   
   # Check real-time event stream length
   redis-cli LLEN list:orderbook:events
   # Should show increasing numbers as updates arrive
   ```
   
   If you see order book data with valid checksums and increasing event counts, the subscription is working correctly!

5. **Monitor in real-time**
   ```bash
   # Watch order book updates in Redis
   redis-cli MONITOR
   
   # Check order book snapshot for a specific pair
   redis-cli HGETALL orderbook:BTC-USDT-SWAP
   

   
   # View event stream
   redis-cli LRANGE list:orderbook:events 0 10
   ```

6. **Test dynamic subscription**
   ```bash
   # Add a new trading pair (will be subscribed in ~20 seconds)
   redis-cli SADD trading_pairs:active SOL-USDT-SWAP
   
   # Remove a trading pair (will be unsubscribed in ~20 seconds)
   redis-cli SREM trading_pairs:active ETH-USDT-SWAP
   ```

### Original Dev Setup (M1)

For reference, the original M1 setup instructions: (dev)

#### 1. Prerequisites
- **Golang**: 1.20+
- **Python**: 3.9+ (for Bytewax / analysis flow)
- **Redis**: 6.x (running locally, e.g. `localhost:6379`)

#### 2. Environment configuration

From the project root (`/Users/anthony/Documents/github/okex-buddy`), load dev environment variables:

```bash
cd /Users/anthony/Documents/github/okex-buddy

# App-level dev config (Redis, OKEx WS, API bind, etc.)
export $(grep -v '^#' config/app.dev.env | xargs)


```

> Keep real secrets (passwords, tokens) in local `.env` or your shell, do not commit them.

#### 3. Start backend services

- **WebSocket client service (Go)**

```bash
cd /Users/anthony/Documents/github/okex-buddy/backend/go
go run ./cmd/websocket_client
```

- **API / monitoring backend service (Go)**

```bash
cd /Users/anthony/Documents/github/okex-buddy/backend/go
go run ./cmd/api_server
```

Both services automatically load configuration from environment variables via the unified `internal/config` module. Make sure you've exported the env vars (step 2) before running.

#### 4. Start analysis flow (Python / Bytewax)

Current skeleton (before wiring Bytewax `Dataflow`) can be run with:

```bash
cd /Users/anthony/Documents/github/okex-buddy
python analysis/bytewax/analysis_flow.py
```

Once the Bytewax flow is implemented, you can also use:

```bash
cd /Users/anthony/Documents/github/okex-buddy/analysis/bytewax
bytewax run analysis_flow.py
```



#### 5. Process overview in dev
- **Go WebSocket client**: connects to OKEx WS, processes order book data, writes into Redis.
- **Python / Bytewax analysis**: consumes buffered data (via Redis List), computes deeper analytics, writes results to Redis.
- **Go API server**: exposes REST endpoints for current state and metrics.

## API参数配置指南

### 1. ComputeSupportResistance

**功能**：计算订单簿支撑位和阻力位
**分析角度**：价格区间交易量分布
**应用场景**：技术分析，识别价格可能反转的关键水平

**参数配置**：
| 参数 | 类型 | 描述 | 推荐值 | 调整建议 |
|------|------|------|--------|----------|
| binCount | int | 价格区间划分数量 | 50 | 高流动性交易对：30-50<br>低流动性交易对：20-30 |
| significanceThreshold | float64 | 支撑/阻力位显著性阈值 | 1.5 | 市场波动大时：1.2-1.5<br>市场稳定时：1.5-2.0 |
| topN | int | 返回的支撑/阻力位数量 | 2 | 短期交易：1-2<br>中长期分析：3-5 |
| minDistancePercent | float64 | 支撑/阻力位之间的最小价格差异百分比 | 0.5 | 高波动市场：0.3-0.5<br>低波动市场：0.5-1.0 |

**示例用法**：
```go
// 计算BTC-USDT-SWAP的支撑和阻力位
supports, resistances, err := obManager.ComputeSupportResistance("BTC-USDT-SWAP", 50, 1.5, 2, 0.5)
```

### 2. ComputeLargeOrderDistribution

**功能**：分析大额订单分布和市场情绪
**分析角度**：订单规模和价格距离加权分布
**应用场景**：监控大额资金流向，分析市场多空情绪

**参数配置**：
| 参数 | 类型 | 描述 | 推荐值 | 调整建议 |
|------|------|------|--------|----------|
| percentileAlpha | float64 | 大额订单的百分位数阈值 | 0.95 | 活跃市场：0.90-0.95<br>清淡市场：0.95-0.98 |
| decayLambda | float64 | 价格距离衰减因子 | 5.0 | 高流动性：3.0-5.0<br>低流动性：5.0-8.0 |
| sentimentDeadzoneThreshold | float64 | 情绪中性区间阈值 | 0.3 | 低波动市场：0.2-0.3<br>高波动市场：0.3-0.5 |

**示例用法**：
```go
// 分析ETH-USDT-SWAP的大额订单分布
largeBuyNotional, largeSellNotional, sentiment, err := obManager.ComputeLargeOrderDistribution("ETH-USDT-SWAP", 0.95, 5.0, 0.3)
```

### 3. DetectDepthAnomaly

**功能**：检测订单簿深度的异常变化
**分析角度**：深度变化的统计显著性
**应用场景**：预警订单簿结构突变，可能预示市场即将发生重大变动

**参数配置**：
| 参数 | 类型 | 描述 | 推荐值 | 调整建议 |
|------|------|------|--------|----------|
| priceRangePercent | float64 | 计算深度的价格范围百分比 | 0.5 | 高流动性：0.1-0.5<br>中流动性：0.5-1.0<br>低流动性：1.0-3.0 |
| windowSize | int | 历史数据窗口大小 | 30 | 高频交易：15-30<br>趋势跟踪：30-60 |
| zThreshold | float64 | Z分数异常阈值 | 2.0 | 保守策略：2.5-3.0<br>平衡策略：2.0-2.5<br>激进策略：1.5-2.0 |

**示例用法**：
```go
// 检测SOL-USDT-SWAP的深度异常
depthAnomaly, err := obManager.DetectDepthAnomaly("SOL-USDT-SWAP", 0.5, 30, 2.0)
```

### 4. DetectLiquidityShrinkage

**功能**：检测流动性收缩情况
**分析角度**：流动性的时间趋势变化
**应用场景**：预警市场流动性风险，帮助交易者管理滑点风险

**参数配置**：
| 参数 | 类型 | 描述 | 推荐值 | 调整建议 |
|------|------|------|--------|----------|
| instID | string | 交易对ID | - | - |
| nearPriceDeltaPercent | float64 | 价格附近的百分比阈值 | 0.5 | 高流动性：0.1-0.5<br>低流动性：0.5-1.5 |
| shortWindowSeconds | int | 短期趋势窗口（秒） | 30 | 快速响应：15-30<br>平滑波动：30-60 |
| longWindowSeconds | int | 长期基准窗口（秒） | 1800 | 短期交易：900-1800<br>长期分析：1800-3600 |
| slopeThreshold | float64 | 流动性下降斜率阈值 | -0.01 | 敏感检测：-0.005<br>稳定检测：-0.01 |

**示例用法**：
```go
// 检测DOGE-USDT-SWAP的流动性收缩情况
liquidityShrinkData, err := obManager.DetectLiquidityShrinkage("DOGE-USDT-SWAP", 0.5, 30, 1800, -0.01)
```

## 参数组合优化

### 高频交易策略
```go
// 快速响应市场变化
supports, resistances, err := obManager.ComputeSupportResistance("BTC-USDT-SWAP", 30, 1.2, 1, 0.3)
depthAnomaly, err := obManager.DetectDepthAnomaly("BTC-USDT-SWAP", 0.3, 20, 1.8)
```

### 趋势跟踪策略
```go
// 识别稳定的市场趋势
supports, resistances, err := obManager.ComputeSupportResistance("ETH-USDT-SWAP", 60, 1.8, 3, 0.8)
liquidityShrinkData, err := obManager.DetectLiquidityShrinkage("ETH-USDT-SWAP", 0.8, 60, 3600, -0.01)
```

### 大额订单分析
```go
// 检测市场大额订单活动
largeBuyNotional, largeSellNotional, sentiment, err := obManager.ComputeLargeOrderDistribution("SOL-USDT-SWAP", 0.95, 5.0, 0.3)
depthAnomaly, err := obManager.DetectDepthAnomaly("SOL-USDT-SWAP", 1.0, 40, 2.2)
```

建议根据市场条件和交易策略动态调整参数，以获得最佳效果。
