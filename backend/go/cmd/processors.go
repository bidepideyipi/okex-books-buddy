package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// ProcessInstrument handles all analysis computations for a single instrument
func ProcessInstrument(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	_, _, err := obManager.GetTop400(instID)
	if err != nil {
		log.Printf("Order book not ready yet %s: %v", instID, err)
		return
	}

	go processSnapshot(instID, obManager, redisClient)
	go processSupportResistance(instID, obManager, redisClient, cfg)
	go processSpreadZScore(instID, obManager, redisClient, cfg)
	go processLargeOrderDistribution(instID, obManager, redisClient, cfg)
	go processDepthAnomaly(instID, obManager, redisClient, cfg)
	go processLiquidityShrinkage(instID, obManager, redisClient, cfg)
}

func processSnapshot(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client) {
	if ticker, exists := obManager.GetTicker(instID); exists && ticker != nil {
		if err := redisClient.StoreTickerSnapshot(instID, ticker); err != nil {
			log.Printf("Failed to store ticker snapshot for %s: %v", instID, err)
		}
	}

	if book, exists := obManager.GetOrderBook(instID); exists && book != nil {
		asks, bids, err := obManager.GetTop400(instID)
		if err != nil {
			log.Printf("Failed to get order book for %s: %v", instID, err)
			return
		}
		if err := redisClient.StoreOrderBookSnapshot(instID, asks, bids, book.Checksum); err != nil {
			log.Printf("Failed to store order book snapshot for %s: %v", instID, err)
		}
	}
}

func processSupportResistance(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	supports, resistances, spread, err := obManager.ComputeSupportResistance(
		instID,
		cfg.Analysis.SupportResistanceBinCount,
		cfg.Analysis.SupportResistanceSignificanceThreshold,
		cfg.Analysis.SupportResistanceTopN,
		cfg.Analysis.SupportResistanceMinDistancePercent,
	)
	if err != nil {
		log.Printf("Failed to compute support/resistance for %s: %v", instID, err)
		return
	}
	if err := redisClient.StoreSupportResistance(instID, supports, resistances, spread); err != nil {
		log.Printf("Failed to store support/resistance for %s: %v", instID, err)
	}
}

func processSpreadZScore(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
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
		log.Printf("%sSupport Resistance Spread %s for: %s, significantly Z-Score=%.4f, current spread: %.6f%s",
			config.Yellow, trend, instID, zScore, currentSpread, config.Reset)
	}
	if err := redisClient.StoreSpreadZScore(instID, zScore, currentSpread); err != nil {
		log.Printf("Failed to store spread Z-score for %s: %v", instID, err)
	}
}

func processLargeOrderDistribution(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	largeBuy, largeSell, sentiment, err := obManager.ComputeLargeOrderDistribution(
		instID,
		cfg.Analysis.LargeOrderPercentileAlpha,
		cfg.Analysis.LargeOrderDecayLambda,
		cfg.Analysis.LargeOrderSentimentDeadzoneThreshold,
	)
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
}

func processDepthAnomaly(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	depthAnomaly, err := obManager.DetectDepthAnomaly(
		instID,
		cfg.Analysis.DepthAnomalyPriceRangePercent,
		cfg.Analysis.DepthAnomalyWindowSize,
		cfg.Analysis.DepthAnomalyZThreshold,
	)
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
}

func processLiquidityShrinkage(instID string, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	liquidityShrink, err := obManager.DetectLiquidityShrinkage(
		instID,
		cfg.Analysis.LiquidityShrinkNearPriceDeltaPercent,
		cfg.Analysis.LiquidityShrinkShortWindowSeconds,
		cfg.Analysis.LiquidityShrinkLongWindowSeconds,
		cfg.Analysis.LiquidityShrinkSlopeThreshold,
	)
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
}

// StartOrderBookProcessor starts the order book processing goroutine
func StartOrderBookProcessor(ctx context.Context, wsClient *ws.Client, obManager *orderbook.Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, 10)

			for _, instID := range wsClient.GetSubscribed() {
				wg.Add(1)
				go func(instrumentID string) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					ProcessInstrument(instrumentID, obManager, redisClient, cfg)
				}(instID)
			}

			wg.Wait()
		case <-ctx.Done():
			log.Println("Order book processing stopped")
			return
		}
	}
}
