package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// Signal represents a trading signal from Redis
type Signal struct {
	SignalID        string `json:"signal_id"`
	StrategyName    string `json:"strategy_name"`
	InstID          string `json:"inst_id"`
	Side            string `json:"side"`
	OrdType         string `json:"ord_type"`
	PosSide         string `json:"pos_side"`
	Sz              string `json:"sz"`
	Px              string `json:"px"`
	ReduceOnly      bool   `json:"reduce_only"`
	TPTriggerPx     string `json:"tp_trigger_px"`
	TPTriggerPxType string `json:"tp_trigger_px_type"`
	SlTriggerPx     string `json:"sl_trigger_px"`
	SlTriggerPxType string `json:"sl_trigger_px_type"`
	Ccy             string `json:"ccy"`
	Tag             string `json:"tag"`
	Timestamp       int64  `json:"timestamp"`
}

// SignalConsumer consumes trading signals from Redis List
type SignalConsumer struct {
	redisClient   *redis.Client
	mongoClient   *mongodb.Client
	strategies    []string
	timeout       time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	orderCallback func(*Signal) (string, string, error)
}

// NewSignalConsumer creates a new signal consumer
func NewSignalConsumer(redisClient *redis.Client, mongoClient *mongodb.Client, strategies []string) *SignalConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SignalConsumer{
		redisClient: redisClient,
		mongoClient: mongoClient,
		strategies:  strategies,
		timeout:     5 * time.Second,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetOrderCallback sets the callback function for placing orders
func (c *SignalConsumer) SetOrderCallback(callback func(*Signal) (string, string, error)) {
	c.orderCallback = callback
}

// Start starts consuming signals from Redis
func (c *SignalConsumer) Start() {
	log.Printf("Signal consumer started, watching strategies: %v", c.strategies)

	var wg sync.WaitGroup
	for _, strategy := range c.strategies {
		wg.Add(1)
		go func(strategyName string) {
			defer wg.Done()
			c.consumeSignals(strategyName)
		}(strategy)
	}

	<-c.ctx.Done()
	log.Println("Signal consumer stopping...")
}

// Stop stops the signal consumer
func (c *SignalConsumer) Stop() {
	c.cancel()
}

// consumeSignals consumes signals from a specific strategy
func (c *SignalConsumer) consumeSignals(strategyName string) {
	key := fmt.Sprintf("trading_signals:%s", strategyName)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			result, err := c.redisClient.BRPop(c.ctx, c.timeout, key).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error consuming signal from %s: %v", key, err)
				}
				continue
			}

			if len(result) < 2 {
				log.Printf("Invalid BRPOP result: %v", result)
				continue
			}

			signalData := result[1]
			if err := c.processSignal(signalData); err != nil {
				log.Printf("Error processing signal: %v", err)
			}
		}
	}
}

// processSignal processes a trading signal
func (c *SignalConsumer) processSignal(signalData string) error {
	var signal Signal
	if err := json.Unmarshal([]byte(signalData), &signal); err != nil {
		return fmt.Errorf("failed to unmarshal signal: %w", err)
	}

	if err := c.validateSignal(&signal); err != nil {
		return fmt.Errorf("signal validation failed: %w", err)
	}

	tradingSignal := &mongodb.TradingSignal{
		ID:               fmt.Sprintf("signal_%s", signal.SignalID),
		SignalID:         signal.SignalID,
		StrategyName:     signal.StrategyName,
		InstID:           signal.InstID,
		Side:             signal.Side,
		OrdType:          signal.OrdType,
		PosSide:          signal.PosSide,
		Sz:               signal.Sz,
		Px:               signal.Px,
		ReduceOnly:       signal.ReduceOnly,
		Status:           "pending",
		SignalTimestamp:  signal.Timestamp,
		ProcessTimestamp: time.Now().UnixMilli(),
		CreatedAt:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	if err := c.mongoClient.InsertTradingSignal(tradingSignal); err != nil {
		return fmt.Errorf("failed to insert trading signal: %w", err)
	}

	log.Printf("Signal recorded: %s (inst=%s, side=%s, type=%s)",
		signal.SignalID, signal.InstID, signal.Side, signal.OrdType)

	if c.orderCallback != nil {
		clOrdID, ordID, err := c.orderCallback(&signal)
		if err != nil {
			log.Printf("Failed to place order for signal %s: %v", signal.SignalID, err)
			c.mongoClient.UpdateSignalStatusWithError(signal.SignalID, "failed", err.Error())
			return err
		}

		if err := c.mongoClient.UpdateSignalWithOrderID(signal.SignalID, ordID, clOrdID, "processing"); err != nil {
			log.Printf("Failed to update signal with order ID: %v", err)
		}

		log.Printf("Order placed for signal %s: clOrdID=%s, ordID=%s", signal.SignalID, clOrdID, ordID)
	}

	return nil
}

// validateSignal validates the trading signal
func (c *SignalConsumer) validateSignal(signal *Signal) error {
	if signal.SignalID == "" {
		return fmt.Errorf("signal_id is required")
	}

	if signal.StrategyName == "" {
		return fmt.Errorf("strategy_name is required")
	}

	if signal.InstID == "" {
		return fmt.Errorf("inst_id is required")
	}

	if signal.Side != "buy" && signal.Side != "sell" {
		return fmt.Errorf("side must be 'buy' or 'sell'")
	}

	if signal.OrdType == "" {
		return fmt.Errorf("ord_type is required")
	}

	if signal.PosSide != "long" && signal.PosSide != "short" && signal.PosSide != "net" {
		return fmt.Errorf("pos_side must be 'long', 'short', or 'net'")
	}

	if signal.Sz == "" {
		return fmt.Errorf("sz (size) is required")
	}

	if sz, err := strconv.ParseFloat(signal.Sz, 64); err != nil || sz <= 0 {
		return fmt.Errorf("sz must be a positive number")
	}

	if signal.OrdType == "limit" && signal.Px == "" {
		return fmt.Errorf("px (price) is required for limit orders")
	}

	if signal.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}

	return nil
}

// GetSignalStatus retrieves the status of a trading signal
func (c *SignalConsumer) GetSignalStatus(signalID string) (string, error) {
	return "", nil
}

// StartSignalConsumer starts the trading signal consumer
func StartSignalConsumer(redisClient *redisclient.Client, mongoClient *mongodb.Client, privateClient *ws.PrivateClient) {
	strategies := []string{"momentum_strategy"}
	consumer := NewSignalConsumer(redisClient.Client(), mongoClient, strategies)

	orderProcessor := NewOrderProcessor(privateClient, mongoClient)
	consumer.SetOrderCallback(func(sig *Signal) (string, string, error) {
		return orderProcessor.PlaceOrder(sig)
	})

	go consumer.Start()
	go orderProcessor.Start()

	log.Println("Signal consumer started")
}
