package main

import (
	"fmt"
	"log"
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

		// Publish to Redis List for Bytewax processing
		if err := redisClient.PublishOrderBookEvent("orderbook:events", string(msg)); err != nil {
			log.Printf("Failed to publish to Redis: %v", err)
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
					continue // Order book not ready yet
				}

				book, _ := obManager.GetOrderBook(instID)
				if book != nil {
					if err := redisClient.StoreOrderBookSnapshot(instID, asks, bids, book.Checksum); err != nil {
						log.Printf("Failed to store snapshot for %s: %v", instID, err)
					}
				}

				// Compute support/resistance levels using in-memory order book and store in Redis
				supports, resistances, err := obManager.ComputeSupportResistance(instID, 80, 1.5, 2)
				if err != nil {
					log.Printf("Failed to compute support/resistance for %s: %v", instID, err)
					continue
				}

				if err := redisClient.StoreSupportResistance(instID, supports, resistances); err != nil {
					log.Printf("Failed to store support/resistance for %s: %v", instID, err)
				}

				// Compute large order distribution and sentiment and store in Redis
				largeBuy, largeSell, sentiment, err := obManager.ComputeLargeOrderDistribution(instID, 0.95, 7.0, 0.3)
				if err != nil {
					log.Printf("Failed to compute large order distribution for %s: %v", instID, err)
					continue
				}

				if err := redisClient.StoreLargeOrderDistribution(instID, largeBuy, largeSell, sentiment); err != nil {
					log.Printf("Failed to store large order distribution for %s: %v", instID, err)
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
