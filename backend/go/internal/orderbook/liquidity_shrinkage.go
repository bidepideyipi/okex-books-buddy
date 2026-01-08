package orderbook

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/utils"
)

// DetectLiquidityShrinkage detects liquidity shrinkage using multiple conditions
// 参数说明 ：
// - instID ：交易对ID（如 "BTC-USDT"）
// - nearPriceDeltaPercent ：价格附近的百分比阈值（用于计算流动性）
// - shortWindowSeconds ：短期趋势分析窗口（秒）
// - longWindowSeconds ：长期基准比较窗口（秒）
// - slopeThreshold ：流动性变化斜率阈值（负值表示收缩趋势）
// 返回值 ：
// - *LiquidityShrinkData ：包含流动性状态、警告级别等信息的结构体
// - error ：可能的错误信息
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

	// Use time window utility for automatic expiration management
	if m.liquidityWindows[instID] == nil {
		m.liquidityWindows[instID] = utils.NewGenericTimeWindow(int64(longWindowSeconds))
	}

	// Add current metrics to the time window
	liquidityItem := &LiquidityWindowItem{
		Metrics:   *currentMetrics,
		Timestamp: time.Now().Unix(),
	}
	m.liquidityWindows[instID].Add(liquidityItem)

	// Get current items from window
	windowItems := m.liquidityWindows[instID].GetItems()
	if len(windowItems) < 2 {
		return &LiquidityShrinkData{
			Warning:      false,
			WarningLevel: "none",
			Liquidity:    currentMetrics.Liquidity,
			Spread:       currentMetrics.Spread,
			Depth:        currentMetrics.Depth,
			Slope:        0,
			Timestamp:    time.Now().Unix(),
		}, nil
	}

	// Convert window items to typed items for processing
	typedItems := make([]LiquidityWindowItem, len(windowItems))
	var longWindowLiquidity []float64
	var longWindowSpread []float64

	for i, item := range windowItems {
		if typedItem, ok := item.(*LiquidityWindowItem); ok {
			typedItems[i] = *typedItem
			longWindowLiquidity = append(longWindowLiquidity, typedItem.Metrics.Liquidity)
			longWindowSpread = append(longWindowSpread, typedItem.Metrics.Spread)
		}
	}

	// Separate data for short-term trend analysis
	shortWindowStart := time.Now().Unix() - int64(shortWindowSeconds)
	var shortWindowItems []LiquidityWindowItem

	for _, item := range typedItems {
		if item.Timestamp >= shortWindowStart {
			shortWindowItems = append(shortWindowItems, item)
		}
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
		Timestamp:    time.Now().Unix(),
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

	// Use utility function for percentile calculation
	return utils.CalculatePercentile(sorted, percentile*100) // Convert to percentage (0.25 -> 25)
}

// PerformLinearRegression performs linear regression on time series data to calculate the slope
// 对时间序列数据进行线性回归，计算斜率
func (m *Manager) PerformLinearRegression(items []LiquidityWindowItem) float64 {
	n := len(items)
	if n < 2 {
		return 0
	}

	// Extract x and y values for regression
	xValues := make([]float64, n)
	yValues := make([]float64, n)
	for i, item := range items {
		xValues[i] = float64(item.Timestamp)
		yValues[i] = item.Metrics.Liquidity
	}

	// Use utility function for linear regression
	slope, _ := utils.PerformLinearRegression(xValues, yValues)

	return slope
}
