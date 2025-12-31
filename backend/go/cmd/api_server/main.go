package main

import (
	"fmt"

	"github.com/supermancell/okex-buddy/internal/config"
)

// Entry point for the API / monitoring backend service.
// Later milestones will add HTTP/WS APIs to expose Redis & InfluxDB data.
func main() {
	cfg := config.LoadFromEnv()
	fmt.Println("okex-buddy api server service (M1 skeleton)")
	fmt.Printf("Config loaded: API bind=%s, Influx Org=%s, Bucket=%s\n",
		cfg.APIHTTPAddr, cfg.Influx.Org, cfg.Influx.Bucket)
}
