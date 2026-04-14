package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/supermancell/okex-buddy/internal/assembly"
	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/handler"
	httpserver "github.com/supermancell/okex-buddy/internal/http"
)

func main() {

	var cfg *config.AppConfig
	/**
	 * 加载配置
	 */
	app := assembly.NewAssembly()

	cfg = app.Config
	log.Println("OKEx Buddy - Combined WebSocket Client and API Server")
	log.Printf("Config loaded: Redis=%s, OKEx WS=%s, API HTTP=%s\n", cfg.Redis.Addr, cfg.OKEX.PublicWSURL, cfg.APIHTTPAddr)
	log.Printf("Proxy config: USE_PROXY=%v, PROXY_ADDR=%s", cfg.OKEX.UseProxy, cfg.OKEX.ProxyAddr)
	log.Printf("WebSocket enable: PublicWS=%v, BusinessWS=%v, PrivateWS=%v", cfg.OKEX.EnablePublicWS, cfg.OKEX.EnableBusinessWS, cfg.OKEX.EnablePrivateWS)

	/**
	 * 连接到 Redis
	 */
	if err := app.LoadRedis(); err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		//当 Redis 连接失败时，立即将 Redis 健康状态设置为 false ，然后程序退出。这确保了即使服务启动失败，健康检查也能正确反映 Redis 的状态
		log.Fatalf("Cannot start service without Redis connection")
		return
	}

	log.Println("Connected to Redis")

	/**
	 * 连接到 MongoDB
	 */
	if err := app.LoadMongo(); err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
		return
	}

	/**
	* 连接到Private websocket，消息处理委托给msgHandler
	 */
	msgHandler := handler.NewPrivateMessageHandler(app.MongoClient)
	if connectErr := app.LoadPrivateWs(msgHandler); connectErr != nil {
		log.Printf("Failed to connect to Private WebSocket: %v", connectErr)
	}

	/**
	 * 连接到 OKEx REST API
	 */
	if err := app.LoadOkxRest(); err != nil {
		log.Printf("Failed to connect to OKEx REST API: %v", err)
		return
	}

	log.Println("Connected to OKEx REST API")

	/**
	* 连接到Redis stream，消息处理委托给signalHandler
	 */
	signalHandler := handler.NewRedisStreamMessageHandler(app.MongoClient, app.OkxClient)
	app.LoadStreamConsumer(signalHandler)

	//启动组件的优雅停机监
	gracefulShutdown(app)

	/**
	* Http接口开放
	 */
	httpServerDone := make(chan struct{})
	httpServerStop := make(chan struct{})
	go httpserver.StartHTTPServer(cfg.APIHTTPAddr, httpServerDone, httpServerStop)

	close(httpServerStop)
	<-httpServerDone
	log.Println("HTTP server stopped")
	log.Println("Shutdown complete")

}

func gracefulShutdown(app *assembly.Assembly) {

	/**
	* 处理停机信号
	 */
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Service is running. Press Ctrl+C to exit.")
	<-sigChan

	log.Println("Received shutdown signal...")
	log.Println("Shutting down gracefully...")

	err := app.CloseRedis()
	if err != nil {
		log.Printf("Error closing Redis Client: %v", err)
	}

	app.CloseStreamConsumer()

	err = app.ClosePrivateWs()
	if err != nil {
		log.Printf("Error closing Private WebSocket: %v", err)
	}

	err = app.CloseMongo()
	if err != nil {
		log.Printf("Error closing Mongo Client: %v", err)
	}

}
