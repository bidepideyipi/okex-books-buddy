package signal

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/redisclient"
)

const (
	STREAM_KEY    = "signals"
	GROUP_NAME    = "gp-go"
	CONSUMER_NAME = "consumer-go"
)

// StreamSignal represents a signal from Redis Stream
type StreamSignal struct {
	Timestamp           string
	InstID              string
	Bar                 string
	Prediction          string
	PredictionLabel     string
	PredictionHigh      string
	PredictionHighLabel string
	PredictionLow       string
	PredictionLowLabel  string
	Probabilities       string
	ProbabilitiesHigh   string
	ProbabilitiesLow    string
	FeaturesCount       string
	Price               string
	Line1               string
	Line2               string
}

// StreamConsumer consumes signals from Redis Stream
type StreamConsumer struct {
	redisClient    *redis.Client
	mongoClient    *mongodb.Client
	timeout        time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	signalCallback func(*StreamSignal) error
}

// NewStreamConsumer creates a new stream consumer
func NewStreamConsumer(redisClient *redis.Client, mongoClient *mongodb.Client) *StreamConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamConsumer{
		redisClient: redisClient,
		mongoClient: mongoClient,
		timeout:     5 * time.Second,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetSignalCallback sets the callback function for processing signals
func (c *StreamConsumer) SetSignalCallback(callback func(*StreamSignal) error) {
	c.signalCallback = callback
}

// Start starts consuming signals from Redis Stream
func (c *StreamConsumer) Start() {
	log.Println("[INFO] Stream consumer starting, watching stream:", STREAM_KEY)
	log.Println("[INFO] Consumer group:", GROUP_NAME, "Consumer name:", CONSUMER_NAME)

	// Create consumer group if it doesn't exist
	if err := c.createConsumerGroup(); err != nil {
		log.Printf("[ERROR] Failed to create consumer group: %v", err)
		return
	}

	go c.consumeStream()

	log.Println("[INFO] Stream consumer goroutine started")

	<-c.ctx.Done()
	log.Println("[INFO] Stream consumer stopping...")
}

// Stop stops the stream consumer
func (c *StreamConsumer) Stop() {
	c.cancel()
}

// createConsumerGroup creates consumer group if it doesn't exist
func (c *StreamConsumer) createConsumerGroup() error {
	err := c.redisClient.XGroupCreateMkStream(c.ctx, STREAM_KEY, GROUP_NAME, "0").Err()
	if err != nil {
		// If group already exists, that's fine
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			log.Printf("[INFO] Consumer group %s already exists", GROUP_NAME)
			return nil
		}
		return err
	}
	log.Printf("[INFO] Created consumer group: %s", GROUP_NAME)
	return nil
}

// getStringValue safely extracts string value from interface
func getStringValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// consumeStream consumes messages from Redis Stream
func (c *StreamConsumer) consumeStream() {
	log.Println("[INFO] Starting consumeStream loop")

	for {
		select {
		case <-c.ctx.Done():
			log.Println("[INFO] consumeStream loop stopped by context")
			return
		default:
			log.Printf("[DEBUG] Attempting to read from stream group %s, consumer %s", GROUP_NAME, CONSUMER_NAME)

			streams, err := c.redisClient.XReadGroup(c.ctx, &redis.XReadGroupArgs{
				Group:    GROUP_NAME,
				Consumer: CONSUMER_NAME,
				Streams:  []string{STREAM_KEY, ">"},
				Count:    10,
				Block:    c.timeout,
				NoAck:    false,
			}).Result()

			if err != nil {
				if err != redis.Nil {
					log.Printf("[ERROR] Error reading stream: %v", err)
				}
				continue
			}

			log.Printf("[DEBUG] Received %d stream(s)", len(streams))

			for _, stream := range streams {
				log.Printf("[DEBUG] Stream: %s has %d message(s)", stream.Stream, len(stream.Messages))

				for _, message := range stream.Messages {
					log.Printf("[DEBUG] Processing message ID: %s", message.ID)

					if err := c.processStreamMessage(message); err != nil {
						log.Printf("[ERROR] Error processing stream message %s: %v", message.ID, err)
						// Don't ack on error, will be retried
					} else {
						// Acknowledge message
						if err := c.redisClient.XAck(c.ctx, STREAM_KEY, GROUP_NAME, message.ID).Err(); err != nil {
							log.Printf("[ERROR] Failed to acknowledge message %s: %v", message.ID, err)
						}
					}
				}
			}
		}
	}
}

// processStreamMessage processes a message from Redis Stream
func (c *StreamConsumer) processStreamMessage(message redis.XMessage) error {
	// Print original message before type conversion
	log.Printf("[DEBUG] Original message values: %+v", message.Values)

	signal := &StreamSignal{
		Timestamp:           getStringValue(message.Values["timestamp"]),
		InstID:              getStringValue(message.Values["inst_id"]),
		Bar:                 getStringValue(message.Values["bar"]),
		Prediction:          getStringValue(message.Values["prediction"]),
		PredictionLabel:     getStringValue(message.Values["prediction_label"]),
		PredictionHigh:      getStringValue(message.Values["prediction_high"]),
		PredictionHighLabel: getStringValue(message.Values["prediction_high_label"]),
		PredictionLow:       getStringValue(message.Values["prediction_low"]),
		PredictionLowLabel:  getStringValue(message.Values["prediction_low_label"]),
		Probabilities:       getStringValue(message.Values["probabilities"]),
		ProbabilitiesHigh:   getStringValue(message.Values["probabilities_high"]),
		ProbabilitiesLow:    getStringValue(message.Values["probabilities_low"]),
		FeaturesCount:       getStringValue(message.Values["features_count"]),
		Price:               getStringValue(message.Values["price"]),
		Line1:               getStringValue(message.Values["line1"]),
		Line2:               getStringValue(message.Values["line2"]),
	}

	if err := c.validateStreamSignal(signal); err != nil {
		return fmt.Errorf("stream signal validation failed: %w", err)
	}

	log.Printf("[INFO] Stream signal received: id=%s, inst=%s, prediction=%s, price=%s",
		message.ID, signal.InstID, signal.Prediction, signal.Price)

	if c.signalCallback != nil {
		if err := c.signalCallback(signal); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	return nil
}

// validateStreamSignal validates the stream signal
func (c *StreamConsumer) validateStreamSignal(signal *StreamSignal) error {
	if signal.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}

	if signal.InstID == "" {
		return fmt.Errorf("inst_id is required")
	}

	if signal.Prediction == "" {
		return fmt.Errorf("prediction is required")
	}

	if signal.Price == "" {
		return fmt.Errorf("price is required")
	}

	return nil
}

// StartStreamConsumer starts the stream signal consumer
func StartStreamConsumer(redisClient *redisclient.Client, mongoClient *mongodb.Client) {
	consumer := NewStreamConsumer(redisClient.Client(), mongoClient)

	// Set callback function for processing signals
	consumer.SetSignalCallback(func(sig *StreamSignal) error {
		// Process the stream signal here
		// Example: store in MongoDB, trigger trading logic, etc.
		log.Printf("[INFO] Processing stream signal: inst=%s, prediction=%s, price=%s, bar=%s",
			sig.InstID, sig.Prediction, sig.Price, sig.Bar)

		// TODO: Add business logic here
		return nil
	})

	go consumer.Start()

	log.Println("Stream consumer started")
}
