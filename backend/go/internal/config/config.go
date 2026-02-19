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
	TradingPairsKey      = "config:trading_pairs" //运行时会去订阅的交易对
	OrderBookKey         = "orderbook:%s"
	TickerKey            = "ticker:%s"
	SupportResistanceKey = "analysis:supp_resi:%s" //支撑位和阻力位
	SentimentKey         = "analysis:sentiment:%s" //多空情绪
	DepthAnomalyKey      = "analysis:dept_anom:%s" //深度异常波动
	LiquidityShrinkKey   = "analysis:liqu_shri:%s" //流动性萎缩预警
)

const (
	BooksChannel  = "books"   //订单薄频道
	TickerChannel = "tickers" //行情频道
	Candle1D      = "candle1D"
	Candle4H      = "candle4H"
	Candle1H      = "candle1H"
	Candle15m     = "candle15m"
)

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr            string
	Password        string
	TradingPairsKey string // Redis key for trading pairs configuration
	PollIntervalSec int    // Polling interval for config changes in seconds
}

// MongoDBConfig holds MongoDB connection settings.
type MongoDBConfig struct {
	Addr     string
	Database string
}

// OKEXConfig holds OKEx WebSocket endpoint configuration.
type OKEXConfig struct {
	PublicWSURL      string
	BusinessWSURL    string
	PrivateWSURL     string
	UseProxy         bool
	ProxyAddr        string
	HTTPProxyAddr    string
	EnablePublicWS   bool
	EnableBusinessWS bool
	EnablePrivateWS  bool
}

// AnalysisConfig holds configuration for analysis functions.
type AnalysisConfig struct {
	// ComputeSupportResistance
	SupportResistanceBinCount              int     // 价格区间划分数量
	SupportResistanceSignificanceThreshold float64 // 支撑/阻力位显著性阈值
	SupportResistanceTopN                  int     // 返回的支撑/阻力位数量
	SupportResistanceMinDistancePercent    float64 // 支撑/阻力位之间的最小价格差异百分比

	// ComputeLargeOrderDistribution
	LargeOrderPercentileAlpha            float64 // 大额订单的百分位数阈值
	LargeOrderDecayLambda                float64 // 价格距离衰减因子
	LargeOrderSentimentDeadzoneThreshold float64 // 情绪中性区间阈值

	// DetectDepthAnomaly
	DepthAnomalyPriceRangePercent float64 // 计算深度的价格范围百分比
	DepthAnomalyWindowSize        int     // 历史数据窗口大小
	DepthAnomalyZThreshold        float64 // Z分数异常阈值

	// DetectLiquidityShrinkage
	LiquidityShrinkNearPriceDeltaPercent float64 // 价格附近的百分比阈值
	LiquidityShrinkShortWindowSeconds    int     // 短期趋势窗口（秒）
	LiquidityShrinkLongWindowSeconds     int     // 长期基准窗口（秒）
	LiquidityShrinkSlopeThreshold        float64 // 流动性下降斜率阈值
}

// AppConfig aggregates all runtime configuration needed by backend services.
type AppConfig struct {
	Redis             RedisConfig
	MongoDB           MongoDBConfig
	OKEX              OKEXConfig
	Analysis          AnalysisConfig
	APIHTTPAddr       string
	FrontendDevServer string
}

