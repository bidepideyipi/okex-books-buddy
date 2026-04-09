package signal

import (
	"context"
	"log"
	"sync"

	"github.com/supermancell/okex-buddy/internal/ws"
)

// PositionProcessor handles positions based on trading signals
type PositionProcessor struct {
	privateClient *ws.PrivateClient
	ctx           context.Context
	cancel        context.CancelFunc
	orderIDMap    sync.Map
	clOrdIDMap    sync.Map
}

// NewPositionProcessor creates a new position processor
func NewPositionProcessor(privateClient *ws.PrivateClient) *PositionProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &PositionProcessor{
		privateClient: privateClient,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the order processor
func (p *PositionProcessor) Start() {
	log.Println("Order processor started")
	<-p.ctx.Done()
	log.Println("Order processor stopping...")
}

// Stop stops the order processor
func (p *PositionProcessor) Stop() {
	p.cancel()
}

// handlePositionData processes position channel data
func (p *PositionProcessor) HandleEvent(data []interface{}) error {

	// for _, item := range data {
	// 	if position, ok := item.(map[string]interface{}); ok {
	// 		log.Printf("[DEBUG] Processing position data: %+v", position)
	// 	}
	// }
	return nil
}
