package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/supermancell/okex-buddy/internal/config"
	httpserver "github.com/supermancell/okex-buddy/internal/http"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	signalservice "github.com/supermancell/okex-buddy/internal/signal"
	"github.com/supermancell/okex-buddy/internal/subscription"
	"github.com/supermancell/okex-buddy/internal/ws"
	"github.com/supermancell/okex-buddy/internal/wshub"
)

func main() {
	cfg := config.LoadFromEnv()
	log.Println("OKEx Buddy - Combined WebSocket Client and API Server")
	log.Printf("Config loaded: Redis=%s, OKEx WS=%s, API HTTP=%s\n", cfg.Redis.Addr, cfg.OKEX.PublicWSURL, cfg.APIHTTPAddr)
	log.Printf("Proxy config: USE_PROXY=%v, PROXY_ADDR=%s", cfg.OKEX.UseProxy, cfg.OKEX.ProxyAddr)
	log.Printf("WebSocket enable: PublicWS=%v, BusinessWS=%v, PrivateWS=%v", cfg.OKEX.EnablePublicWS, cfg.OKEX.EnableBusinessWS, cfg.OKEX.EnablePrivateWS)

	redisClient, err := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		httpserver.SetRedisHealthy(false)
		log.Fatalf("Cannot start service without Redis connection")
	}
	defer func() {
		httpserver.SetRedisHealthy(false)
		if err := redisClient.Close(); err != nil {
			log.Printf("Failed to close Redis client: %v", err)
		}
	}()
	log.Println("Connected to Redis")

	var mongoClient *mongodb.Client
	if cfg.MongoDB.Addr != "" {
		mongoClient, err = mongodb.NewClient(cfg.MongoDB.Addr, cfg.MongoDB.Database)
		if err != nil {
			log.Printf("Failed to connect to MongoDB: %v", err)
		} else {
			defer func() {
				if err := mongoClient.Close(); err != nil {
					log.Printf("Failed to close MongoDB client: %v", err)
				}
			}()
			log.Println("Connected to MongoDB")
		}
	}

	obManager := orderbook.NewManager()

	var wsClient *ws.PublicClient
	if cfg.OKEX.EnablePublicWS {
		wsClient = ConnectPublicWebSocket(cfg, obManager)
		defer func() {
			if wsClient != nil {
				httpserver.SetWSHealthy(false)
				wsClient.Close()
			}
		}()
	} else {
		log.Println("Public WebSocket is disabled, skipping connection")
	}

	var businessWsClient *ws.BusinessClient
	if mongoClient != nil && cfg.OKEX.EnableBusinessWS {
		businessWsClient = ConnectBusinessWebSocket(cfg, mongoClient)
		if businessWsClient != nil {
			defer businessWsClient.Close()
		}
	} else if mongoClient != nil {
		log.Println("Business WebSocket is disabled, skipping connection")
	}

	var privateWsClient *ws.PrivateClient
	if mongoClient != nil && cfg.OKEX.EnablePrivateWS {
		privateWsClient = ConnectPrivateWebSocket(cfg, mongoClient, redisClient)
		if privateWsClient != nil {
			defer privateWsClient.Close()

			if cfg.OKEX.EnablePrivateWS {
				signalservice.StartSignalConsumer(redisClient, mongoClient, privateWsClient)
			}
		}
	} else if mongoClient != nil {
		log.Println("Private WebSocket is disabled, skipping connection")
	}

	hub := wshub.NewHub()
	go hub.Run()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if wsClient != nil {
		go orderbook.StartOrderBookProcessor(ctx, wsClient, obManager, redisClient, cfg)
	}

	var subManager *subscription.SubscriptionManager
	if wsClient != nil {
		subManager = subscription.NewSubscriptionManager(
			wsClient,
			redisClient,
			cfg.Redis.TradingPairsKey,
			cfg.Redis.PollIntervalSec,
		)

		if err := subManager.Start(); err != nil {
			log.Fatalf("Failed to start subscription manager: %v", err)
		}
		defer subManager.Stop()
		log.Printf("Subscription manager started (polling every %d seconds)", cfg.Redis.PollIntervalSec)
	} else {
		log.Println("Subscription manager skipped because Public WebSocket is disabled")
	}

	monitoringData := map[string]interface{}{
		"websocket_connections": 0,
		"active_pairs":          0,
	}
	if wsClient != nil {
		monitoringData["websocket_connections"] = 1
		monitoringData["active_pairs"] = len(wsClient.GetSubscribed())
	}
	if err := redisClient.UpdateSystemMonitoring(monitoringData); err != nil {
		log.Printf("Failed to update system monitoring: %v", err)
	}

	httpServerDone := make(chan struct{})
	httpServerStop := make(chan struct{})
	go httpserver.StartHTTPServer(cfg.APIHTTPAddr, httpServerDone, httpServerStop)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Service is running. Press Ctrl+C to exit.")
	<-sigChan

	log.Println("Received shutdown signal...")
	log.Println("Shutting down gracefully...")
	cancel()
	log.Println("Context cancelled, waiting for order book processing to stop...")

	close(httpServerStop)
	<-httpServerDone
	log.Println("HTTP server stopped")
	log.Println("Shutdown complete")
}
