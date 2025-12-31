package config

import (
	"os"
	"strconv"
)

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr            string
	Password        string
	TradingPairsKey string // Redis key for trading pairs configuration
	PollIntervalSec int    // Polling interval for config changes in seconds
}

// OKEXConfig holds OKEx WebSocket endpoint configuration.
type OKEXConfig struct {
	PublicWSURL string
}

// InfluxConfig holds InfluxDB 2.x connection configuration.
type InfluxConfig struct {
	URL      string
	Org      string
	Bucket   string
	Username string
	Password string
}

// AppConfig aggregates all runtime configuration needed by backend services.
type AppConfig struct {
	Redis             RedisConfig
	OKEX              OKEXConfig
	Influx            InfluxConfig
	APIHTTPAddr       string
	FrontendDevServer string
}

// LoadFromEnv loads configuration from environment variables.
// It is designed to work with the *.env files under the config/ directory
// (e.g. app.dev.env, influxdb.dev.env) once they are exported into the shell.
func LoadFromEnv() AppConfig {
	return AppConfig{
		Redis: RedisConfig{
			Addr:            getenvWithDefault("REDIS_ADDR", "localhost:6379"),
			Password:        os.Getenv("REDIS_PASSWORD"),
			TradingPairsKey: getenvWithDefault("REDIS_TRADING_PAIRS_KEY", "config:trading_pairs"),
			PollIntervalSec: getenvIntWithDefault("TRADING_PAIRS_POLL_INTERVAL", 20),
		},
		OKEX: OKEXConfig{
			PublicWSURL: getenvWithDefault("OKEX_WS_PUBLIC", "wss://ws.okx.com:8443/ws/v5/public"),
		},
		Influx: InfluxConfig{
			URL:      os.Getenv("INFLUX_URL"),
			Org:      os.Getenv("INFLUX_ORG"),
			Bucket:   os.Getenv("INFLUX_BUCKET"),
			Username: os.Getenv("INFLUX_USERNAME"),
			Password: os.Getenv("INFLUX_PASSWORD"),
		},
		APIHTTPAddr:       getenvWithDefault("API_HTTP_ADDR", "0.0.0.0:8080"),
		FrontendDevServer: os.Getenv("FRONTEND_DEV_SERVER"),
	}
}

func getenvWithDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvIntWithDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
