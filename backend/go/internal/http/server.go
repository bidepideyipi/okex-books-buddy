package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	wsHealthy  int32 = 1
	redisHealthy int32 = 1
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

// SetWSHealthy sets the WebSocket health status
func SetWSHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&wsHealthy, 1)
	} else {
		atomic.StoreInt32(&wsHealthy, 0)
	}
}

// SetRedisHealthy sets the Redis health status
func SetRedisHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&redisHealthy, 1)
	} else {
		atomic.StoreInt32(&redisHealthy, 0)
	}
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

	wsStatus := atomic.LoadInt32(&wsHealthy)
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
