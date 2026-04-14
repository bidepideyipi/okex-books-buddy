package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/supermancell/okex-buddy/internal/common"
)

// StreamConsumer consumes signals from Redis Stream
type StreamConsumer struct {
	redisClient    *redis.Client
	timeout        time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	messageHandler common.StreamSignalHandler
}

// NewStreamConsumer creates a new stream consumer
func NewStreamConsumer(redisClient *redis.Client, messageHandler common.StreamSignalHandler) *StreamConsumer {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamConsumer{
		messageHandler: messageHandler,
		redisClient:    redisClient,
		timeout:        5 * time.Second,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts consuming signals from Redis Stream
func (c *StreamConsumer) Start() {
	log.Println("[INFO] Stream consumer starting, watching stream:", common.STREAM_KEY)
	log.Println("[INFO] Consumer group:", common.GROUP_NAME, "Consumer name:", common.CONSUMER_NAME)

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
	err := c.redisClient.XGroupCreateMkStream(c.ctx, common.STREAM_KEY, common.GROUP_NAME, "0").Err()
	if err != nil {
		// If group already exists, that's fine
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			log.Printf("[INFO] Consumer group %s already exists", common.GROUP_NAME)
			return nil
		}
		return err
	}
	log.Printf("[INFO] Created consumer group: %s", common.GROUP_NAME)
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

// parseMap64 parses JSON string to map[string]float64
func parseMap64(val interface{}) map[string]float64 {
	if val == nil {
		return nil
	}

	jsonStr, ok := val.(string)
	if !ok {
		return nil
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &resultMap); err != nil {
		log.Printf("[ERROR] Failed to parse JSON map: %v", err)
		return nil
	}

	result := make(map[string]float64)
	for k, v := range resultMap {
		switch num := v.(type) {
		case float64:
			result[k] = num
		case string:
			if num, err := strconv.ParseFloat(num, 64); err == nil {
				result[k] = num
			} else {
				log.Printf("[ERROR] Failed to parse float64 for key %s: %v", k, err)
			}
		case int:
			result[k] = float64(num)
		case int64:
			result[k] = float64(num)
		default:
			log.Printf("[WARN] Unsupported type for key %s: %T", k, v)
		}
	}

	return result
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
			// log.Printf("[DEBUG] Attempting to read from stream group %s, consumer %s", common.GROUP_NAME, common.CONSUMER_NAME)

			streams, err := c.redisClient.XReadGroup(c.ctx, &redis.XReadGroupArgs{
				Group:    common.GROUP_NAME,
				Consumer: common.CONSUMER_NAME,
				Streams:  []string{common.STREAM_KEY, ">"},
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
						if err := c.redisClient.XAck(c.ctx, common.STREAM_KEY, common.GROUP_NAME, message.ID).Err(); err != nil {
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
	//log.Printf("[DEBUG] Original message values: %+v", message.Values)

	priceStr := getStringValue(message.Values["price"])
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		log.Printf("[ERROR] Failed to parse price: %v", err)
		return err
	}

	line1Str := getStringValue(message.Values["line1"])
	line1, err := strconv.ParseFloat(line1Str, 64)
	if err != nil {
		log.Printf("[ERROR] Failed to parse line1: %v", err)
		return err
	}

	line2Str := getStringValue(message.Values["line2"])
	line2, err := strconv.ParseFloat(line2Str, 64)
	if err != nil {
		log.Printf("[ERROR] Failed to parse line2: %v", err)
		return err
	}

	signal := &common.StreamSignal{
		Timestamp:           extractTimestampFromID(message.ID),
		InstID:              getStringValue(message.Values["inst_id"]),
		Bar:                 getStringValue(message.Values["bar"]),
		Prediction:          getStringValue(message.Values["prediction"]),
		PredictionLabel:     getStringValue(message.Values["prediction_label"]),
		PredictionHigh:      getStringValue(message.Values["prediction_high"]),
		PredictionHighLabel: getStringValue(message.Values["prediction_high_label"]),
		PredictionLow:       getStringValue(message.Values["prediction_low"]),
		PredictionLowLabel:  getStringValue(message.Values["prediction_low_label"]),
		Probabilities:       parseMap64(message.Values["probabilities"]),
		ProbabilitiesHigh:   parseMap64(message.Values["probabilities_high"]),
		ProbabilitiesLow:    parseMap64(message.Values["probabilities_low"]),
		FeaturesCount:       getStringValue(message.Values["features_count"]),
		Price:               price,
		Line1:               line1,
		Line2:               line2,
	}

	if err := c.validateStreamSignal(signal); err != nil {
		return fmt.Errorf("stream signal validation failed: %w", err)
	}

	log.Printf("[INFO] Stream signal received: id=%s, inst=%s, prediction=%s, price=%.2f, Probability=%.2f",
		message.ID, signal.InstID, signal.Prediction, signal.Price, signal.Probabilities[signal.Prediction])

	if c.messageHandler != nil {
		if err := c.messageHandler(signal); err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	return nil
}

// validateStreamSignal validates the stream signal
func (c *StreamConsumer) validateStreamSignal(signal *common.StreamSignal) error {
	if signal.Timestamp == 0 {
		return fmt.Errorf("timestamp is required")
	}

	if signal.InstID == "" {
		return fmt.Errorf("inst_id is required")
	}

	if signal.Prediction == "" {
		return fmt.Errorf("prediction is required")
	}

	if signal.Price == 0 {
		return fmt.Errorf("price is required")
	}

	return nil
}

// ExtractTimestampFromID 从消息ID提取时间戳
func extractTimestampFromID(msgID string) int64 {
	// Redis Stream 消息ID格式: "timestamp-sequence"
	parts := strings.Split(msgID, "-")
	if len(parts) < 1 {
		return 0
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}

	return timestamp
}


