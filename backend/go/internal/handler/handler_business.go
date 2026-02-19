package handler

import (
	"log"

	"github.com/supermancell/okex-buddy/internal/candlestick"
	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/mongodb"
)

// NewBusinessMessageHandler creates a message handler for business WebSocket
func NewBusinessMessageHandler(mongoClient *mongodb.Client) common.MessageHandler {
	return func(msg []byte) error {
		candles, err := candlestick.ParseCandlestick(msg)
		if err != nil {
			log.Printf("Failed to parse candlestick message: %v", err)
			return err
		}

		for _, candle := range candles {
			log.Printf("[DEBUG] Inserting candlestick: %s, %s, %v", candle.InstrumentID, candle.Bar, candle.Timestamp)
			if err := mongoClient.InsertCandlestick(&candle); err != nil {
				log.Printf("Failed to insert candlestick: %v", err)
			}
		}
		return nil
	}
}
