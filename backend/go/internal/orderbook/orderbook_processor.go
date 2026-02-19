package orderbook

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// ProcessInstrument handles all analysis computations for a single instrument
func ProcessInstrument(instID string, obManager *Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	_, _, err := obManager.GetTop400(instID)
	if err != nil {
		log.Printf("Order book not ready yet %s: %v", instID, err)
		return
	}

	go processSnapshot(instID, obManager, redisClient)
}

func processSnapshot(instID string, obManager *Manager, redisClient *redisclient.Client) {
	bids, asks, err := obManager.GetTop400(instID)
	if err != nil {
		log.Printf("Failed to get order book snapshot for %s: %v", instID, err)
		return
	}

	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"bids":      bids,
		"asks":      asks,
	}

	if err := redisClient.StoreOrderBookSnapshot(instID, data["asks"], data["bids"], 0); err != nil {
		log.Printf("Failed to save order book snapshot for %s: %v", instID, err)
	}
}

// StartOrderBookProcessor starts order book processing loop
func StartOrderBookProcessor(ctx context.Context, wsClient *ws.PublicClient, obManager *Manager, redisClient *redisclient.Client, cfg config.AppConfig) {
	ticker := time.NewTicker(time.Duration(cfg.Redis.PollIntervalSec) * time.Second)
	defer ticker.Stop()

	const maxConcurrent = 10
	semaphore := make(chan struct{}, maxConcurrent)

	for {
		select {
		case <-ticker.C:
			var wg sync.WaitGroup

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
