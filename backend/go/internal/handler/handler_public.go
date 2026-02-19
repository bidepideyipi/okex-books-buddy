package handler

import (
	"fmt"

	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/orderbook"
)

// NewPublicMessageHandler creates a message handler for public WebSocket
func NewPublicMessageHandler(obManager *orderbook.Manager) common.MessageHandler {
	return func(msg []byte) error {
		if err := obManager.ProcessMessage(msg); err != nil {
			return fmt.Errorf("failed to process message: %w", err)
		}
		return nil
	}
}
