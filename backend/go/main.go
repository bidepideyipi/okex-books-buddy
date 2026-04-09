package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/supermancell/okex-buddy/internal/config"
	connect "github.com/supermancell/okex-buddy/internal/connector"
	httpserver "github.com/supermancell/okex-buddy/internal/http"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	signalservice "github.com/supermancell/okex-buddy/internal/signal"
	"github.com/supermancell/okex-buddy/internal/subscription"
	"github.com/supermancell/okex-buddy/internal/ws"
)

func main() {
	/**
	 * 加载配置
	 */
	cfg := config.LoadFromEnv()
	log.Println("OKEx Buddy - Combined WebSocket Client and API Server")
	log.Printf("Config loaded: Redis=%s, OKEx WS=%s, API HTTP=%s\n", cfg.Redis.Addr, cfg.OKEX.PublicWSURL, cfg.APIHTTPAddr)
	log.Printf("Proxy config: USE_PROXY=%v, PROXY_ADDR=%s", cfg.OKEX.UseProxy, cfg.OKEX.ProxyAddr)
	log.Printf("WebSocket enable: PublicWS=%v, BusinessWS=%v, PrivateWS=%v", cfg.OKEX.EnablePublicWS, cfg.OKEX.EnableBusinessWS, cfg.OKEX.EnablePrivateWS)

	/**
	 * 连接到 Redis
	 */
	redisClient, err := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		//当 Redis 连接失败时，立即将 Redis 健康状态设置为 false ，然后程序退出。这确保了即使服务启动失败，健康检查也能正确反映 Redis 的状态
		httpserver.SetRedisHealthy(false)
		log.Fatalf("Cannot start service without Redis connection")
	}
	defer func() {
		//当程序正常退出时，在 defer 函数中设置 Redis 健康状态为 false 。这确保了服务停止时，健康检查能正确反映 Redis 连接已关闭。
		httpserver.SetRedisHealthy(false)
		if err := redisClient.Close(); err != nil {
			log.Printf("Failed to close Redis client: %v", err)
		}
	}()
	log.Println("Connected to Redis")

	/**
	 * 连接到 MongoDB
	 */
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

	/**
	 * 连接到 Public WebSocket
	 */
	var wsClient *ws.PublicClient
	if cfg.OKEX.EnablePublicWS {
		var connectErr error
		wsClient, connectErr = connect.ConnectPublicWebSocket(cfg, obManager)
		if connectErr != nil {
			log.Printf("Failed to connect to Public WebSocket: %v", connectErr)
		} else if wsClient != nil {
			httpserver.SetPublicWSHealthy(true)
			defer func() {
				httpserver.SetPublicWSHealthy(false)
				wsClient.Close()
			}()
		}
	} else {
		log.Println("Public WebSocket is disabled, skipping connection")
	}

	/**
	 * a.创建一个可取消的上下文
	 * b.创建订单薄处理器
	 * c. SubscriptionManager订阅管理
	 */
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
	}

	/**
	 * 连接到 Business WebSocket
	 */
	var businessWsClient *ws.BusinessClient
	if mongoClient != nil && cfg.OKEX.EnableBusinessWS {
		var connectErr error
		businessWsClient, connectErr = connect.ConnectBusinessWebSocket(cfg, mongoClient)
		if connectErr != nil {
			log.Printf("Failed to connect to Business WebSocket: %v", connectErr)
		} else if businessWsClient != nil {
			httpserver.SetBusinessWSHealthy(true)
			defer func() {
				httpserver.SetBusinessWSHealthy(false)
				businessWsClient.Close()
			}()
		}
	} else if mongoClient != nil {
		log.Println("Business WebSocket is disabled, skipping connection")
	}

	/**
	 * 连接到 Private WebSocket
	 */
	var privateWsClient *ws.PrivateClient
	if mongoClient != nil && cfg.OKEX.EnablePrivateWS {
		var connectErr error
		privateWsClient, connectErr = connect.ConnectPrivateWebSocket(cfg, mongoClient, redisClient)
		if connectErr != nil {
			log.Printf("Failed to connect to Private WebSocket: %v", connectErr)
		} else if privateWsClient != nil {
			httpserver.SetPrivateWSHealthy(true)
			defer func() {
				httpserver.SetPrivateWSHealthy(false)
				privateWsClient.Close()
			}()

			if cfg.OKEX.EnablePrivateWS {
				//开启交易型号接收器
				signalservice.StartSignalConsumer(redisClient, mongoClient, privateWsClient)
				signalservice.StartStreamConsumer(redisClient, mongoClient)
			}
		}
	} else if mongoClient != nil {
		log.Println("Private WebSocket is disabled, skipping connection")
	}

	/**
	* Http接口开放
	 */
	httpServerDone := make(chan struct{})
	httpServerStop := make(chan struct{})
	go httpserver.StartHTTPServer(cfg.APIHTTPAddr, httpServerDone, httpServerStop)

	/**
	* 处理停机信号
	 */
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
