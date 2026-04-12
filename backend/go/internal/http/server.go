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
	publicWSHealthy   int32 = 0
	privateWSHealthy  int32 = 0
	businessWSHealthy int32 = 0
	redisHealthy      int32 = 1
	okexHTTPClient    *OKExHTTPClient
)

// SetPublicWSHealthy sets the Public WebSocket health status
func SetPublicWSHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&publicWSHealthy, 1)
	} else {
		atomic.StoreInt32(&publicWSHealthy, 0)
	}
}

// SetPrivateWSHealthy sets the Private WebSocket health status
func SetPrivateWSHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&privateWSHealthy, 1)
	} else {
		atomic.StoreInt32(&privateWSHealthy, 0)
	}
}

// SetBusinessWSHealthy sets the Business WebSocket health status
func SetBusinessWSHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&businessWSHealthy, 1)
	} else {
		atomic.StoreInt32(&businessWSHealthy, 0)
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

	publicWSStatus := atomic.LoadInt32(&publicWSHealthy)
	privateWSStatus := atomic.LoadInt32(&privateWSHealthy)
	businessWSStatus := atomic.LoadInt32(&businessWSHealthy)
	redisStatus := atomic.LoadInt32(&redisHealthy)

	response := HealthCheckResponse{
		Code:    200,
		Message: "success",
	}

	if publicWSStatus == 1 {
		response.Data.PublicWebSocket.Status = "healthy"
		response.Data.PublicWebSocket.Message = "Public WebSocket connection is active"
	} else {
		response.Data.PublicWebSocket.Status = "closed"
		response.Data.PublicWebSocket.Message = "Public WebSocket connection is closed"
		response.Code = 503
	}
	response.Data.PublicWebSocket.Timestamp = time.Now().Unix()

	if privateWSStatus == 1 {
		response.Data.PrivateWebSocket.Status = "healthy"
		response.Data.PrivateWebSocket.Message = "Private WebSocket connection is active"
	} else {
		response.Data.PrivateWebSocket.Status = "closed"
		response.Data.PrivateWebSocket.Message = "Private WebSocket connection is closed"
		response.Code = 503
	}
	response.Data.PrivateWebSocket.Timestamp = time.Now().Unix()

	if businessWSStatus == 1 {
		response.Data.BusinessWebSocket.Status = "healthy"
		response.Data.BusinessWebSocket.Message = "Business WebSocket connection is active"
	} else {
		response.Data.BusinessWebSocket.Status = "closed"
		response.Data.BusinessWebSocket.Message = "Business WebSocket connection is closed"
		response.Code = 503
	}
	response.Data.BusinessWebSocket.Timestamp = time.Now().Unix()

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
