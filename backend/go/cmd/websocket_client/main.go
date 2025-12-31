package main

import (
	"fmt"

	"github.com/supermancell/okex-buddy/internal/config"
)

// Entry point for the WebSocket client service.
// Later milestones will add OKEx connection, order book handling, and Redis publishing.
func main() {
	cfg := config.LoadFromEnv()
	fmt.Println("okex-buddy websocket client service (M1 skeleton)")
	fmt.Printf("Config loaded: Redis=%s, OKEx WS=%s, Influx URL=%s\n",
		cfg.Redis.Addr, cfg.OKEX.PublicWSURL, cfg.Influx.URL)
}
