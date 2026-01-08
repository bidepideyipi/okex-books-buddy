package orderbook

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/utils"
)

// ComputeSupportResistance computes support and resistance levels for a given instrument
// based on the current in-memory order book.
// The implementation is a simplified version of the PRD algorithm:
//   - price range is divided into bins
//   - per-bin notional volume is accumulated
//   - local maxima above a significance threshold are selected and sorted
//     TODO 支撑位和阻力位之间的间隔太近了，没有什么实际意义
func (m *Manager) ComputeSupportResistance(instID string, binCount int, significanceThreshold float64, topN int, minDistancePercent float64) (supports, resistances []float64, spread float64, err error) {
	// First, compute the current support and resistance levels
	asks, bids, err := m.GetTop400(instID)
	if err != nil {
		return nil, nil, 0, err
	}

	if len(asks) == 0 && len(bids) == 0 {
		return nil, nil, 0, fmt.Errorf("empty order book for %s", instID)
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
		return nil, nil, 0, fmt.Errorf("invalid price range for %s", instID)
	}

	binWidth := (maxPrice - minPrice) / float64(binCount)
	if binWidth <= 0 {
		return nil, nil, 0, fmt.Errorf("invalid bin width for %s", instID)
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
	// Calculate spread as the distance between highest support and lowest resistance
	if len(supports) > 0 && len(resistances) > 0 {
		maxSupport := supports[0]
		minResistance := resistances[0]

		// Find highest support (maximum value in supports)
		for _, s := range supports {
			if s > maxSupport {
				maxSupport = s
			}
		}

		// Find lowest resistance (minimum value in resistances)
		for _, r := range resistances {
			if r < minResistance {
				minResistance = r
			}
		}

		spread = minResistance - maxSupport
	} else {
		spread = 0 // No valid support/resistance pair to calculate spread
	}

	// Add current result to sliding window for historical tracking
	m.supportResistanceWindows[instID] = append(m.supportResistanceWindows[instID], SupportResistanceWindowItem{
		Data: SupportResistanceData{
			Supports:    supports,
			Resistances: resistances,
			Spread:      spread,
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

	// Add current spread to sliding window for historical tracking
	m.spreadWindows[instID] = append(m.spreadWindows[instID], SpreadWindowItem{
		Spread:    spread,
		Timestamp: currentTime,
	})

	// Keep only the most recent entries within a time window (e.g., 30 minutes)
	cutoffTime = currentTime - maxWindowSeconds
	startIndex = 0
	for i, item := range m.spreadWindows[instID] {
		if item.Timestamp > cutoffTime {
			startIndex = i
			break
		}
	}
	m.spreadWindows[instID] = m.spreadWindows[instID][startIndex:]

	//log.Printf("Computed support and resistance levels for %s: supports=%v, resistances=%v", instID, supports, resistances)
	return supports, resistances, spread, nil
}

// AnalyzeSpreadZScore calculates a Z-score for the current spread relative to historical values
// This provides a standardized measure of how unusual the current spread is
func (m *Manager) AnalyzeSpreadZScore(instID string, windowSizeMinutes int) (zScore float64, currentSpread float64, err error) {
	if windowSizeMinutes <= 0 {
		windowSizeMinutes = 5 // default to 5 minutes
	}

	spreads := m.spreadWindows[instID]
	if len(spreads) < 2 {
		return 0, 0, fmt.Errorf("insufficient spread data for %s", instID)
	}

	// Calculate statistics for the specified time window
	currentTime := time.Now().Unix()
	cutoffTime := currentTime - int64(windowSizeMinutes*60)

	// Collect spreads within the time window
	var windowSpreads []float64
	for _, item := range spreads {
		if item.Timestamp >= cutoffTime {
			windowSpreads = append(windowSpreads, item.Spread)
		}
	}

	// If not enough data in the time window, use all available data
	if len(windowSpreads) < 2 {
		for _, item := range spreads {
			windowSpreads = append(windowSpreads, item.Spread)
		}
	}

	if len(windowSpreads) < 2 {
		return 0, 0, fmt.Errorf("not enough spread data for %s", instID)
	}

	// Get the current spread
	currentSpread = spreads[len(spreads)-1].Spread

	// Calculate Z-score using utility function
	zScore = utils.CalculateZScore(currentSpread, windowSpreads)

	return zScore, currentSpread, nil
}
