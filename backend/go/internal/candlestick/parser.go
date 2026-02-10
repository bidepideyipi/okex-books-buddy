package candlestick

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/mongodb"
)

// Message represents the incoming WebSocket message structure
type Message struct {
	Arg struct {
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Data [][]string `json:"data"`
}

// ParseCandlestick parses a candlestick message and converts it to MongoDB format
func ParseCandlestick(msg []byte) ([]mongodb.Candlestick, error) {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	candlesticks := make([]mongodb.Candlestick, 0, len(message.Data))

	for _, candleData := range message.Data {
		candle, err := convertToCandlestick(message.Arg.InstID, message.Arg.Channel, candleData)
		if err != nil {
			return nil, err
		}
		candlesticks = append(candlesticks, candle)
	}

	return candlesticks, nil
}

// convertToCandlestick converts raw candlestick data to MongoDB format
func convertToCandlestick(instID, channel string, data []string) (mongodb.Candlestick, error) {
	if len(data) < 9 {
		return mongodb.Candlestick{}, fmt.Errorf("invalid candlestick data length: %d", len(data))
	}

	timestamp, err := strconv.ParseInt(data[0], 10, 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	open, err := strconv.ParseFloat(data[1], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse open: %w", err)
	}

	high, err := strconv.ParseFloat(data[2], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse high: %w", err)
	}

	low, err := strconv.ParseFloat(data[3], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse low: %w", err)
	}

	close, err := strconv.ParseFloat(data[4], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse close: %w", err)
	}

	vol, err := strconv.ParseFloat(data[5], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse vol_ccy: %w", err)
	}

	volCcyQuote, err := strconv.ParseFloat(data[6], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse vol_ccy_quote: %w", err)
	}

	volCcyQuoteConfirm, err := strconv.ParseFloat(data[7], 64)
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse vol_ccy_quote_confirm: %w", err)
	}

	confirm, err := strconv.Atoi(data[8])
	if err != nil {
		return mongodb.Candlestick{}, fmt.Errorf("failed to parse confirm: %w", err)
	}

	t := time.UnixMilli(timestamp)
	bar := extractBar(channel)

	return mongodb.Candlestick{
		InstrumentID: instID,
		Bar:          bar,
		Timestamp:    timestamp,
		Open:         open,
		High:         high,
		Low:          low,
		Close:        close,
		Volume:       vol,
		VolCcy:       volCcyQuote,
		VolCcyQuote:  volCcyQuoteConfirm,
		Confirm:      confirm,
		DayOfWeek:    int(t.Weekday()),
		RecordDT:     t.Format("2006-01-02"),
		RecordHour:   t.Hour(),
	}, nil
}

// extractBar extracts the bar period from the channel name
func extractBar(channel string) string {
	switch channel {
	case "candle1D":
		return "1D"
	case "candle4H":
		return "4H"
	case "candle1H":
		return "1H"
	case "candle15m":
		return "15m"
	default:
		return channel
	}
}

// RoundFloat rounds a float64 to a specified precision
func RoundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
