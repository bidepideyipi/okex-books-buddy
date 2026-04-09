package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// OrderProcessor handles placing orders based on trading signals
type OrderProcessor struct {
	privateClient *ws.PrivateClient
	mongoClient   *mongodb.Client
	ctx           context.Context
	cancel        context.CancelFunc
	orderIDMap    sync.Map
	clOrdIDMap    sync.Map
}

// NewOrderProcessor creates a new order processor
func NewOrderProcessor(privateClient *ws.PrivateClient, mongoClient *mongodb.Client) *OrderProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &OrderProcessor{
		privateClient: privateClient,
		mongoClient:   mongoClient,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the order processor
func (p *OrderProcessor) Start() {
	log.Println("Order processor started")
	<-p.ctx.Done()
	log.Println("Order processor stopping...")
}

// Stop stops the order processor
func (p *OrderProcessor) Stop() {
	p.cancel()
}

// PlaceOrder places an order based on trading signal
func (p *OrderProcessor) PlaceOrder(signal *Signal) (clOrdID, ordID string, err error) {
	if !p.privateClient.IsAuthenticated() {
		return "", "", fmt.Errorf("private client not authenticated")
	}

	clOrdID = fmt.Sprintf("%d", time.Now().UnixMilli())

	args := []map[string]string{
		{
			"instIdCode": signal.InstID,
			"tdMode":     "cross",
			"clOrdId":    clOrdID,
			"side":       signal.Side,
			"ordType":    signal.OrdType,
			"posSide":    signal.PosSide,
			"sz":         signal.Sz,
			"reduceOnly": fmt.Sprintf("%t", signal.ReduceOnly),
		},
	}

	if signal.OrdType == "limit" && signal.Px != "" {
		args[0]["px"] = signal.Px
	}

	if err := p.privateClient.PlaceOrder(args); err != nil {
		return "", "", err
	}

	p.clOrdIDMap.Store(signal.SignalID, clOrdID)

	return clOrdID, "", nil
}

// handleOrderData processes order channel data
func (p *OrderProcessor) HandleOrderEvent(data []interface{}) error {
	log.Printf("[DEBUG] Processing order data: %+v", data)
	for _, item := range data {
		if order, ok := item.(map[string]interface{}); ok {
			// Parse order ID
			ordID := ""
			if id, ok := order["ordId"].(string); ok {
				ordID = id
			} else if idFloat, ok := order["ordId"].(float64); ok {
				ordID = fmt.Sprintf("%.0f", idFloat)
			}

			if ordID != "" {
				log.Printf("[DEBUG] Processing order: %s", ordID)
				p.orderIDMap.Store(ordID, order)

				// Find associated signal by client order ID
				if clOrdID, ok := order["clOrdId"].(string); ok {
					log.Printf("[DEBUG] Found client order ID: %s", clOrdID)
					signalID := p.findSignalIDByClOrdID(clOrdID)
					if signalID != "" {
						log.Printf("[DEBUG] Found associated signal ID: %s", signalID)
						if err := p.mongoClient.UpdateSignalWithOrderID(signalID, ordID, clOrdID, "success"); err != nil {
							log.Printf("[ERROR] Failed to update signal %s with order ID: %v", signalID, err)
						}
					}
				}
			}
		}
	}
	return nil
}

// handlePositionData processes position channel data
func (p *OrderProcessor) HandlePositionEvent(data []interface{}) error {
	//for _, item := range data {
	//if position, ok := item.(map[string]interface{}); ok {
	//log.Printf("[DEBUG] Processing position data: %+v", position)
	// TODO: Implement position processing logic
	//}
	//}
	return nil
}

// HandleErrorResponse handles error response from WebSocket
func (p *OrderProcessor) HandleErrorResponse(message []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return err
	}

	if op, ok := msg["op"].(string); ok && op == "order" {
		if code, ok := msg["code"].(string); ok {
			if code != "0" {
				msgText, _ := msg["msg"].(string)
				log.Printf("[ERROR] Order failed: code=%s, msg=%s", code, msgText)

				return fmt.Errorf("order error: %s - %s", code, msgText)
			}
		}
	}

	return nil
}

// findSignalIDByClOrdID finds signal ID by client order ID
func (p *OrderProcessor) findSignalIDByClOrdID(clOrdID string) string {
	var signalID string
	p.clOrdIDMap.Range(func(key, value interface{}) bool {
		if value.(string) == clOrdID {
			signalID = key.(string)
			return false
		}
		return true
	})
	return signalID
}

// GenerateClOrdID generates a unique client order ID
func GenerateClOrdID(signalID string) string {
	return fmt.Sprintf("client_%d_%s", time.Now().UnixMilli(), signalID)
}
