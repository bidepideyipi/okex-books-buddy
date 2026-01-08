package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/ws"
)

func main() {
	// Load configuration
	cfg := config.LoadFromEnv()
	fmt.Println("OKEx WebSocket Client - M2 Implementation")
	fmt.Printf("Config loaded: Redis=%s, OKEx WS=%s\n", cfg.Redis.Addr, cfg.OKEX.PublicWSURL)

	// Initialize Redis client
	redisClient, err := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Initialize order book manager
	obManager := orderbook.NewManager()

	// Create message handler that processes order book updates
	messageHandler := func(msg []byte) error {
		// Process message and update order book
		if err := obManager.ProcessMessage(msg); err != nil {
			return fmt.Errorf("failed to process message: %w", err)
		}

		return nil
	}

	// Initialize WebSocket client
	var wsClient *ws.Client
	if cfg.OKEX.UseProxy {
		log.Printf("Proxy enabled: %s", cfg.OKEX.ProxyAddr)
		wsClient = ws.NewClientWithProxy(cfg.OKEX.PublicWSURL, messageHandler, true, cfg.OKEX.ProxyAddr)
	} else {
		log.Println("Proxy disabled, connecting directly")
		wsClient = ws.NewClient(cfg.OKEX.PublicWSURL, messageHandler)
	}

	// Connect to OKEx WebSocket
	if err := wsClient.Connect(); err != nil {
		log.Fatalf("Failed to connect to OKEx WebSocket: %v", err)
	}
	defer wsClient.Close()
	log.Println("Connected to OKEx WebSocket")

	// Start goroutine to periodically store order book snapshots to Redis Hash
	go func() {
		ticker := time.NewTicker(1 * time.Second) // Store snapshot every second
		defer ticker.Stop()

		for range ticker.C {
			// Store snapshots for all subscribed instruments
			for _, instID := range wsClient.GetSubscribed() {
				asks, bids, err := obManager.GetTop400(instID)
				if err != nil {
					log.Printf("Order book not ready yet %s: %v", instID, err)
					continue // Order book not ready yet
				}

				book, _ := obManager.GetOrderBook(instID)
				if book != nil {
					if err := redisClient.StoreOrderBookSnapshot(instID, asks, bids, book.Checksum); err != nil {
						log.Printf("Failed to store snapshot for %s: %v", instID, err)
					}
				}

				// Compute support/resistance levels using in-memory order book and store in Redis
				supports, resistances, spread, err := obManager.ComputeSupportResistance(instID, cfg.Analysis.SupportResistanceBinCount, cfg.Analysis.SupportResistanceSignificanceThreshold, cfg.Analysis.SupportResistanceTopN, cfg.Analysis.SupportResistanceMinDistancePercent)
				if err != nil {
					log.Printf("Failed to compute support/resistance for %s: %v", instID, err)
					continue
				}

				if err := redisClient.StoreSupportResistance(instID, supports, resistances, spread); err != nil {
					log.Printf("Failed to store support/resistance for %s: %v", instID, err)
				}

				// Analyze and store spread Z-score metric
				zScore, currentSpread, err := obManager.AnalyzeSpreadZScore(instID, 5) // 5-minute window
				if err != nil {
					log.Printf("Failed to analyze spread Z-score for %s: %v", instID, err)
				} else {
					if math.Abs(zScore) > 2.5 { // Only log if volatility is significant (>2.5 standard deviations)
						trend := "expanded"
						if zScore < 0 {
							trend = "contracted"
						}
						log.Printf("\033[33mSpread %s significantly (Z-Score=%.4f) for %s: current spread: %.6f\033[0m",
							trend, zScore, instID, currentSpread)
					}
					if err := redisClient.StoreSpreadZScore(instID, zScore, currentSpread); err != nil {
						log.Printf("Failed to store spread Z-score for %s: %v", instID, err)
					}
				}

				/*
					Compute large order distribution with deadzone threshold
					Parameters: instID, percentileAlpha(0.95), decayLambda(7.0), sentimentDeadzoneThreshold(0.3)
						## 参数详解
						### 1. percentileAlpha (0.95)
						- 含义 ：大订单阈值百分位。计算所有订单名义价值（price×size）的第95百分位数，高于此值的订单被视为"大订单"。
						- 作用 ：动态适应不同市场的流动性状况，自动调整大订单的判断标准。
						- 影响 ：
						- 值越大，大订单的定义越严格（只有极少数最大订单被识别）
						- 值越小，会识别更多相对较小的订单为"大订单"
						- 默认值 ：0.95（与当前设置一致）
						### 2. decayLambda (7.0)
						- 含义 ：距离衰减系数。用于计算订单权重的指数衰减参数，订单距离中间价越远，权重越低。
						- 计算公式 ： weight = exp(-decayLambda × |price - mid| / mid)
						- 作用 ：给更接近当前价格的大订单赋予更高权重，反映短期市场意图。
						- 影响 ：
						- 值越大，距离对权重的影响越显著（远价订单权重快速衰减）
						- 值越小，距离的影响越弱（远近订单权重差异较小）
						- 默认值 ：5.0（当前设置7.0比默认值更强调近价订单）
						### 3. sentimentThreshold (0.3)
						- 含义 ：情绪强度阈值。用于解释最终情绪指标的参考值（实际计算中仅作为默认值，函数内部不直接使用此阈值进行判断）。
						- 作用 ：提供情绪指标的解读标准，如绝对值超过0.3表示较强的多空倾向。
						- 默认值 ：0.3（与当前设置一致）
						## 与默认值对比分析
						当前参数设置与默认值的主要差异在于：

						- decayLambda从5.0提高到7.0 ：增强了对近价大订单的关注度，过滤掉更多远价大订单的影响，更聚焦于短期市场动力。
						- 其他参数保持默认值，兼顾了大订单识别的严格性和情绪解读的标准性。
						### 1. 日内短线交易（Scalping）
						- percentileAlpha = 0.98 ：识别极少数真正的超大订单
						- decayLambda = 3.0 ：适当关注稍远价格的大订单，捕捉潜在突破
						- sentimentDeadzoneThreshold = 0.4 ：提高情绪中性死区阈值，只关注强烈信号
						### 2. 趋势跟踪（Trend Following）
						- percentileAlpha = 0.95 ：标准设置，平衡敏感度和准确性
						- decayLambda = 5.0 ：默认值，综合考虑不同价格的大订单
						- sentimentDeadzoneThreshold = 0.3 ：标准中性死区阈值，适应趋势交易的信号需求
						### 3. 做市策略（Market Making）
						- percentileAlpha = 0.90 ：识别更多相对较大的订单，提前感知流动性变化
						- decayLambda = 8.0 ：强烈关注近价订单，保护做市头寸
						- sentimentDeadzoneThreshold = 0.2 ：降低中性死区阈值，捕捉微弱信号
						### 4. 高波动市场
						- 降低 decayLambda 至3.0-4.0，扩大关注范围
						- 提高 percentileAlpha 至0.97-0.99，避免噪音干扰
						### 5. 低波动市场
						- 提高 decayLambda 至6.0-8.0，聚焦核心价格区域
						- 降低 percentileAlpha 至0.92-0.94，提高信号敏感度
						函数返回三个值：
						1. largeBuyNotional ：加权后的大买单总名义价值（bullPower）
						2. largeSellNotional ：加权后的大卖单总名义价值（bearPower）
						3. sentiment ：标准化情绪指标，计算公式： (bullPower - bearPower) / (bullPower + bearPower)
						### 情绪指标解读
						- sentiment > 0.3 ：强烈看涨信号
						- 0.1 < sentiment ≤ 0.3 ：温和看涨信号
						- -0.1 ≤ sentiment ≤ 0.1 ：中性市场
						- -0.3 ≤ sentiment < -0.1 ：温和看跌信号
						- sentiment < -0.3 ：强烈看跌信号
				*/
				largeBuy, largeSell, sentiment, err := obManager.ComputeLargeOrderDistribution(instID, cfg.Analysis.LargeOrderPercentileAlpha, cfg.Analysis.LargeOrderDecayLambda, cfg.Analysis.LargeOrderSentimentDeadzoneThreshold)
				if err != nil {
					log.Printf("Failed to compute large order distribution for %s: %v", instID, err)
					continue
				}

				if math.Abs(sentiment) > 0.3 {
					var colorCode string
					if sentiment > 0.3 {
						colorCode = config.Blue // 绿色 - 强烈看涨
					} else if sentiment < -0.3 {
						colorCode = config.Red // 红色 - 强烈看跌
					} else {
						colorCode = config.Yellow
					}

					log.Printf("%sLargeBuy: %.2f, LargeSell: %.2f, Sentiment: %.4f for %s%s", colorCode, largeBuy, largeSell, sentiment, instID, config.Reset)
				}

				hashKey := fmt.Sprintf(config.SentimentKey, instID)

				fields := map[string]interface{}{
					"instrument_id": instID,
					"analysis_time": time.Now().Unix(),
					"sentiment":     sentiment,
				}

				if err := redisClient.HashSave(hashKey, fields); err != nil {
					log.Printf("Failed to store large order distribution for %s: %v", instID, err)
				}

				// Compute depth anomaly detection and store in Redis
				// 计算订单簿深度异常检测结果并存储到Redis
				depthAnomaly, err := obManager.DetectDepthAnomaly(instID, cfg.Analysis.DepthAnomalyPriceRangePercent, cfg.Analysis.DepthAnomalyWindowSize, cfg.Analysis.DepthAnomalyZThreshold)
				if err != nil {
					log.Printf("Failed to detect depth anomaly for %s: %v", instID, err)
				} else {
					if depthAnomaly.Anomaly && depthAnomaly.Intensity > 2.5 {
						log.Printf("%sDepth Anomaly Detected for %s: Z-Score=%.4f, Direction=%s, Intensity=%.4f%s",
							config.Green, instID, depthAnomaly.ZScore, depthAnomaly.Direction, depthAnomaly.Intensity, config.Reset)
					}

					// Store depth anomaly in Redis
					if err := redisClient.StoreDepthAnomaly(instID, depthAnomaly.ToRedisMap()); err != nil {
						log.Printf("Failed to store depth anomaly for %s: %v", instID, err)
					}
				}

				// Compute liquidity shrinkage warning and store in Redis
				// 计算流动性收缩警告并存储到Redis
				liquidityShrink, err := obManager.DetectLiquidityShrinkage(instID, cfg.Analysis.LiquidityShrinkNearPriceDeltaPercent, cfg.Analysis.LiquidityShrinkShortWindowSeconds, cfg.Analysis.LiquidityShrinkLongWindowSeconds, cfg.Analysis.LiquidityShrinkSlopeThreshold)
				if err != nil {
					log.Printf("Failed to detect liquidity shrinkage for %s: %v", instID, err)
				} else {
					if liquidityShrink.Warning {
						var warningColor string
						switch liquidityShrink.WarningLevel {
						case "light":
							warningColor = config.Yellow
						case "moderate":
							warningColor = config.Red
						case "severe":
							warningColor = config.Red
						default:
							warningColor = config.Reset
						}

						if warningColor == config.Red && liquidityShrink.Slope < -20 {
							log.Printf("%sLiquidity Shrinkage Warning for %s: Level=%s, Liquidity=%.4f, Slope=%.6f%s",
								warningColor, instID, liquidityShrink.WarningLevel, liquidityShrink.Liquidity, liquidityShrink.Slope, config.Reset)
						}
					}

					// Store liquidity shrinkage in Redis
					if err := redisClient.StoreLiquidityShrink(instID, liquidityShrink.ToRedisMap()); err != nil {
						log.Printf("Failed to store liquidity shrinkage for %s: %v", instID, err)
					}
				}
			}
		}
	}()

	// Initialize subscription manager for dynamic subscription
	subManager := ws.NewSubscriptionManager(
		wsClient,
		redisClient,
		cfg.Redis.TradingPairsKey,
		cfg.Redis.PollIntervalSec,
	)

	// Start subscription manager
	if err := subManager.Start(); err != nil {
		log.Fatalf("Failed to start subscription manager: %v", err)
	}
	defer subManager.Stop()
	log.Printf("Subscription manager started (polling every %d seconds)", cfg.Redis.PollIntervalSec)

	// Update system monitoring
	if err := redisClient.UpdateSystemMonitoring(map[string]interface{}{
		"websocket_connections": 1,
		"active_pairs":          len(wsClient.GetSubscribed()),
	}); err != nil {
		log.Printf("Failed to update system monitoring: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("WebSocket client is running. Press Ctrl+C to exit.")
	<-sigChan

	log.Println("Shutting down gracefully...")
}