// LoadFromEnv loads configuration from environment variables.
// If not set, it will try to load from config/app.env file.
func LoadFromEnv() AppConfig {
	loadEnvFile("config/app.env")

	return AppConfig{
		Redis: RedisConfig{
			Addr:            getenvWithDefault("REDIS_ADDR", "localhost:6379"),
			Password:        os.Getenv("REDIS_PASSWORD"),
			TradingPairsKey: getenvWithDefault("REDIS_TRADING_PAIRS_KEY", "config:trading_pairs"),
			PollIntervalSec: getenvIntWithDefault("TRADING_PAIRS_POLL_INTERVAL", 20),
		},
		MongoDB: MongoDBConfig{
			Addr:     getenvWithDefault("MONGODB_ADDR", "mongodb://127.0.0.1:27017"),
			Database: getenvWithDefault("MONGODB_DATABASE", "technical_analysis"),
		},
		OKEX: OKEXConfig{
			PublicWSURL:      getenvWithDefault("OKEX_WS_PUBLIC", "wss://ws.okx.com:8443/ws/v5/public"),
			BusinessWSURL:    getenvWithDefault("OKEX_WS_BUSINESS", "wss://ws.okx.com:8443/ws/v5/business"),
			PrivateWSURL:     getenvWithDefault("OKEX_WS_PRIVATE", "wss://ws.okx.com:8443/ws/v5/private"),
			UseProxy:         getenvBoolWithDefault("USE_PROXY", true),
			ProxyAddr:        getenvWithDefault("PROXY_ADDR", "127.0.0.1:4781"),
			HTTPProxyAddr:    getenvWithDefault("HTTP_PROXY_ADDR", "127.0.0.1:4780"),
			EnablePublicWS:   getenvBoolWithDefault("ENABLE_PUBLIC_WS", false),
			EnableBusinessWS: getenvBoolWithDefault("ENABLE_BUSINESS_WS", false),
			EnablePrivateWS:  getenvBoolWithDefault("ENABLE_PRIVATE_WS", true),
		},
		Analysis: AnalysisConfig{
			// ComputeSupportResistance
			SupportResistanceBinCount:              getenvIntWithDefault("SUPPORT_RESISTANCE_BIN_COUNT", 50),
			SupportResistanceSignificanceThreshold: getenvFloat64WithDefault("SUPPORT_RESISTANCE_SIGNIFICANCE_THRESHOLD", 1.5),
			SupportResistanceTopN:                  getenvIntWithDefault("SUPPORT_RESISTANCE_TOP_N", 2),
			SupportResistanceMinDistancePercent:    getenvFloat64WithDefault("SUPPORT_RESISTANCE_MIN_DISTANCE_PERCENT", 0.5),

			// ComputeLargeOrderDistribution
			LargeOrderPercentileAlpha:            getenvFloat64WithDefault("LARGE_ORDER_PERCENTILE_ALPHA", 0.95),
			LargeOrderDecayLambda:                getenvFloat64WithDefault("LARGE_ORDER_DECAY_LAMBDA", 5.0),
			LargeOrderSentimentDeadzoneThreshold: getenvFloat64WithDefault("LARGE_ORDER_SENTIMENT_DEADZONE_THRESHOLD", 0.3),

			// DetectDepthAnomaly
			DepthAnomalyPriceRangePercent: getenvFloat64WithDefault("DEPTH_ANOMALY_PRICE_RANGE_PERCENT", 0.5),
			DepthAnomalyWindowSize:        getenvIntWithDefault("DEPTH_ANOMALY_WINDOW_SIZE", 30),
			DepthAnomalyZThreshold:        getenvFloat64WithDefault("DEPTH_ANOMALY_Z_THRESHOLD", 2.0),

			// DetectLiquidityShrinkage
			LiquidityShrinkNearPriceDeltaPercent: getenvFloat64WithDefault("LIQUIDITY_SHRINK_NEAR_PRICE_DELTA_PERCENT", 0.5),
			LiquidityShrinkShortWindowSeconds:    getenvIntWithDefault("LIQUIDITY_SHRINK_SHORT_WINDOW_SECONDS", 30),
			LiquidityShrinkLongWindowSeconds:     getenvIntWithDefault("LIQUIDITY_SHRINK_LONG_WINDOW_SECONDS", 1800),
			LiquidityShrinkSlopeThreshold:        getenvFloat64WithDefault("LIQUIDITY_SHRINK_SLOPE_THRESHOLD", -0.01),
		},
		APIHTTPAddr:       getenvWithDefault("API_HTTP_ADDR", "0.0.0.0:8080"),
		FrontendDevServer: os.Getenv("FRONTEND_DEV_SERVER"),
	}
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filePath string) {
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
				} else {
					// Try one level up
					upPath := filepath.Join(wd, "..", filePath)
					if _, err := os.Stat(upPath); err == nil {
						absPath = upPath
					}
				}
			}
		}
	}

	file, err := os.Open(absPath)
	if err != nil {
		return
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

func getenvFloat64WithDefault(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
