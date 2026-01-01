package orderbook

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"sort"
	"strconv"
	"strings"
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

// Manager manages order books for multiple instruments
type Manager struct {
	books map[string]*OrderBook // instrument_id -> order book
}

// NewManager creates a new order book manager
func NewManager() *Manager {
	return &Manager{
		books: make(map[string]*OrderBook),
	}
}

// ProcessMessage processes incoming WebSocket messages
func (m *Manager) ProcessMessage(msg []byte) error {
	var okexMsg OKExMessage
	if err := json.Unmarshal(msg, &okexMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Handle subscription confirmation
	if okexMsg.Event == "subscribe" {
		// Extract instID from arg field for logging
		var arg ArgData
		if err := json.Unmarshal(okexMsg.Arg, &arg); err == nil {
			log.Printf("Subscription confirmed: %s", arg.InstID)
		} else {
			log.Printf("Subscription confirmed (failed to parse instID)")
		}
		return nil
	}

	// Handle error messages
	if okexMsg.Event == "error" {
		return fmt.Errorf("OKEx error: code=%s, msg=%s", okexMsg.Code, okexMsg.Msg)
	}

	// Process order book data
	if len(okexMsg.Data) == 0 {
		return nil // No data to process
	}

	// Extract instID from arg field
	var arg ArgData
	if err := json.Unmarshal(okexMsg.Arg, &arg); err != nil {
		log.Printf("WARNING: Failed to parse arg field: %v", err)
		return nil
	}

	// Process each data item with the instID from arg
	for _, data := range okexMsg.Data {
		// Use instID from arg field (this is where OKX puts it)
		data.InstID = arg.InstID

		if err := m.updateOrderBook(data, okexMsg.Action); err != nil {
			return fmt.Errorf("failed to update order book for %s: %w", data.InstID, err)
		}
	}

	return nil
}

// updateOrderBook updates the order book based on snapshot or incremental data
func (m *Manager) updateOrderBook(data BookData, action string) error {
	ts, err := strconv.ParseInt(data.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	// Handle snapshot (full order book)
	if action == "snapshot" || action == "" {
		book := &OrderBook{
			InstrumentID: data.InstID,
			Timestamp:    ts,
			Checksum:     data.Checksum,
		}

		// Parse asks
		book.Asks = make([]PriceLevel, 0, len(data.Asks))
		for _, ask := range data.Asks {
			if len(ask) < 2 {
				continue
			}
			book.Asks = append(book.Asks, PriceLevel{
				Price: ask[0],
				Size:  ask[1],
			})
		}

		// Parse bids
		book.Bids = make([]PriceLevel, 0, len(data.Bids))
		for _, bid := range data.Bids {
			if len(bid) < 2 {
				continue
			}
			book.Bids = append(book.Bids, PriceLevel{
				Price: bid[0],
				Size:  bid[1],
			})
		}

		// IMPORTANT: Sort the data after parsing
		// Asks should be sorted ascending by price
		m.sortLevels(&book.Asks, true)
		// Bids should be sorted descending by price
		m.sortLevels(&book.Bids, false)

		// Store the order book
		m.books[data.InstID] = book

		// Verify checksum (log warning but don't fail)
		if err := m.verifyChecksum(data.InstID); err != nil {
			log.Printf("WARNING: %v (continuing anyway)", err)
			// Don't return error - just log and continue
		}

		return nil
	}

	// Handle incremental update
	if action == "update" {
		book, exists := m.books[data.InstID]
		if !exists {
			return fmt.Errorf("order book not initialized for %s", data.InstID)
		}

		book.Timestamp = ts

		// Update asks
		for _, ask := range data.Asks {
			if len(ask) < 2 {
				continue
			}
			m.updateLevel(&book.Asks, ask[0], ask[1], true)
		}

		// Update bids
		for _, bid := range data.Bids {
			if len(bid) < 2 {
				continue
			}
			m.updateLevel(&book.Bids, bid[0], bid[1], false)
		}

		// Trim to top 400 levels
		if len(book.Asks) > 400 {
			book.Asks = book.Asks[:400]
		}
		if len(book.Bids) > 400 {
			book.Bids = book.Bids[:400]
		}

		book.Checksum = data.Checksum

		// Verify checksum (log warning but don't fail)
		if err := m.verifyChecksum(data.InstID); err != nil {
			log.Printf("WARNING: %v (continuing anyway)", err)
			// Don't return error - just log and continue
		}

		return nil
	}

	return fmt.Errorf("unknown action: %s", action)
}

// updateLevel updates a single price level
func (m *Manager) updateLevel(levels *[]PriceLevel, price, size string, isAsk bool) {
	sizeFloat, err := strconv.ParseFloat(size, 64)
	if err != nil || sizeFloat == 0 {
		// Remove this level if size is 0 or invalid
		m.removeLevel(levels, price)
		return
	}

	// Find and update or insert new level
	found := false
	for i, level := range *levels {
		if level.Price == price {
			(*levels)[i].Size = size
			found = true
			break
		}
	}

	if !found {
		// Insert new level and sort
		*levels = append(*levels, PriceLevel{
			Price: price,
			Size:  size,
		})
		m.sortLevels(levels, isAsk)
	}
}

// removeLevel removes a price level
func (m *Manager) removeLevel(levels *[]PriceLevel, price string) {
	for i, level := range *levels {
		if level.Price == price {
			*levels = append((*levels)[:i], (*levels)[i+1:]...)
			return
		}
	}
}

// sortLevels sorts price levels
func (m *Manager) sortLevels(levels *[]PriceLevel, isAsk bool) {
	sort.Slice(*levels, func(i, j int) bool {
		pi, _ := strconv.ParseFloat((*levels)[i].Price, 64)
		pj, _ := strconv.ParseFloat((*levels)[j].Price, 64)
		if isAsk {
			return pi < pj // asks ascending
		}
		return pi > pj // bids descending
	})
}

// verifyChecksum verifies the order book checksum according to OKEx specification
// Reference: https://www.okx.com/docs-v5/en/#overview-websocket-books-channel
// Checksum calculation:
//  1. When both bids and asks have >= 25 levels:
//     Interleave: bid1:ask1:bid2:ask2:...:bid25:ask25
//  2. When either has < 25 levels:
//     Continue with available data, ignore missing levels
func (m *Manager) verifyChecksum(instID string) error {
	book, exists := m.books[instID]
	if !exists {
		return fmt.Errorf("order book not found for %s", instID)
	}

	// Build checksum string according to OKEx spec
	var parts []string

	// Determine how many levels to include (max 25)
	maxLevels := 25
	bidCount := len(book.Bids)
	askCount := len(book.Asks)

	// Calculate actual max levels (take minimum of available and 25)
	maxBids := bidCount
	if maxBids > maxLevels {
		maxBids = maxLevels
	}
	maxAsks := askCount
	if maxAsks > maxLevels {
		maxAsks = maxLevels
	}

	// Use the larger of the two for interleaving
	iterCount := maxBids
	if maxAsks > iterCount {
		iterCount = maxAsks
	}

	// Interleave bids and asks: bid[price:size]:ask[price:size]:...
	for i := 0; i < iterCount; i++ {
		// Add bid if available
		if i < maxBids {
			parts = append(parts, book.Bids[i].Price)
			parts = append(parts, book.Bids[i].Size)
		}
		// Add ask if available
		if i < maxAsks {
			parts = append(parts, book.Asks[i].Price)
			parts = append(parts, book.Asks[i].Size)
		}
	}

	// Calculate CRC32 checksum
	checksumStr := strings.Join(parts, ":")
	calculated := int32(crc32.ChecksumIEEE([]byte(checksumStr)))

	if calculated != book.Checksum {
		// Log first few levels for debugging
		log.Printf("Checksum mismatch for %s:", instID)
		log.Printf("  Calculated: %d, Expected: %d", calculated, book.Checksum)
		log.Printf("  Checksum string (first 200 chars): %s", checksumStr[:min(200, len(checksumStr))])
		log.Printf("  Bids count: %d (using %d), Asks count: %d (using %d)", bidCount, maxBids, askCount, maxAsks)
		if maxBids > 0 {
			log.Printf("  First bid: %s @ %s", book.Bids[0].Size, book.Bids[0].Price)
		}
		if maxAsks > 0 {
			log.Printf("  First ask: %s @ %s", book.Asks[0].Size, book.Asks[0].Price)
		}
		return fmt.Errorf("checksum mismatch: calculated=%d, expected=%d, instID=%s", calculated, book.Checksum, instID)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetOrderBook returns the order book for an instrument
func (m *Manager) GetOrderBook(instID string) (*OrderBook, bool) {
	book, exists := m.books[instID]
	return book, exists
}

// GetTop400 returns the top 400 levels of asks and bids
func (m *Manager) GetTop400(instID string) (asks, bids []PriceLevel, err error) {
	book, exists := m.books[instID]
	if !exists {
		return nil, nil, fmt.Errorf("order book not found for %s", instID)
	}

	askCount := 400
	if len(book.Asks) < askCount {
		askCount = len(book.Asks)
	}
	asks = book.Asks[:askCount]

	bidCount := 400
	if len(book.Bids) < bidCount {
		bidCount = len(book.Bids)
	}
	bids = book.Bids[:bidCount]

	return asks, bids, nil
}
