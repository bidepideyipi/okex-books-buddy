package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supermancell/okex-buddy/internal/config"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/wshub"
)

// Entry point for the API / monitoring backend service.
// Exposes HTTP APIs and WebSocket endpoint for the Vue dashboard.
func main() {
	cfg := config.LoadFromEnv()

	redisClient, err := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("Failed to close Redis client: %v", err)
		}
	}()

	// Create WebSocket hub
	hub := wshub.NewHub()
	go hub.Run()

	// Start background worker to push analysis updates via WebSocket
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Get list of active trading pairs from Redis
			pairs, err := redisClient.GetTradingPairs("trading_pairs:active")
			if err != nil {
				log.Printf("Failed to get active pairs: %v", err)
				continue
			}

			for _, instID := range pairs {
				// Fetch analysis data for each pair
				supportKey := fmt.Sprintf("analysis:support_resistance:%s", instID)
				largeKey := fmt.Sprintf("analysis:large_orders:%s", instID)

				supportHash, _ := redisClient.GetHash(supportKey)
				largeHash, _ := redisClient.GetHash(largeKey)

				if len(supportHash) > 0 || len(largeHash) > 0 {
					data := make(map[string]interface{})
					if len(supportHash) > 0 {
						data["support_resistance"] = supportHash
					}
					if len(largeHash) > 0 {
						data["large_orders"] = largeHash
					}

					// Broadcast to WebSocket clients
					hub.BroadcastAnalysisUpdate(instID, data)
				}
			}
		}
	}()

	mux := http.NewServeMux()

	// CORS middleware
	corsHandler := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// GET /api/analysis/{instId}
	// Returns current analysis snapshot for a given instrument from Redis.
	mux.HandleFunc("/api/analysis/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("{\"code\":405,\"message\":\"method not allowed\",\"data\":null}"))
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/analysis/")
		if path == "" || strings.Contains(path, "/") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("{\"code\":400,\"message\":\"invalid instrument id\",\"data\":null}"))
			return
		}
		instID := path

		supportKey := fmt.Sprintf("analysis:support_resistance:%s", instID)
		largeKey := fmt.Sprintf("analysis:large_orders:%s", instID)

		supportHash, err := redisClient.GetHash(supportKey)
		if err != nil {
			log.Printf("Failed to get support/resistance hash for %s: %v", instID, err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("{\"code\":500,\"message\":\"failed to fetch analysis\",\"data\":null}"))
			return
		}

		largeHash, err := redisClient.GetHash(largeKey)
		if err != nil {
			log.Printf("Failed to get large_orders hash for %s: %v", instID, err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("{\"code\":500,\"message\":\"failed to fetch analysis\",\"data\":null}"))
			return
		}

		if len(supportHash) == 0 && len(largeHash) == 0 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("{\"code\":404,\"message\":\"analysis not found\",\"data\":null}"))
			return
		}

		data := struct {
			InstrumentID      string            `json:"instrument_id"`
			SupportResistance map[string]string `json:"support_resistance,omitempty"`
			LargeOrders       map[string]string `json:"large_orders,omitempty"`
		}{
			InstrumentID: instID,
		}

		if len(supportHash) > 0 {
			data.SupportResistance = supportHash
		}
		if len(largeHash) > 0 {
			data.LargeOrders = largeHash
		}

		resp := struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data"`
		}{
			Code:    200,
			Message: "success",
			Data:    data,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode analysis response for %s: %v", instID, err)
		}
	}))

	// WebSocket endpoint for real-time updates
	mux.HandleFunc("/ws/analysis", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(w, r)
	})

	// GET /api/websocket/status
	// Returns status of all WebSocket connections (placeholder - will be implemented when websocket_client tracks connection status)
	mux.HandleFunc("/api/websocket/status", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("{\"code\":405,\"message\":\"method not allowed\",\"data\":null}"))
			return
		}

		// Placeholder response - will be implemented when websocket_client stores connection status in Redis
		resp := struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data"`
		}{
			Code:    200,
			Message: "success",
			Data: map[string]interface{}{
				"total_connections": 0,
				"active_pairs":      0,
				"connections":       []interface{}{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode websocket status response: %v", err)
		}
	}))

	// GET /api/websocket/status/{instId}
	// Returns status of WebSocket connection for a specific instrument (placeholder)
	mux.HandleFunc("/api/websocket/status/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("{\"code\":405,\"message\":\"method not allowed\",\"data\":null}"))
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/websocket/status/")
		if path == "" || strings.Contains(path, "/") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("{\"code\":400,\"message\":\"invalid instrument id\",\"data\":null}"))
			return
		}

		// Placeholder - return 404 for now
		w.WriteHeader(http.StatusNotFound)
		resp := struct {
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Data    interface{} `json:"data"`
		}{
			Code:    404,
			Message: "connection status not available yet",
			Data:    nil,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Failed to encode websocket status response: %v", err)
		}
	}))

	log.Printf("okex-buddy api server listening on %s", cfg.APIHTTPAddr)
	if err := http.ListenAndServe(cfg.APIHTTPAddr, mux); err != nil {
		log.Fatalf("API server exited: %v", err)
	}
}
