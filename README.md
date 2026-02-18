## OKEX-BUDDY

### Local Development

### Prerequisites

- Go 1.20+
- Redis 7.x (running on localhost:6379)


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
   redis-cli SADD trading_pairs:active ETH-USDT-SWAP DOGE-USDT-SWAP
   
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
go run .
```

## 四大核心算法详解

### 1. ComputeSupportResistance（支撑阻力位计算）

**算法原理**：基于订单簿中的价格-数量聚集特征，识别具有显著买卖压力的价格水平。

**技术实现**：
1. **价格区间划分**：将订单簿价格范围划分为固定宽度的区间（建议50-100个区间）
2. **累计订单量计算**：分别计算买单和卖单在各区间的加权金额（价格×数量）
3. **显著性识别**：使用局部极大值检测识别支撑/阻力位，阈值为平均累计量的1.5-2.0倍
4. **结果排序**：按累计量降序排序，返回Top-N（建议3-5个）支撑位和阻力位

**应用场景**：
- 技术分析，识别价格可能反转的关键水平
- 为自动化交易策略提供关键价位参考
- 监控市场重要支撑/阻力位的突破情况

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

**输出示例**：
```json
{
  "supports": [45000.0, 44500.0],
  "resistances": [46000.0, 46500.0],
  "spread": 1000.0
}
```

### 2. ComputeLargeOrderDistribution（大额订单分布分析）

**算法原理**：通过识别大额订单（whale orders）的分布，推断机构或大户的交易意图，并使用非线性变换模型计算更准确的市场情绪指标。

**技术实现**：
1. **大额订单阈值确定**：使用订单金额分位数动态确定阈值（建议90-95分位数）
2. **大额订单识别**：识别金额超过阈值的订单
3. **价格距离加权**：越接近当前价格的订单权重越高，权重公式：$w(p) = e^{-\lambda \cdot \frac{|p - P_{\text{mid}}|}{P_{\text{mid}}}}$
4. **多空力量对比**：计算加权后的买卖金额，得出原始多空倾向指标
5. **非线性情绪变换**：使用带有死区阈值的非线性变换模型，更准确反映市场情绪强度

**应用场景**：
- 监控大额资金流向，分析市场主力动向
- 为机构投资者提供大户交易意图分析
- 辅助判断市场多空情绪和潜在趋势转换

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

**输出示例**：
```json
{
  "bullPower": 123456.78,
  "bearPower": 78901.23,
  "sentiment": 0.22,
  "interpretation": "轻微看多"
}
```

**情绪指标解读**：
- `sentiment > 0.3`：强烈看涨信号
- `0.1 < sentiment ≤ 0.3`：温和看涨信号
- `-0.1 ≤ sentiment ≤ 0.1`：中性市场
- `-0.3 ≤ sentiment < -0.1`：温和看跌信号
- `sentiment < -0.3`：强烈看跌信号

### 3. DetectDepthAnomaly（深度异常检测）

**算法原理**：使用时间窗口统计和Z-score检测订单簿深度的突变，识别可能预示市场重大变动的订单簿结构变化。

**技术实现**：
1. **深度指标定义**：计算某一价格范围内（如当前价格的±0.5%）的总订单量
2. **滑动窗口统计**：计算过去W个时间点的均值和标准差
3. **异常检测（Z-score）**：$Z(t) = \frac{D(t, r) - \mu_D}{\sigma_D}$，当$|Z(t)| > Z_{\text{threshold}}$时触发异常
4. **波动方向与强度**：根据Z值的正负判断深度增加或减少，强度由$|Z(t)|$决定

**应用场景**：
- 预警订单簿结构突变，可能预示市场即将发生重大变动
- 为高频交易提供市场微观结构变化的早期信号
- 监控市场流动性突然变化的风险事件

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

**输出示例**：
```json
{
  "anomaly": true,
  "z_score": -13.7423,
  "direction": "decrease",
  "intensity": 13.7423,
  "depth": 12345.67
}
```

**日志输出示例**：
```text
Depth Anomaly Detected for DOGE-USDT-SWAP: Z-Score=-13.7423, Direction=decrease, Intensity=13.7423
- Z值为-13.7423：这表示当前监控价格区间内的深度比历史平均值低了约13.74个标准差
- 方向=下跌：负的Z值明确指示订单簿深度相较历史平均水平急剧下降
- 强度=13.7423：这个数值就是Z值的绝对值，代表了信号强度
注意：系统刚启动时历史数据不足可能导致假阳性警报
```

### 4. DetectLiquidityShrinkage（流动性收缩预警）

**算法原理**：综合评估订单簿的深度、价差和时间趋势，检测流动性恶化，为交易者提供滑点风险管理预警。

**技术实现**：
1. **流动性指标定义**：
   - 有效价差：$\text{Spread}(t) = \frac{P_{\text{ask}}(t) - P_{\text{bid}}(t)}{P_{\text{mid}}(t)}$
   - 近价深度：距离中间价Δ范围内的订单总量
   - 综合流动性指标：$L(t) = \frac{\text{Depth}(t, \Delta)}{1 + \text{Spread}(t)}$
2. **趋势检测**：在短期窗口内使用线性回归计算流动性指标的趋势斜率
3. **萎缩判定**：同时满足以下三个条件时触发预警：
   - 流动性绝对水平低（低于长期25分位数）
   - 流动性呈下降趋势（斜率为负且超过阈值）
   - 价差扩大（高于历史75分位数）
4. **预警分级**：根据满足条件数量分为轻度、中度、严重三级预警

**应用场景**：
- 预警市场流动性风险，帮助交易者管理滑点风险
- 为算法交易提供流动性环境变化的实时反馈
- 监控市场微观结构恶化的早期信号

**参数配置**：
| 参数 | 类型 | 描述 | 推荐值 | 调整建议 |
|------|------|------|--------|----------|
| nearPriceDeltaPercent | float64 | 价格附近的百分比阈值 | 0.5 | 高流动性：0.1-0.5<br>低流动性：0.5-1.5 |
| shortWindowSeconds | int | 短期趋势窗口（秒） | 30 | 快速响应：15-30<br>平滑波动：30-60 |
| longWindowSeconds | int | 长期基准窗口（秒） | 1800 | 短期交易：900-1800<br>长期分析：1800-3600 |
| slopeThreshold | float64 | 流动性下降斜率阈值 | -0.01 | 敏感检测：-0.005<br>稳定检测：-0.01 |

**示例用法**：
```go
// 检测DOGE-USDT-SWAP的流动性收缩情况
liquidityShrinkData, err := obManager.DetectLiquidityShrinkage("DOGE-USDT-SWAP", 0.5, 30, 1800, -0.01)
```

**输出示例**：
```json
{
  "warning": true,
  "warning_level": "severe",
  "liquidity": 27717.2699,
  "spread": 0.0015,
  "depth": 123456.78,
  "slope": -1.552002
}
```

**日志输出示例**：
```text
Liquidity Shrinkage Warning for BTC-USDT-SWAP: Level=severe, Liquidity=27717.2699, Slope=-1.552002
严重负趋势：触发此预警需要3个条件满足且斜率达到严重程度
3个判定条件：
- Low absolute liquidity：当前流动性低于长期25分位数
- Negative trend：短期流动性呈负趋势（斜率 < -0.01）
- High spread：当前价差高于历史75分位数
Slope值解读：Slope = -82.74意味着流动性正在"高速下滑"，表明市场深度正在经历剧烈恶化
代码中当Slope < -20才会触发严重级别预警
```

## 算法协同使用策略

### 1. 高频套利策略
```go
// 快速响应市场微观结构变化
// 紧密监控支撑阻力位突破和深度异常
supports, resistances, spread, err := obManager.ComputeSupportResistance("BTC-USDT-SWAP", 30, 1.2, 1, 0.3)
depthAnomaly, err := obManager.DetectDepthAnomaly("BTC-USDT-SWAP", 0.3, 15, 1.8)

