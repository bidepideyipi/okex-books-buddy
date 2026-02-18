package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// HealthCheckResponse represents the health check response structure
type HealthCheckResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		WebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"websocket"`
		Redis struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"redis"`
	} `json:"data"`
}

// StartHTTPServer starts the HTTP server in a separate goroutine
func StartHTTPServer(addr string, done chan struct{}, stop chan struct{}) {
	defer close(done)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	mux.HandleFunc("/health", handleHealthCheck)

	go func() {
		log.Printf("HTTP server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down HTTP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped gracefully")
	}
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    405,
			"message": "method not allowed",
		})
		return
	}

	wsStatus := atomic.LoadInt32(&websocketHealthy)
	redisStatus := atomic.LoadInt32(&redisHealthy)

	response := HealthCheckResponse{
		Code:    200,
		Message: "success",
	}

	if wsStatus == 1 {
		response.Data.WebSocket.Status = "healthy"
		response.Data.WebSocket.Message = "WebSocket connections are active"
	} else {
		response.Data.WebSocket.Status = "unhealthy"
		response.Data.WebSocket.Message = "WebSocket connections failed or reached max reconnection attempts"
		response.Code = 503
	}
	response.Data.WebSocket.Timestamp = time.Now().Unix()

	if redisStatus == 1 {
		response.Data.Redis.Status = "healthy"
		response.Data.Redis.Message = "Redis connection is active"
	} else {
		response.Data.Redis.Status = "unhealthy"
		response.Data.Redis.Message = "Redis connection failed or closed"
		response.Code = 503
	}
	response.Data.Redis.Timestamp = time.Now().Unix()

	if response.Code == 503 {
		response.Message = "service unavailable"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Code)
	json.NewEncoder(w).Encode(response)
}

// ConnectPublicWebSocket connects to the public WebSocket endpoint
func ConnectPublicWebSocket(cfg config.AppConfig, obManager *orderbook.Manager) *ws.Client {
	log.Printf("Public WebSocket is enabled, connecting to: %s", cfg.OKEX.PublicWSURL)
	messageHandler := NewPublicMessageHandler(obManager)

	var wsClient *ws.Client
	if cfg.OKEX.UseProxy {
		log.Printf("Proxy enabled: %s", cfg.OKEX.ProxyAddr)
		wsClient = ws.NewClientWithProxy(cfg.OKEX.PublicWSURL, messageHandler, true, cfg.OKEX.ProxyAddr)
	} else {
		log.Println("Proxy disabled, connecting directly")
		wsClient = ws.NewClient(cfg.OKEX.PublicWSURL, messageHandler)
	}

	if err := wsClient.Connect(); err != nil {
		log.Printf("Failed to connect to OKEx WebSocket: %v", err)
		atomic.StoreInt32(&websocketHealthy, 0)
		log.Fatalf("Cannot start service without WebSocket connection")
	}

	log.Println("Connected to OKEx WebSocket")
	return wsClient
}

// ConnectBusinessWebSocket connects to the business WebSocket endpoint
func ConnectBusinessWebSocket(cfg config.AppConfig, mongoClient *mongodb.Client) *ws.BusinessClient {
	log.Printf("Business WebSocket is enabled, connecting to: %s", cfg.OKEX.BusinessWSURL)
	businessMessageHandler := NewBusinessMessageHandler(mongoClient)

	var businessWsClient *ws.BusinessClient
	if cfg.OKEX.UseProxy {
		log.Printf("Business WebSocket using proxy: %s", cfg.OKEX.ProxyAddr)
		businessWsClient = ws.NewBusinessClientWithProxy(cfg.OKEX.BusinessWSURL, businessMessageHandler, true, cfg.OKEX.ProxyAddr)
	} else {
		log.Println("Business WebSocket proxy disabled, connecting directly")
		businessWsClient = ws.NewBusinessClient(cfg.OKEX.BusinessWSURL, businessMessageHandler)
	}

	log.Println("Attempting to connect to Business WebSocket...")
	if err := businessWsClient.Connect(); err != nil {
		log.Printf("Failed to connect to OKEx Business WebSocket: %v", err)
		return nil
	}

	log.Println("Connected to OKEx Business WebSocket")

	instruments := []string{"ETH-USDT-SWAP"}
	channels := []string{config.Candle1D, config.Candle4H, config.Candle1H, config.Candle15m}
	log.Printf("Subscribing to Business WebSocket instruments: %v, channels: %v", instruments, channels)
	if err := businessWsClient.Subscribe(instruments, channels); err != nil {
		log.Printf("Failed to subscribe to candlestick channels: %v", err)
	} else {
		log.Println("Successfully subscribed to candlestick channels")
	}

	return businessWsClient
}
