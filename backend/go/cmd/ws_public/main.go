package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// processInstrument handles all analysis computations for a single instrument concurrently
func processInstrument(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	// Get order book data
	asks, bids, err := obManager.GetTop400(instID)
	if err != nil {
		log.Printf("Order book not ready yet %s: %v", instID, err)
		return
	}

	go func() {
		// Store ticker snapshot
		if ticker, exists := obManager.GetTicker(instID); exists && ticker != nil {
			if err := redisClient.StoreTickerSnapshot(instID, ticker); err != nil {
				log.Printf("Failed to store ticker snapshot for %s: %v", instID, err)
			}
		}

		// Store order book snapshot
		if book, exists := obManager.GetOrderBook(instID); exists && book != nil {
			if err := redisClient.StoreOrderBookSnapshot(instID, asks, bids, book.Checksum); err != nil {
				log.Printf("Failed to store order book snapshot for %s: %v", instID, err)
			}
		}
	}()

	// Compute and store support/resistance levels
	go func() {
		supports, resistances, spread, err := obManager.ComputeSupportResistance(instID, cfg.Analysis.SupportResistanceBinCount, cfg.Analysis.SupportResistanceSignificanceThreshold, cfg.Analysis.SupportResistanceTopN, cfg.Analysis.SupportResistanceMinDistancePercent)
		if err != nil {
			log.Printf("Failed to compute support/resistance for %s: %v", instID, err)
			return
		}
		if err := redisClient.StoreSupportResistance(instID, supports, resistances, spread); err != nil {
			log.Printf("Failed to store support/resistance for %s: %v", instID, err)
		}
	}()

	// Compute and store spread Z-score
	go func() {
		zScore, currentSpread, err := obManager.AnalyzeSpreadZScore(instID, 5)
		if err != nil {
			log.Printf("Failed to analyze spread Z-score for %s: %v", instID, err)
			return
		}
		if math.Abs(zScore) > 2.5 {
			trend := "expanded"
			if zScore < 0 {
				trend = "contracted"
			}
			log.Printf("\033[33mSupport Resistance Spread %s for: %s, significantly Z-Score=%.4f, current spread: %.6f\033[0m",
				trend, instID, zScore, currentSpread)
		}
		if err := redisClient.StoreSpreadZScore(instID, zScore, currentSpread); err != nil {
			log.Printf("Failed to store spread Z-score for %s: %v", instID, err)
		}
	}()

	// Compute and store large order distribution
	go func() {
		largeBuy, largeSell, sentiment, err := obManager.ComputeLargeOrderDistribution(instID, cfg.Analysis.LargeOrderPercentileAlpha, cfg.Analysis.LargeOrderDecayLambda, cfg.Analysis.LargeOrderSentimentDeadzoneThreshold)
		if err != nil {
			log.Printf("Failed to compute large order distribution for %s: %v", instID, err)
			return
		}

		if math.Abs(sentiment) > 0.3 {
			var colorCode string
			if sentiment > 0.3 {
				colorCode = config.Blue
			} else if sentiment < -0.3 {
				colorCode = config.Red
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
	}()

	// Compute and store depth anomaly
	go func() {
		depthAnomaly, err := obManager.DetectDepthAnomaly(instID, cfg.Analysis.DepthAnomalyPriceRangePercent, cfg.Analysis.DepthAnomalyWindowSize, cfg.Analysis.DepthAnomalyZThreshold)
		if err != nil {
			log.Printf("Failed to detect depth anomaly for %s: %v", instID, err)
			return
		}
		if depthAnomaly.Anomaly && depthAnomaly.Intensity > 2.5 {
			log.Printf("%sDepth Anomaly Detected for %s: Z-Score=%.4f, Direction=%s, Intensity=%.4f%s",
				config.Green, instID, depthAnomaly.ZScore, depthAnomaly.Direction, depthAnomaly.Intensity, config.Reset)
		}
		if err := redisClient.StoreDepthAnomaly(instID, depthAnomaly.ToRedisMap()); err != nil {
			log.Printf("Failed to store depth anomaly for %s: %v", instID, err)
		}
	}()

	// Compute and store liquidity shrinkage
	go func() {
		liquidityShrink, err := obManager.DetectLiquidityShrinkage(instID, cfg.Analysis.LiquidityShrinkNearPriceDeltaPercent, cfg.Analysis.LiquidityShrinkShortWindowSeconds, cfg.Analysis.LiquidityShrinkLongWindowSeconds, cfg.Analysis.LiquidityShrinkSlopeThreshold)
		if err != nil {
			log.Printf("Failed to detect liquidity shrinkage for %s: %v", instID, err)
			return
		}
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
		if err := redisClient.StoreLiquidityShrink(instID, liquidityShrink.ToRedisMap()); err != nil {
			log.Printf("Failed to store liquidity shrinkage for %s: %v", instID, err)
		}
	}()
}

func main() {
	// Load configuration
	cfg := config.LoadFromEnv()
	log.Println("OKEx WebSocket Client - Books Implementation")
	log.Printf("Config loaded: Redis=%s, OKEx WS=%s\n", cfg.Redis.Addr, cfg.OKEX.PublicWSURL)

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
			// Process all subscribed instruments concurrently with semaphore limiting
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, 10) // Limit concurrent processing to 10 instruments

			for _, instID := range wsClient.GetSubscribed() {
				wg.Add(1)
				go func(instrumentID string) {
					defer wg.Done()
					semaphore <- struct{}{}        // Acquire semaphore
					defer func() { <-semaphore }() // Release semaphore

					processInstrument(instrumentID, obManager, redisClient, cfg)
				}(instID)
			}

			wg.Wait() // Wait for all instruments to be processed
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
