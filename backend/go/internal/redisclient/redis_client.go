package redisclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps Redis operations for the system
type Client struct {
	rdb *redis.Client
	ctx context.Context
}

// NewClient creates a new Redis client
func NewClient(addr, password string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		rdb: rdb,
		ctx: ctx,
	}, nil
}

// GetTradingPairs returns the set of trading pairs from Redis
func (c *Client) GetTradingPairs(key string) ([]string, error) {
	members, err := c.rdb.SMembers(c.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get trading pairs from Redis: %w", err)
	}
	return members, nil
}

// PublishOrderBookEvent publishes an order book event to Redis List
func (c *Client) PublishOrderBookEvent(listKey string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := c.rdb.LPush(c.ctx, listKey, data).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis list: %w", err)
	}

	return nil
}

// StoreOrderBookSnapshot stores the latest order book snapshot in Redis Hash
func (c *Client) StoreOrderBookSnapshot(instID string, asks, bids interface{}, checksum int32) error {
	hashKey := fmt.Sprintf("orderbook:%s", instID)

	asksJSON, err := json.Marshal(asks)
	if err != nil {
		return fmt.Errorf("failed to marshal asks: %w", err)
	}

	bidsJSON, err := json.Marshal(bids)
	if err != nil {
		return fmt.Errorf("failed to marshal bids: %w", err)
	}

	// Store in Redis Hash
	fields := map[string]interface{}{
		"instrument_id": instID,
		"timestamp":     time.Now().Unix(),
		"asks":          string(asksJSON),
		"bids":          string(bidsJSON),
		"checksum":      checksum,
	}

	if err := c.rdb.HSet(c.ctx, hashKey, fields).Err(); err != nil {
		return fmt.Errorf("failed to store order book snapshot: %w", err)
	}

	return nil
}

// StoreSupportResistance stores support and resistance levels for an instrument in Redis Hash
func (c *Client) StoreSupportResistance(instID string, supports, resistances []float64) error {
	hashKey := fmt.Sprintf("analysis:support_resistance:%s", instID)

	fields := map[string]interface{}{
		"instrument_id": instID,
		"analysis_time": time.Now().Unix(),
	}

	if len(supports) > 0 {
		fields["support_level_1"] = supports[0]
	}
	if len(supports) > 1 {
		fields["support_level_2"] = supports[1]
	}
	if len(resistances) > 0 {
		fields["resistance_level_1"] = resistances[0]
	}
	if len(resistances) > 1 {
		fields["resistance_level_2"] = resistances[1]
	}

	if err := c.rdb.HSet(c.ctx, hashKey, fields).Err(); err != nil {
		return fmt.Errorf("failed to store support/resistance levels: %w", err)
	}

	return nil
}

// StoreLargeOrderDistribution stores large order distribution metrics in Redis Hash
func (c *Client) StoreLargeOrderDistribution(instID string, largeBuyNotional, largeSellNotional, sentiment float64) error {
	hashKey := fmt.Sprintf("analysis:large_orders:%s", instID)

	trend := "neutral"
	if sentiment > 0.3 {
		trend = "bullish"
	} else if sentiment < -0.3 {
		trend = "bearish"
	}

	fields := map[string]interface{}{
		"instrument_id":     instID,
		"analysis_time":     time.Now().Unix(),
		"large_buy_orders":  largeBuyNotional,
		"large_sell_orders": largeSellNotional,
		"sentiment":         sentiment,
		"large_order_trend": trend,
	}

	if err := c.rdb.HSet(c.ctx, hashKey, fields).Err(); err != nil {
		return fmt.Errorf("failed to store large order distribution: %w", err)
	}

	return nil
}

// GetHash returns all fields of a Redis hash as a map
func (c *Client) GetHash(key string) (map[string]string, error) {
	result, err := c.rdb.HGetAll(c.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get hash %s: %w", key, err)
	}
	return result, nil
}

// UpdateSystemMonitoring updates system monitoring metrics in Redis
func (c *Client) UpdateSystemMonitoring(fields map[string]interface{}) error {
	if err := c.rdb.HSet(c.ctx, "system:monitoring", fields).Err(); err != nil {
		return fmt.Errorf("failed to update system monitoring: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}