// 结合流动性预警避免滑点风险
liquidityData, err := obManager.DetectLiquidityShrinkage("BTC-USDT-SWAP", 0.2, 20, 900, -0.005)

// 策略逻辑：
// if depthAnomaly.Anomaly && depthAnomaly.Direction == "increase" && 
//    !liquidityData.Warning {
//     // 深度增加且无流动性风险时执行套利
// }
```

### 2. 趋势跟踪策略
```go
// 识别稳定的市场趋势和关键价位
supports, resistances, spread, err := obManager.ComputeSupportResistance("ETH-USDT-SWAP", 60, 1.8, 3, 0.8)
largeBuy, largeSell, sentiment, err := obManager.ComputeLargeOrderDistribution("ETH-USDT-SWAP", 0.90, 7.0, 0.2)

// 长期流动性监控
liquidityData, err := obManager.DetectLiquidityShrinkage("ETH-USDT-SWAP", 0.8, 60, 3600, -0.01)

// 策略逻辑：
// if sentiment > 0.3 && !liquidityData.Warning && 
//    price > highest_support && price < lowest_resistance {
//     // 强势多头情绪 + 良好流动性 + 价格在合理区间时建立多头仓位
// }
```

### 3. 机构资金流向分析
```go
// 深度分析大额订单分布和市场情绪
largeBuyNotional, largeSellNotional, sentiment, err := obManager.ComputeLargeOrderDistribution("SOL-USDT-SWAP", 0.95, 5.0, 0.3)
supports, resistances, spread, err := obManager.ComputeSupportResistance("SOL-USDT-SWAP", 40, 1.5, 2, 0.5)

