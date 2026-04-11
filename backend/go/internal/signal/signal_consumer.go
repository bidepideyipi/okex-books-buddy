package signal

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/supermancell/okex-buddy/internal/common"
)

// SignalConsumer consumes trading signals from Redis List
type SignalConsumer struct {
	timeout        time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	messageHandler common.MessageHandler
	redisClient    *redis.Client
}

// NewSignalConsumer creates a new signal consumer
func NewSignalConsumer(redisClient *redis.Client) *SignalConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SignalConsumer{
		timeout:     5 * time.Second,
		ctx:         ctx,
		cancel:      cancel,
		redisClient: redisClient,
	}
}

// SetOrderCallback sets the callback function for placing orders
func (c *SignalConsumer) SetMessageHandler(handler common.MessageHandler) {
	c.messageHandler = handler
}

// Start starts consuming signals from Redis
func (c *SignalConsumer) Start() {
	log.Printf("Signal consumer started, watching list: %s", common.LIST_KEY)

	go c.consumeSignals()

	<-c.ctx.Done()
	log.Println("Signal consumer stopping...")
}

// Stop stops the signal consumer
func (c *SignalConsumer) Stop() {
	c.cancel()
}

// consumeSignals consumes signals from a specific strategy
func (c *SignalConsumer) consumeSignals() {

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			result, err := c.redisClient.BRPop(c.ctx, c.timeout, common.LIST_KEY).Result()
			if err != nil {
				if err != redis.Nil {
					log.Printf("Error consuming signal from %s: %v", common.LIST_KEY, err)
				}
				continue
			}

			if len(result) < 2 {
				log.Printf("Invalid BRPOP result: %v", result)
				continue
			}

			signalData := result[1]
			if c.messageHandler == nil {
				log.Printf("No message handler set, skipping signal: %s", signalData)
				continue
			}

			if err := c.messageHandler([]byte(signalData)); err != nil {
				log.Printf("Error processing signal: %v", err)
			}
		}
	}
}

// processSignal processes a trading signal

// GetSignalStatus retrieves the status of a trading signal
func (c *SignalConsumer) GetSignalStatus(signalID string) (string, error) {
	return "", nil
}

// StartSignalConsumer starts the trading signal consumer
func StartSignalConsumer(redisClient *redis.Client, msgHandler common.MessageHandler) {
	consumer := NewSignalConsumer(redisClient)

	consumer.SetMessageHandler(msgHandler)

	go consumer.Start()
	//go orderProcessor.Start()

	log.Println("Signal consumer started")
}
