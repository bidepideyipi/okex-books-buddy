package orderbook

import (
	"encoding/json"
)

// OrderBook represents the order book for a trading pair
type OrderBook struct {
	InstrumentID string
	Timestamp    int64
	Asks         []PriceLevel // sorted ascending by price
	Bids         []PriceLevel // sorted descending by price
	Checksum     int32
}

// PriceLevel represents a single price level with price and size
type PriceLevel struct {
	Price      string
	Size       string
	OrderCount int
}

// OKExMessage represents the WebSocket message from OKEx
type OKExMessage struct {
	Event  string          `json:"event"`
	Arg    json.RawMessage `json:"arg"`
	Data   json.RawMessage `json:"data"` // Raw data - will be parsed based on channel type
	Code   string          `json:"code"`
	Msg    string          `json:"msg"`
	Action string          `json:"action"`
}

// TickerData represents the ticker data structure
type TickerData struct {
	InstType  string `json:"instType"`
	InstID    string `json:"instId"`
	Last      string `json:"last"`
	LastSz    string `json:"lastSz"`
	AskPx     string `json:"askPx"`
	AskSz     string `json:"askSz"`
	BidPx     string `json:"bidPx"`
	BidSz     string `json:"bidSz"`
	Open24h   string `json:"open24h"`
	High24h   string `json:"high24h"`
	Low24h    string `json:"low24h"`
	VolCcy24h string `json:"volCcy24h"`
	Vol24h    string `json:"vol24h"`
	SodUtc0   string `json:"sodUtc0"`
	SodUtc8   string `json:"sodUtc8"`
	Timestamp string `json:"ts"`
}

// ArgData represents the arg field structure
type ArgData struct {
	Channel string `json:"channel"`
	InstID  string `json:"instId"`
}

// BookData represents the order book data
type BookData struct {
	Asks      [][]string `json:"asks"`
	Bids      [][]string `json:"bids"`
	Timestamp string     `json:"ts"`
	Checksum  int32      `json:"checksum"`
	InstID    string     `json:"instId"`
}

// PriceLevelWithTime represents a price level with timestamp for sliding window calculations
type PriceLevelWithTime struct {
	PriceLevel PriceLevel
	Timestamp  int64
}

// PriceLevelWithTimeItem represents an item in the sliding window of sentiment values
type PriceLevelWithTimeItem struct {
	Value     float64
	Timestamp int64
}

// DepthAnomalyData represents the depth anomaly detection result
type DepthAnomalyData struct {
	Anomaly   bool    `json:"anomaly"`
	ZScore    float64 `json:"z_score"`
	Depth     float64 `json:"depth"`
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"std_dev"`
	Timestamp int64   `json:"timestamp"`
	Direction string  `json:"direction"` // "increase" or "decrease"
	Intensity float64 `json:"intensity"`
}

// DepthWindowItem represents an item in the depth sliding window
type DepthWindowItem struct {
	Depth     float64
	Timestamp int64
}

// SupportResistanceData represents support and resistance levels
type SupportResistanceData struct {
	Supports    []float64 `json:"supports"`
	Resistances []float64 `json:"resistances"`
	Spread      float64   `json:"spread"`
	Timestamp   int64     `json:"timestamp"`
}

// ToRedisMap converts SupportResistanceData to a map for Redis storage
func (s SupportResistanceData) ToRedisMap() map[string]interface{} {
	fields := map[string]interface{}{
		"timestamp": s.Timestamp,
		"spread":    s.Spread,
	}

	if len(s.Supports) > 0 {
		fields["support_high"] = s.Supports[0]
	}
	if len(s.Supports) > 1 {
		fields["support_low"] = s.Supports[1]
	}
	if len(s.Resistances) > 0 {
		fields["resistance_high"] = s.Resistances[0]
	}
	if len(s.Resistances) > 1 {
		fields["resistance_low"] = s.Resistances[1]
	}

	return fields
}

// SupportResistanceWindowItem represents an item in the support/resistance sliding window
type SupportResistanceWindowItem struct {
	Data      SupportResistanceData
	Timestamp int64
}

// LiquidityMetrics represents the liquidity metrics at a point in time
type LiquidityMetrics struct {
	Spread    float64 `json:"spread"`
	Depth     float64 `json:"depth"`
	Liquidity float64 `json:"liquidity"`
	Timestamp int64   `json:"timestamp"`
}

// LiquidityShrinkData represents the liquidity shrinkage warning result
type LiquidityShrinkData struct {
	Warning      bool    `json:"warning"`
	WarningLevel string  `json:"warning_level"` // "none", "light", "moderate", "severe"
	Liquidity    float64 `json:"liquidity"`
	Spread       float64 `json:"spread"`
	Depth        float64 `json:"depth"`
	Slope        float64 `json:"slope"`
	Timestamp    int64   `json:"timestamp"`
}

// LiquidityWindowItem represents an item in the liquidity sliding window
type LiquidityWindowItem struct {
	Metrics   LiquidityMetrics
	Timestamp int64
}

// SpreadWindowItem represents an item in the spread sliding window
type SpreadWindowItem struct {
	Spread    float64
	Timestamp int64
}

// Implement TimeWindowItem interface for all window items
func (i *PriceLevelWithTimeItem) GetTimestamp() int64 {
	return i.Timestamp
}

func (i *DepthWindowItem) GetTimestamp() int64 {
	return i.Timestamp
}

func (i *SupportResistanceWindowItem) GetTimestamp() int64 {
	return i.Timestamp
}

func (i *LiquidityWindowItem) GetTimestamp() int64 {
	return i.Timestamp
}

func (i *SpreadWindowItem) GetTimestamp() int64 {
	return i.Timestamp
}
