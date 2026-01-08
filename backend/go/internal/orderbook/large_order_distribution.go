package orderbook

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/utils"
)

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
	if m.sentimentMap[instID] == nil {
		m.sentimentMap[instID] = utils.NewGenericTimeWindow(30) // 30 seconds
	}

	// Add current sentiment to the time window
	sentimentItem := &PriceLevelWithTimeItem{
		Value:     transformedSentiment,
		Timestamp: time.Now().Unix(),
	}
	m.sentimentMap[instID].Add(sentimentItem)

	// Calculate smoothed sentiment as average of values in the window
	windowItems := m.sentimentMap[instID].GetItems()
	if len(windowItems) > 0 {
		var sum float64
		for _, item := range windowItems {
			if sentimentItem, ok := item.(*PriceLevelWithTimeItem); ok {
				sum += sentimentItem.Value
			}
		}
		sentiment = sum / float64(len(windowItems))
	} else {
		sentiment = transformedSentiment
	}

	return largeBuyNotional, largeSellNotional, sentiment, nil
}
