## OKEX-BUDDY

### Prerequisites

- Go 1.20+
- Redis 7.x (running on localhost:6379)
- MongoDB 6.0.6

### WebSocket Client Setup

#### Proxy Configuration

The WebSocket client supports SOCKS5 proxy for local development:

- **Development (local)**: Enable proxy in `config/app.dev.env`
  ```bash
  USE_PROXY=true
  PROXY_ADDR=127.0.0.1:7890
  ```
- **Production (Hong Kong server)**: Disable proxy in `config/app.prod.env`
  ```bash
  USE_PROXY=false
  ```

The client will automatically use SOCKS5 proxy when `USE_PROXY=true` is set.

#### WebSocket Integration Details

In the current version, main.go only integrates the WebSocket PrivateClient. BusinessClient and PublicClient are not integrated. If needed, you can refer to the 'assembly' package to integrate them yourself.

<br />

#### BusinessClient

**Description**
BusinessClient is a WebSocket client for subscribing to business trading pairs on OKEx exchange. Its main function is to dynamically subscribe to candlestick data for trading pairs based on Redis configuration, while storing candlestick data to MongoDB.

#### PrivateClient

**Description**
PrivateClient is a WebSocket client for subscribing to private trading pairs on OKEx exchange. Its main function is to monitor Position change events, while storing Position data to MongoDB.

#### PublicClient

**Description**
PublicClient is a WebSocket client for subscribing to public trading pairs on OKEx exchange. Its main function is to dynamically subscribe to order book data for trading pairs based on Redis configuration, while performing analysis and storing results to Redis.

**Configure trading pairs in Redis**

```bash
# Add trading pairs to monitor (use SWAP contracts for real-time data)
redis-cli SADD trading_pairs:active ETH-USDT-SWAP DOGE-USDT-SWAP

# Verify configuration
redis-cli SMEMBERS trading_pairs:active
```

1. **Verify subscription success**

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
2. **Monitor in real-time**
   ```bash
   # Watch order book updates in Redis
   redis-cli MONITOR

   # Check order book snapshot for a specific pair
   redis-cli HGETALL orderbook:BTC-USDT-SWAP

   # View event stream
   redis-cli LRANGE list:orderbook:events 0 10
   ```
3. **Test dynamic subscription**
   ```bash
   # Add a new trading pair (will be subscribed in ~20 seconds)
   redis-cli SADD trading_pairs:active SOL-USDT-SWAP

   # Remove a trading pair (will be unsubscribed in ~20 seconds)
   redis-cli SREM trading_pairs:active ETH-USDT-SWAP
   ```

##### Detailed Explanation of Four Core Order Book Algorithms

###### 1. ComputeSupportResistance （支撑阻力位计算）

**Algorithm Principle**：Based on price-volume aggregation characteristics in the order book, identify price levels with significant buying or selling pressure.

**Technical Implementation**：

1. **Price Range Division**：Divide the order book price range into fixed-width bins (recommended 50-100 bins)
2. **Cumulative Order Volume Calculation**：Calculate weighted amounts (price × quantity) for buy and sell orders in each bin
3. **Significance Identification**：Use local maximum detection to identify support/resistance levels, with threshold set to 1.5-2.0 times the average cumulative volume
4. **Result Sorting**：Sort by cumulative volume in descending order, return Top-N (recommended 3-5) support and resistance levels

**Application Scenarios**：

- Technical analysis to identify key levels where price may reverse
- Provide key price level references for automated trading strategies
- Monitor breakouts of important market support/resistance levels

**Parameter Configuration**：

| Parameter             | Type    | Description                                                           | Recommended Value | Adjustment Suggestions                                          |
| --------------------- | ------- | --------------------------------------------------------------------- | ----------------- | --------------------------------------------------------------- |
| binCount              | int     | Number of price range divisions                                       | 50                | High liquidity pairs: 30-50, Low liquidity pairs: 20-30         |
| significanceThreshold | float64 | Support/resistance significance threshold                             | 1.5               | High market volatility: 1.2-1.5, Stable market: 1.5-2.0         |
| topN                  | int     | Number of support/resistance levels to return                         | 2                 | Short-term trading: 1-2, Long-term analysis: 3-5                |
| minDistancePercent    | float64 | Minimum price difference percentage between support/resistance levels | 0.5               | High volatility market: 0.3-0.5, Low volatility market: 0.5-1.0 |

**Example Usage**：

```go
// Calculate support and resistance levels for BTC-USDT-SWAP
supports, resistances, err := obManager.ComputeSupportResistance("BTC-USDT-SWAP", 50, 1.5, 2, 0.5)
```

**Output Example**：

```json
{
  "supports": [45000.0, 44500.0],
  "resistances": [46000.0, 46500.0],
  "spread": 1000.0
}
```

###### 2. ComputeLargeOrderDistribution （大额订单分布分析）

**Algorithm Principle**：By identifying the distribution of large orders (whale orders), infer the trading intentions of institutions or large traders, and use a non-linear transformation model to calculate more accurate market sentiment indicators.

**Technical Implementation**：

1. **Large Order Threshold Determination**：Dynamically determine threshold using order amount quantile (recommended 90-95 percentile)
2. **Large Order Identification**：Identify orders with amounts exceeding the threshold
3. **Price Distance Weighting**：Orders closer to the current price have higher weights, weight formula: $w(p) = e^{-ambda \cdot \frac{|p - P\_{\text{mid}}|}{P\_{\text{mid}}}}$
4. **Long-Short Power Comparison**：Calculate weighted buy and sell amounts to derive original long-short tendency indicator
5. **Non-linear Sentiment Transformation**：Use a non-linear transformation model with deadzone threshold to more accurately reflect market sentiment intensity

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

**Example Usage**：

```go
// 分析ETH-USDT-SWAP的大额订单分布
largeBuyNotional, largeSellNotional, sentiment, err := obManager.ComputeLargeOrderDistribution("ETH-USDT-SWAP", 0.95, 5.0, 0.3)
```

**Output Example**：

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

**Algorithm Principle**：Use time window statistics and Z-score to detect sudden changes in order book depth, identifying order book structure changes that may predict significant market movements.

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

**Example Usage**：

```go
// 检测SOL-USDT-SWAP的深度异常
depthAnomaly, err := obManager.DetectDepthAnomaly("SOL-USDT-SWAP", 0.5, 30, 2.0)
```

**Output Example**：

```json
{
  "anomaly": true,
  "z_score": -13.7423,
  "direction": "decrease",
  "intensity": 13.7423,
  "depth": 12345.67
}
```

**Log Output Example**：

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

**Example Usage**：

```go
// 检测DOGE-USDT-SWAP的流动性收缩情况
liquidityShrinkData, err := obManager.DetectLiquidityShrinkage("DOGE-USDT-SWAP", 0.5, 30, 1800, -0.01)
```

**Output Example**：

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

**Log Output Example**：

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

1. 趋势跟踪策略

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

1. 机构资金流向分析

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

1. 风险管理策略

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

###### 参数动态调整

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

