package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 定义颜色常量
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
)

// RedisKey
const (
	TradingPairsKey           = "config:trading_pairs"  //运行时会去订阅的交易对
	PublishOrderBookEventKey  = "list:orderbook:events" //订单薄事件
	PublishTradeSigleEventKey = "list:sigle:events"     //信号事件
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
	UseProxy    bool
	ProxyAddr   string
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
// If not set, it will try to load from config/app.env file.
func LoadFromEnv() AppConfig {
	// Try to load from config file if env vars are not set
	if os.Getenv("REDIS_ADDR") == "" {
		loadEnvFile("config/app.env")
	}

	return AppConfig{
		Redis: RedisConfig{
			Addr:            getenvWithDefault("REDIS_ADDR", "localhost:6379"),
			Password:        os.Getenv("REDIS_PASSWORD"),
			TradingPairsKey: getenvWithDefault("REDIS_TRADING_PAIRS_KEY", "config:trading_pairs"),
			PollIntervalSec: getenvIntWithDefault("TRADING_PAIRS_POLL_INTERVAL", 20),
		},
		OKEX: OKEXConfig{
			PublicWSURL: getenvWithDefault("OKEX_WS_PUBLIC", "wss://ws.okx.com:8443/ws/v5/public"),
			UseProxy:    getenvBoolWithDefault("USE_PROXY", false),
			ProxyAddr:   os.Getenv("PROXY_ADDR"),
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

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filePath string) {
	// Get absolute path
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		if wd, err := os.Getwd(); err == nil {
			// Try workspace root
			rootPath := filepath.Join(wd, "../..", filePath)
			if _, err := os.Stat(rootPath); err == nil {
				absPath = rootPath
			} else {
				// Try current directory
				curPath := filepath.Join(wd, filePath)
				if _, err := os.Stat(curPath); err == nil {
					absPath = curPath
				}
			}
		}
	}

	file, err := os.Open(absPath)
	if err != nil {
		return // File not found, use defaults or existing env vars
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only set if not already in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
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

func getenvBoolWithDefault(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
