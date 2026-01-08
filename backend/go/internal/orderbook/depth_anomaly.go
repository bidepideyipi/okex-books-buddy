package orderbook

import (
	"math"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/utils"
)

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

	// Use time window utility for automatic expiration management
	if m.depthWindows[instID] == nil {
		m.depthWindows[instID] = utils.NewGenericTimeWindow(int64(windowSize))
	}

	// Add current depth to the time window
	depthItem := &DepthWindowItem{
		Depth:     currentDepth,
		Timestamp: time.Now().Unix(),
	}
	m.depthWindows[instID].Add(depthItem)

	// Get current items from window
	windowItems := m.depthWindows[instID].GetItems()
	if len(windowItems) < 2 {
		return &DepthAnomalyData{
			Anomaly:   false,
			ZScore:    0,
			Depth:     currentDepth,
			Mean:      currentDepth,
			StdDev:    0,
			Timestamp: time.Now().Unix(),
			Direction: "", // Not enough data to determine direction
			Intensity: 0,
		}, nil
	}

	// Get historical depths (excluding current value)
	historicalDepths := make([]float64, 0, len(windowItems)-1)
	for i, item := range windowItems {
		if i < len(windowItems)-1 { // Exclude the last (current) item
			if depthItem, ok := item.(*DepthWindowItem); ok {
				historicalDepths = append(historicalDepths, depthItem.Depth)
			}
		}
	}

	// Calculate statistics using utility functions
	historicalMean := utils.CalculateMean(historicalDepths)
	stdDev := utils.CalculateStdDev(historicalDepths)

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
		Timestamp: time.Now().Unix(),
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

// CalculateDepthInRange calculates the total depth within a given price range around the mid price
func (m *Manager) CalculateDepthInRange(instID string, priceRangePercent float64) (float64, error) {
	asks, bids, err := m.GetTop400(instID)
	if err != nil {
		return 0, err
	}

	if len(asks) == 0 || len(bids) == 0 {
		return 0, nil
	}

	// Calculate mid price
	bestBid, err1 := strconv.ParseFloat(bids[0].Price, 64)
	bestAsk, err2 := strconv.ParseFloat(asks[0].Price, 64)
	if err1 != nil || err2 != nil {
		return 0, nil
	}

	midPrice := (bestBid + bestAsk) / 2.0
	if midPrice <= 0 {
		return 0, nil
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
