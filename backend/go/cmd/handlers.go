package main

import (
	"fmt"
	"log"

	"github.com/supermancell/okex-buddy/internal/candlestick"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// NewPublicMessageHandler creates a message handler for public WebSocket
func NewPublicMessageHandler(obManager *orderbook.Manager) ws.MessageHandler {
	return func(msg []byte) error {
		log.Printf("[DEBUG] PublicMessageHandler called with %d bytes", len(msg))
		if err := obManager.ProcessMessage(msg); err != nil {
			return fmt.Errorf("failed to process message: %w", err)
		}
		return nil
	}
}

// NewBusinessMessageHandler creates a message handler for business WebSocket
func NewBusinessMessageHandler(mongoClient *mongodb.Client) ws.MessageHandler {
	return func(msg []byte) error {
		log.Printf("[DEBUG] BusinessMessageHandler called with %d bytes", len(msg))
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
