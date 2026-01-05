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
	Data   []BookData      `json:"data"`
	Code   string          `json:"code"`
	Msg    string          `json:"msg"`
	Action string          `json:"action"`
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
