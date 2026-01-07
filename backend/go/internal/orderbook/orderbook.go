package orderbook

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Manager manages order books for multiple instruments
type Manager struct {
	books                    map[string]*OrderBook                    // instrument_id -> order book
	sentimentMap             map[string][]PriceLevelWithTimeItem      // instrument_id -> sliding window of sentiment values
	depthWindows             map[string][]DepthWindowItem             // instrument_id -> sliding window of depth values
	liquidityWindows         map[string][]LiquidityWindowItem         // instrument_id -> sliding window of liquidity metrics
	supportResistanceWindows map[string][]SupportResistanceWindowItem // instrument_id -> sliding window of support/resistance levels
}

// NewManager creates a new order book manager
func NewManager() *Manager {
	return &Manager{
		books:                    make(map[string]*OrderBook),
		sentimentMap:             make(map[string][]PriceLevelWithTimeItem),
		depthWindows:             make(map[string][]DepthWindowItem),
		liquidityWindows:         make(map[string][]LiquidityWindowItem),
		supportResistanceWindows: make(map[string][]SupportResistanceWindowItem),
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

// ComputeSupportResistance computes support and resistance levels for a given instrument
// based on the current in-memory order book.
// The implementation is a simplified version of the PRD algorithm:
//   - price range is divided into bins
//   - per-bin notional volume is accumulated
//   - local maxima above a significance threshold are selected and sorted
//     TODO 支撑位和阻力位之间的间隔太近了，没有什么实际意义
func (m *Manager) ComputeSupportResistance(instID string, binCount int, significanceThreshold float64, topN int, minDistancePercent float64) (supports, resistances []float64, err error) {
	// First, compute the current support and resistance levels
	asks, bids, err := m.GetTop400(instID)
	if err != nil {
		return nil, nil, err
	}

	if len(asks) == 0 && len(bids) == 0 {
		return nil, nil, fmt.Errorf("empty order book for %s", instID)
	}

	if binCount <= 0 {
		binCount = 50
	}
	if topN <= 0 {
		topN = 2
	}
	if significanceThreshold <= 0 {
		significanceThreshold = 1.5
	}
	if minDistancePercent <= 0 {
		minDistancePercent = 0.5
	}

	// Determine price range from bids and asks
	minPrice := 0.0
	maxPrice := 0.0
	first := true

	updateRange := func(levels []PriceLevel) {
		for _, lvl := range levels {
			p, err := strconv.ParseFloat(lvl.Price, 64)
			if err != nil {
				continue
			}
			if first {
				minPrice, maxPrice = p, p
				first = false
			} else {
				if p < minPrice {
					minPrice = p
				}
				if p > maxPrice {
					maxPrice = p
				}
			}
		}
	}

	updateRange(bids)
	updateRange(asks)

	if first || maxPrice <= minPrice {
		return nil, nil, fmt.Errorf("invalid price range for %s", instID)
	}

	binWidth := (maxPrice - minPrice) / float64(binCount)
	if binWidth <= 0 {
		return nil, nil, fmt.Errorf("invalid bin width for %s", instID)
	}

	bidVolumes := make([]float64, binCount)
	askVolumes := make([]float64, binCount)

	// Accumulate notional by bin for bids and asks
	accumulate := func(levels []PriceLevel, vols []float64) {
		for _, lvl := range levels {
			p, err1 := strconv.ParseFloat(lvl.Price, 64)
			q, err2 := strconv.ParseFloat(lvl.Size, 64)
			if err1 != nil || err2 != nil || q <= 0 {
				continue
			}
			notional := p * q
			idx := int((p - minPrice) / binWidth)
			if idx < 0 {
				idx = 0
			}
			if idx >= binCount {
				idx = binCount - 1
			}
			vols[idx] += notional
		}
	}

	accumulate(bids, bidVolumes)
	accumulate(asks, askVolumes)

	// Helper to find peaks
	findPeaks := func(vols []float64) []struct {
		Index int
		Value float64
	} {
		peaks := make([]struct {
			Index int
			Value float64
		}, 0)

		if len(vols) < 3 {
			return peaks
		}

		// Compute average volume
		total := 0.0
		for _, v := range vols {
			total += v
		}
		avg := total / float64(len(vols))

		for i := 1; i < len(vols)-1; i++ {
			v := vols[i]
			if v <= 0 {
				continue
			}
			if v > significanceThreshold*avg && v > vols[i-1] && v > vols[i+1] {
				peaks = append(peaks, struct {
					Index int
					Value float64
				}{Index: i, Value: v})
			}
		}

		// Fallback: if no peaks, pick top bins by volume
		if len(peaks) == 0 {
			for i, v := range vols {
				if v > 0 {
					peaks = append(peaks, struct {
						Index int
						Value float64
					}{Index: i, Value: v})
				}
			}
		}

		// Sort peaks by volume descending
		sort.Slice(peaks, func(i, j int) bool {
			return peaks[i].Value > peaks[j].Value
		})

		return peaks
	}

	bidPeaks := findPeaks(bidVolumes)
	askPeaks := findPeaks(askVolumes)

	binCenter := func(idx int) float64 {
		return minPrice + (float64(idx)+0.5)*binWidth
	}

	// Collect top-N support levels from bids with minimum distance filtering
	for i := 0; i < len(bidPeaks) && len(supports) < topN; i++ {
		candidate := binCenter(bidPeaks[i].Index)
		// Check distance from all existing supports
		tooClose := false
		for _, existing := range supports {
			diffPercent := math.Abs((candidate-existing)/existing) * 100
			if diffPercent < minDistancePercent {
				tooClose = true
				break
			}
		}
		if !tooClose {
			supports = append(supports, candidate)
		}
	}

	// Collect top-N resistance levels from asks with minimum distance filtering
	for i := 0; i < len(askPeaks) && len(resistances) < topN; i++ {
		candidate := binCenter(askPeaks[i].Index)
		// Check distance from all existing resistances
		tooClose := false
		for _, existing := range resistances {
			diffPercent := math.Abs((candidate-existing)/existing) * 100
			if diffPercent < minDistancePercent {
				tooClose = true
				break
			}
		}
		if !tooClose {
			resistances = append(resistances, candidate)
		}
	}

	// Add current result to sliding window for historical tracking
	currentTime := time.Now().Unix()
	m.supportResistanceWindows[instID] = append(m.supportResistanceWindows[instID], SupportResistanceWindowItem{
		Data: SupportResistanceData{
			Supports:    supports,
			Resistances: resistances,
			Timestamp:   currentTime,
		},
		Timestamp: currentTime,
	})

	// Keep only the most recent entries within a time window (e.g., 30 minutes)
	const maxWindowSeconds = 1800 // 30 minutes
	cutoffTime := currentTime - maxWindowSeconds
	startIndex := 0
	for i, item := range m.supportResistanceWindows[instID] {
		if item.Timestamp > cutoffTime {
			startIndex = i
			break
		}
	}
	m.supportResistanceWindows[instID] = m.supportResistanceWindows[instID][startIndex:]

	//log.Printf("Computed support and resistance levels for %s: supports=%v, resistances=%v", instID, supports, resistances)
	return supports, resistances, nil
}

// ComputeLargeOrderDistribution computes large order distribution and sentiment
// for a given instrument based on the current in-memory order book.
// It implements a simplified version of PRD 3.3.2:
//   - compute notional p*q for each price level
//   - determine dynamic threshold by percentile
//   - apply distance-based exponential decay weighting
//   - aggregate weighted notional for bids (BullPower) and asks (BearPower)
//   - apply sliding window smoothing to sentiment values
func (m *Manager) ComputeLargeOrderDistribution(instID string, percentileAlpha float64, decayLambda float64, sentimentDeadzoneThreshold float64) (largeBuyNotional, largeSellNotional, sentiment float64, err error) {
	asks, bids, err := m.GetTop400(instID)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(asks) == 0 && len(bids) == 0 {
		return 0, 0, 0, fmt.Errorf("empty order book for %s", instID)
	}

	// Determine mid price from best bid / best ask
	if len(bids) == 0 || len(asks) == 0 {
		return 0, 0, 0, fmt.Errorf("cannot compute mid price for %s: missing bids or asks", instID)
	}

	bestBid, err1 := strconv.ParseFloat(bids[0].Price, 64)
	bestAsk, err2 := strconv.ParseFloat(asks[0].Price, 64)
	if err1 != nil || err2 != nil || bestBid <= 0 || bestAsk <= 0 {
		return 0, 0, 0, fmt.Errorf("invalid best bid/ask for %s", instID)
	}

	mid := (bestBid + bestAsk) / 2.0
	if mid <= 0 {
		return 0, 0, 0, fmt.Errorf("invalid mid price for %s", instID)
	}

	// Collect notionals for percentile threshold
	var notionals []float64

	appendNotionals := func(levels []PriceLevel) {
		for _, lvl := range levels {
			p, err1 := strconv.ParseFloat(lvl.Price, 64)
			q, err2 := strconv.ParseFloat(lvl.Size, 64)
			if err1 != nil || err2 != nil || q <= 0 {
				continue
			}
			n := p * q
			if n <= 0 {
				continue
			}
			notionals = append(notionals, n)
		}
	}

	appendNotionals(bids)
	appendNotionals(asks)

	if len(notionals) == 0 {
		// No meaningful orders; treat as no large orders
		return 0, 0, 0, nil
	}

	if percentileAlpha <= 0 || percentileAlpha >= 1 {
		percentileAlpha = 0.95
	}
	if decayLambda <= 0 {
		decayLambda = 5.0
	}
	if sentimentDeadzoneThreshold <= 0 {
		sentimentDeadzoneThreshold = 0.3
	}

	sort.Float64s(notionals)

	idx := int(math.Floor(percentileAlpha * float64(len(notionals)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(notionals) {
		idx = len(notionals) - 1
	}
	threshold := notionals[idx]

	var bullPower, bearPower float64

	// Helper to process one side
	processSide := func(levels []PriceLevel, isBid bool) {
		for _, lvl := range levels {
			p, err1 := strconv.ParseFloat(lvl.Price, 64)
			q, err2 := strconv.ParseFloat(lvl.Size, 64)
			if err1 != nil || err2 != nil || q <= 0 {
				continue
			}
			n := p * q
			if n <= threshold {
				continue
			}

			// Distance-based weight relative to mid price
			w := math.Exp(-decayLambda * math.Abs(p-mid) / mid)

			if isBid {
				largeBuyNotional += n
				bullPower += n * w
			} else {
				largeSellNotional += n
				bearPower += n * w
			}
		}
	}

	processSide(bids, true)
	processSide(asks, false)

	totalPower := bullPower + bearPower
	if totalPower == 0 {
		return largeBuyNotional, largeSellNotional, 0, nil
	}

	// Compute raw sentiment
	rawSentiment := (bullPower - bearPower) / totalPower

	// Non-linear sentiment transformation with deadzone threshold
	var transformedSentiment float64
	if math.Abs(rawSentiment) <= sentimentDeadzoneThreshold {
		// Within deadzone: linear transformation to 0-0.3 range
		transformedSentiment = (rawSentiment / sentimentDeadzoneThreshold) * 0.3
	} else {
		// Outside deadzone: scale remaining sentiment to 0.3-1.0 range
		baseSentiment := math.Copysign(sentimentDeadzoneThreshold, rawSentiment)
		remainingSentiment := rawSentiment - baseSentiment
		transformedSentiment = baseSentiment*0.3 + (remainingSentiment/(1-sentimentDeadzoneThreshold))*0.7
	}

	// Apply sliding window smoothing to sentiment values (30-second window)
	currentTime := time.Now().Unix()
	windowDuration := int64(30) // 30 seconds

	// Add current sentiment to the sliding window
	m.sentimentMap[instID] = append(m.sentimentMap[instID], PriceLevelWithTimeItem{Value: transformedSentiment, Timestamp: currentTime})

	// Remove old entries outside the 30-second window
	newWindow := []PriceLevelWithTimeItem{}
	for _, entry := range m.sentimentMap[instID] {
		if currentTime-entry.Timestamp <= windowDuration {
			newWindow = append(newWindow, entry)
		}
	}
	m.sentimentMap[instID] = newWindow

	// Calculate smoothed sentiment as average of values in the window
	if len(m.sentimentMap[instID]) > 0 {
		var sum float64
		for _, entry := range m.sentimentMap[instID] {
			sum += entry.Value
		}
		sentiment = sum / float64(len(m.sentimentMap[instID]))
	} else {
		sentiment = transformedSentiment
	}

	return largeBuyNotional, largeSellNotional, sentiment, nil
}

// CalculateDepthInRange calculates the total depth within a given price range around the mid price
func (m *Manager) CalculateDepthInRange(instID string, priceRangePercent float64) (float64, error) {
	asks, bids, err := m.GetTop400(instID)
	if err != nil {
		return 0, err
	}

	if len(asks) == 0 || len(bids) == 0 {
		return 0, fmt.Errorf("insufficient data for %s: need both asks and bids", instID)
	}

	// Calculate mid price
	bestBid, err1 := strconv.ParseFloat(bids[0].Price, 64)
	bestAsk, err2 := strconv.ParseFloat(asks[0].Price, 64)
	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("invalid best bid/ask prices for %s", instID)
	}

	midPrice := (bestBid + bestAsk) / 2.0
	if midPrice <= 0 {
		return 0, fmt.Errorf("invalid mid price for %s", instID)
	}

	// Calculate price range boundaries
	priceRange := midPrice * priceRangePercent / 100.0 // Convert percentage to absolute value
	minPrice := midPrice - priceRange
	maxPrice := midPrice + priceRange

	var totalDepth float64

	// Calculate depth for bids (prices >= minPrice and <= maxPrice)
	for _, bid := range bids {
		bidPrice, err := strconv.ParseFloat(bid.Price, 64)
		if err != nil {
			continue
		}
		if bidPrice >= minPrice && bidPrice <= maxPrice {
			bidSize, err := strconv.ParseFloat(bid.Size, 64)
			if err != nil {
				continue
			}
			// Depth = price * quantity for notional value
			totalDepth += bidPrice * bidSize
		}
	}

	// Calculate depth for asks (prices >= minPrice and <= maxPrice)
	for _, ask := range asks {
		askPrice, err := strconv.ParseFloat(ask.Price, 64)
		if err != nil {
			continue
		}
		if askPrice >= minPrice && askPrice <= maxPrice {
			askSize, err := strconv.ParseFloat(ask.Size, 64)
			if err != nil {
				continue
			}
			// Depth = price * quantity for notional value
			totalDepth += askPrice * askSize
		}
	}

	return totalDepth, nil
}

// DetectDepthAnomaly detects anomalies in the order book depth using Z-score
// 检测订单簿深度异常情况，使用Z分数
func (m *Manager) DetectDepthAnomaly(instID string, priceRangePercent float64, windowSize int, zThreshold float64) (*DepthAnomalyData, error) {
	// Calculate current depth in the specified range
	// 计算指定价格范围内的当前深度
	currentDepth, err := m.CalculateDepthInRange(instID, priceRangePercent)
	if err != nil {
		return nil, err
	}

	// Set default parameters if invalid
	if windowSize <= 0 {
		windowSize = 30 // Default to 30 data points
	}
	if zThreshold <= 0 {
		zThreshold = 2.0 // Default to 2.0 standard deviations
	}

	currentTime := time.Now().Unix()

	// Add current depth to the sliding window
	m.depthWindows[instID] = append(m.depthWindows[instID], DepthWindowItem{
		Depth:     currentDepth,
		Timestamp: currentTime,
	})

	// Keep only the most recent windowSize entries
	if len(m.depthWindows[instID]) > windowSize {
		startIndex := len(m.depthWindows[instID]) - windowSize
		m.depthWindows[instID] = m.depthWindows[instID][startIndex:]
	}

	// If we don't have enough data points yet, return normal state
	if len(m.depthWindows[instID]) < 2 {
		return &DepthAnomalyData{
			Anomaly:   false,
			ZScore:    0,
			Depth:     currentDepth,
			Mean:      currentDepth,
			StdDev:    0,
			Timestamp: currentTime,
			Direction: "", // Not enough data to determine direction
			Intensity: 0,
		}, nil
	}

	// Calculate mean of historical depths
	// 计算历史深度的平均值
	var sum float64
	for _, item := range m.depthWindows[instID][:len(m.depthWindows[instID])-1] { // Exclude current value from mean
		sum += item.Depth
	}
	historicalMean := sum / float64(len(m.depthWindows[instID])-1)

	// Calculate standard deviation
	var sumSquares float64
	for _, item := range m.depthWindows[instID][:len(m.depthWindows[instID])-1] { // Exclude current value
		deviation := item.Depth - historicalMean
		sumSquares += deviation * deviation
	}
	stdDev := math.Sqrt(sumSquares / float64(len(m.depthWindows[instID])-1))

	// Calculate Z-score
	zScore := 0.0
	if stdDev > 0 {
		zScore = (currentDepth - historicalMean) / stdDev
	}

	// Determine if it's an anomaly
	isAnomaly := math.Abs(zScore) > zThreshold

	// Determine direction and intensity
	direction := ""
	if zScore > zThreshold {
		direction = "increase" // Depth is significantly higher than normal 深度显著高于正常水平
	} else if zScore < -zThreshold {
		direction = "decrease" // Depth is significantly lower than normal 深度显著低于正常水平
	}

	intensity := math.Abs(zScore)

	result := &DepthAnomalyData{
		Anomaly:   isAnomaly,
		ZScore:    zScore,
		Depth:     currentDepth,
		Mean:      historicalMean,
		StdDev:    stdDev,
		Timestamp: currentTime,
		Direction: direction,
		Intensity: intensity,
	}

	return result, nil
}

// ToRedisMap converts DepthAnomalyData to a map for Redis storage
func (d *DepthAnomalyData) ToRedisMap() map[string]interface{} {
	return map[string]interface{}{
		"anomaly":   d.Anomaly,
		"z_score":   d.ZScore,
		"depth":     d.Depth,
		"mean":      d.Mean,
		"std_dev":   d.StdDev,
		"direction": d.Direction,
		"intensity": d.Intensity,
		"timestamp": d.Timestamp,
	}
}

// CalculateLiquidityMetrics calculates the liquidity metrics for an instrument
// 计算INST的流动性指标
func (m *Manager) CalculateLiquidityMetrics(instID string, nearPriceDeltaPercent float64) (*LiquidityMetrics, error) {
	asks, bids, err := m.GetTop400(instID)
	if err != nil {
		return nil, err
	}

	if len(asks) == 0 || len(bids) == 0 {
		return nil, fmt.Errorf("insufficient data for %s: need both asks and bids", instID)
	}

	// Calculate mid price
	bestBidPrice, err1 := strconv.ParseFloat(bids[0].Price, 64)
	bestAskPrice, err2 := strconv.ParseFloat(asks[0].Price, 64)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("invalid best bid/ask prices for %s", instID)
	}

	midPrice := (bestBidPrice + bestAskPrice) / 2.0
	if midPrice <= 0 {
		return nil, fmt.Errorf("invalid mid price for %s", instID)
	}

	// Calculate spread
	effectiveSpread := (bestAskPrice - bestBidPrice) / midPrice

	// Calculate near-price depth
	priceRange := midPrice * nearPriceDeltaPercent / 100.0
	minPrice := midPrice - priceRange
	maxPrice := midPrice + priceRange

	var totalDepth float64

	// Calculate depth for bids (prices >= minPrice and <= maxPrice)
	for _, bid := range bids {
		bidPrice, err := strconv.ParseFloat(bid.Price, 64)
		if err != nil {
			continue
		}
		if bidPrice >= minPrice && bidPrice <= maxPrice {
			bidSize, err := strconv.ParseFloat(bid.Size, 64)
			if err != nil {
				continue
			}
			totalDepth += bidSize // Using quantity, not notional value for depth
		}
	}

	// Calculate depth for asks (prices >= minPrice and <= maxPrice)
	for _, ask := range asks {
		askPrice, err := strconv.ParseFloat(ask.Price, 64)
		if err != nil {
			continue
		}
		if askPrice >= minPrice && askPrice <= maxPrice {
			askSize, err := strconv.ParseFloat(ask.Size, 64)
			if err != nil {
				continue
			}
			totalDepth += askSize // Using quantity, not notional value for depth
		}
	}

	// Calculate composite liquidity metric
	liquidity := totalDepth / (1 + effectiveSpread)

	currentTime := time.Now().Unix()

	return &LiquidityMetrics{
		Spread:    effectiveSpread,
		Depth:     totalDepth,
		Liquidity: liquidity,
		Timestamp: currentTime,
	}, nil
}

// CalculatePercentile calculates the percentile value from a slice of float64 values
// 计算float64值切片的百分位数
func (m *Manager) CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Make a copy to avoid modifying the original slice
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Sort the values
	sort.Float64s(sorted)

	// Calculate index
	idx := percentile * float64(len(sorted)-1)
	lowerIdx := int(math.Floor(idx))
	upperIdx := int(math.Ceil(idx))

	if lowerIdx < 0 {
		lowerIdx = 0
	}
	if upperIdx >= len(sorted) {
		upperIdx = len(sorted) - 1
	}

	if lowerIdx == upperIdx {
		return sorted[lowerIdx]
	}

	// Linear interpolation
	weight := idx - float64(lowerIdx)
	return sorted[lowerIdx] + weight*(sorted[upperIdx]-sorted[lowerIdx])
}

// PerformLinearRegression performs linear regression on time series data to calculate the slope
// 对时间序列数据进行线性回归，计算斜率
func (m *Manager) PerformLinearRegression(items []LiquidityWindowItem) float64 {
	n := len(items)
	if n < 2 {
		return 0
	}

	// Calculate means
	var sumX, sumY, sumXY, sumX2 float64
	for _, item := range items {
		x := float64(item.Timestamp)
		y := item.Metrics.Liquidity
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	// Calculate slope using least squares method
	numerator := sumXY - float64(n)*meanX*meanY
	denominator := sumX2 - float64(n)*meanX*meanX

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

/*
参数说明 ：
DetectLiquidityShrinkage detects liquidity shrinkage using multiple conditions
- instID ：交易对ID（如 "BTC-USDT"）
- nearPriceDeltaPercent ：价格附近的百分比阈值（用于计算流动性）
- shortWindowSeconds ：短期趋势分析窗口（秒）
- longWindowSeconds ：长期基准比较窗口（秒）
- slopeThreshold ：流动性变化斜率阈值（负值表示收缩趋势）
返回值 ：

- *LiquidityShrinkData ：包含流动性状态、警告级别等信息的结构体
- error ：可能的错误信息
*/
func (m *Manager) DetectLiquidityShrinkage(instID string, nearPriceDeltaPercent float64, shortWindowSeconds int, longWindowSeconds int, slopeThreshold float64) (*LiquidityShrinkData, error) {
	// Calculate current liquidity metrics
	currentMetrics, err := m.CalculateLiquidityMetrics(instID, nearPriceDeltaPercent)
	if err != nil {
		return nil, err
	}

	// Set default parameters if invalid
	if shortWindowSeconds <= 0 {
		shortWindowSeconds = 30 // 短期趋势分析窗口 Default to 30 seconds
	}
	if longWindowSeconds <= 0 {
		longWindowSeconds = 1800 // 长期基准比较窗口 Default to 30 minutes
	}
	if slopeThreshold > 0 {
		slopeThreshold = -slopeThreshold // Ensure it's negative 确保为负值（表示收缩趋势）
	} else if slopeThreshold == 0 {
		slopeThreshold = -0.01 // Default slope threshold 默认斜率阈值
	}
	if nearPriceDeltaPercent <= 0 {
		nearPriceDeltaPercent = 0.5 // Default to 0.5%
	}

	currentTime := time.Now().Unix()

	// Add current metrics to the sliding window
	// 添加当前指标到滑动窗口
	m.liquidityWindows[instID] = append(m.liquidityWindows[instID], LiquidityWindowItem{
		Metrics:   *currentMetrics,
		Timestamp: currentTime,
	})

	// Keep only the most recent entries within the long window
	// 保持滑动窗口内最新的条目，仅保留长期基准比较窗口内的数据
	cutoffTime := currentTime - int64(longWindowSeconds)
	startIndex := 0
	for i, item := range m.liquidityWindows[instID] {
		if item.Timestamp > cutoffTime {
			startIndex = i
			break
		}
	}
	m.liquidityWindows[instID] = m.liquidityWindows[instID][startIndex:]

	// If we don't have enough data points yet, return normal state
	if len(m.liquidityWindows[instID]) < 2 {
		return &LiquidityShrinkData{
			Warning:      false,
			WarningLevel: "none",
			Liquidity:    currentMetrics.Liquidity,
			Spread:       currentMetrics.Spread,
			Depth:        currentMetrics.Depth,
			Slope:        0,
			Timestamp:    currentTime,
		}, nil
	}

	// Separate data for short-term trend analysis and long-term percentiles
	// 分离短期趋势分析数据和长期基准比较数据
	shortWindowStart := currentTime - int64(shortWindowSeconds)
	var shortWindowItems []LiquidityWindowItem
	var longWindowLiquidity []float64
	var longWindowSpread []float64

	for _, item := range m.liquidityWindows[instID] {
		if item.Timestamp >= shortWindowStart {
			shortWindowItems = append(shortWindowItems, item)
		}
		longWindowLiquidity = append(longWindowLiquidity, item.Metrics.Liquidity)
		longWindowSpread = append(longWindowSpread, item.Metrics.Spread)
	}

	// Calculate slope for short-term trend
	// 计算短期趋势分析的斜率
	slope := m.PerformLinearRegression(shortWindowItems)

	// Calculate percentiles for long-term comparison
	// 计算长期基准比较的25%和75%分位数
	liquidity25thPercentile := m.CalculatePercentile(longWindowLiquidity, 0.25)
	spread75thPercentile := m.CalculatePercentile(longWindowSpread, 0.75)

	// Check conditions for liquidity shrinkage
	conditionA := currentMetrics.Liquidity < liquidity25thPercentile // Low absolute liquidity 绝对流动性低
	conditionB := slope < slopeThreshold                             // Negative trend 负趋势
	conditionC := currentMetrics.Spread > spread75thPercentile       // High spread 高价差

	// Count satisfied conditions
	satisfiedConditions := 0
	if conditionA {
		satisfiedConditions++
	}
	if conditionB {
		satisfiedConditions++
	}
	if conditionC {
		satisfiedConditions++
	}

	// Determine warning level
	// 根据满足条件的数量确定警告级别
	warning := satisfiedConditions >= 2
	warningLevel := "none"
	switch satisfiedConditions {
	case 2:
		warningLevel = "light" //轻：2个条件满足
	case 3:
		if slope < 2*slopeThreshold { // Severe negative trend 严重负趋势
			warningLevel = "severe" //重：3个条件满足且斜率达到严重程度
			log.Printf("InstId: %v, Severe negative trend detected: %v, warningLevel: %v", instID, slope, warningLevel)
		} else {
			warningLevel = "moderate" //中：3个条件满足但斜率未达到严重程度
		}
	}

	return &LiquidityShrinkData{
		Warning:      warning,
		WarningLevel: warningLevel,
		Liquidity:    currentMetrics.Liquidity,
		Spread:       currentMetrics.Spread,
		Depth:        currentMetrics.Depth,
		Slope:        slope,
		Timestamp:    currentTime,
	}, nil
}

// ToRedisMap converts LiquidityShrinkData to a map for Redis storage
func (l *LiquidityShrinkData) ToRedisMap() map[string]interface{} {
	return map[string]interface{}{
		"warning":       l.Warning,
		"warning_level": l.WarningLevel,
		"liquidity":     l.Liquidity,
		"spread":        l.Spread,
		"depth":         l.Depth,
		"slope":         l.Slope,
		"timestamp":     l.Timestamp,
	}
}
