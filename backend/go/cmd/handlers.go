package main

import (
	"fmt"
	"log"

	"github.com/supermancell/okex-buddy/internal/candlestick"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/signal"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// NewPublicMessageHandler creates a message handler for public WebSocket
func NewPublicMessageHandler(obManager *orderbook.Manager) ws.MessageHandler {
	return func(msg []byte) error {
		if err := obManager.ProcessMessage(msg); err != nil {
			return fmt.Errorf("failed to process message: %w", err)
		}
		return nil
	}
}

// NewPrivateMessageHandler creates a message handler for private WebSocket
func NewPrivateMessageHandler(mongoClient *mongodb.Client, orderProcessor *signal.OrderProcessor) ws.MessageHandler {
	privateHandler := ws.NewPrivateHandler(mongoClient)
	return func(msg []byte) error {
		if err := privateHandler.HandleMessage(msg); err != nil {
			log.Printf("Failed to handle private message: %v", err)
			return err
		}

		if orderProcessor != nil {
			if err := orderProcessor.HandleOrderResponse(msg); err != nil {
				if err.Error() != "order ID not found in message" {
					log.Printf("Failed to handle order response: %v", err)
				}
			}

			if err := orderProcessor.HandleErrorResponse(msg); err != nil {
				log.Printf("Error in order response: %v", err)
			}
		}

		return nil
	}
}

// NewBusinessMessageHandler creates a message handler for business WebSocket
func NewBusinessMessageHandler(mongoClient *mongodb.Client) ws.MessageHandler {
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
