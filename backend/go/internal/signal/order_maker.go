package signal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/supermancell/okex-buddy/internal/ws"
)

// PrivateClientInterface defines the interface for private client operations
type PrivateClientInterface interface {
	IsAuthenticated() bool
	PlaceOrder(args []map[string]string) error
}

// OrderProcessor handles placing orders based on trading signals
type OrderProcessor struct {
	privateClient PrivateClientInterface
	ctx           context.Context
	cancel        context.CancelFunc
	orderIDMap    sync.Map
	clOrdIDMap    sync.Map
}

// NewOrderMaker creates a new order maker
func NewOrderMaker(privateClient *ws.PrivateClient) *OrderProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &OrderProcessor{
		privateClient: privateClient,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// NewOrderMakerWithInterface creates a new order maker with interface (for testing)
func NewOrderMakerWithInterface(privateClient PrivateClientInterface) *OrderProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &OrderProcessor{
		privateClient: privateClient,
		ctx:           ctx,
		cancel:        cancel,
	}
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