// 监控大单活动引起的价格波动
depthAnomaly, err := obManager.DetectDepthAnomaly("SOL-USDT-SWAP", 1.0, 40, 2.2)

// 策略逻辑：
// if math.Abs(sentiment) > 0.4 && depthAnomaly.Anomaly {
//     // 明显的机构情绪 + 深度异常 = 重要的资金流向信号
//     // 可结合支撑阻力位制定跟随策略
// }
```

### 4. 风险管理策略
```go
// 综合风险监控体系
liquidityData, err := obManager.DetectLiquidityShrinkage("DOGE-USDT-SWAP", 0.5, 30, 1800, -0.01)
depthAnomaly, err := obManager.DetectDepthAnomaly("DOGE-USDT-SWAP", 0.5, 30, 2.5)

// 大额订单监控潜在风险
largeBuy, largeSell, sentiment, err := obManager.ComputeLargeOrderDistribution("DOGE-USDT-SWAP", 0.98, 8.0, 0.4)

// 风险控制逻辑：
// if liquidityData.WarningLevel == "severe" || 
//    (depthAnomaly.Anomaly && depthAnomaly.Intensity > 5.0) ||
//    math.Abs(sentiment) > 0.6 {
//     // 触发任一高级别风险信号时减少仓位或暂停交易
// }
```

## 性能优化建议

### 1. 计算资源分配
- **高频交易**：优先保证深度异常检测和支撑阻力位计算的实时性
- **趋势分析**：重点优化流动性收缩预警和大额订单分析
- **批量处理**：可将部分非紧急计算安排在低峰时段执行

### 2. 参数动态调整
```go
// 根据市场波动性动态调整参数
func adjustParameters(marketVolatility float64) (binCount int, zThreshold float64, slopeThreshold float64) {
    if marketVolatility > 0.02 { // 高波动
        return 25, 2.5, -0.005  // 更敏感的检测
    } else if marketVolatility < 0.005 { // 低波动
        return 75, 1.8, -0.015  // 更稳定的检测
    }
    return 50, 2.0, -0.01     // 默认参数
}
```

### 3. 内存管理优化
- 使用时间窗口工具类自动管理过期数据
- 合理设置各算法的历史数据窗口大小
- 定期清理无用的历史计算结果

建议根据具体的交易场景、市场条件和个人风险偏好，灵活组合使用这四个核心算法，以构建最适合的量化交易策略。
