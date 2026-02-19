package subscription

import (
	"log"
	"time"

	"github.com/supermancell/okex-buddy/internal/common"
)

// SubscriptionManager manages dynamic subscription changes based on Redis config
type SubscriptionManager struct {
	client       common.WSClientInterface
	redisClient  RedisConfigReader
	configKey    string
	pollInterval time.Duration
	stopChan     chan struct{}
}

// RedisConfigReader interface for reading trading pairs from Redis
type RedisConfigReader interface {
	GetTradingPairs(key string) ([]string, error)
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(client common.WSClientInterface, redisClient RedisConfigReader, configKey string, pollInterval int) *SubscriptionManager {
	return &SubscriptionManager{
		client:       client,
		redisClient:  redisClient,
		configKey:    configKey,
		pollInterval: time.Duration(pollInterval) * time.Second,
		stopChan:     make(chan struct{}),
	}
}

// Start initializes subscriptions and starts polling for config changes
func (sm *SubscriptionManager) Start() error {
	// Initial subscription from Redis config
	if err := sm.syncSubscriptions(); err != nil {
		return err
	}

	// Start polling goroutine
	go sm.pollConfigChanges()

	return nil
}

// Stop stops the subscription manager
func (sm *SubscriptionManager) Stop() {
	close(sm.stopChan)
}

// pollConfigChanges polls Redis for config changes every pollInterval
func (sm *SubscriptionManager) pollConfigChanges() {
	ticker := time.NewTicker(sm.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			if err := sm.syncSubscriptions(); err != nil {
				log.Printf("Error syncing subscriptions: %v", err)
			}
		}
	}
}

// syncSubscriptions synchronizes current subscriptions with Redis config
func (sm *SubscriptionManager) syncSubscriptions() error {
	// Get latest config from Redis
	latestPairs, err := sm.redisClient.GetTradingPairs(sm.configKey)
	if err != nil {
		log.Printf("Failed to read trading pairs from Redis: %v", err)
		return err
	}

	// Enforce max 10 pairs limit
	if len(latestPairs) > 10 {
		log.Printf("WARNING: Config has %d trading pairs, limiting to 10", len(latestPairs))
		latestPairs = latestPairs[:10]
	}

	// Get current subscriptions
	currentPairs := sm.client.GetSubscribed()

	// Calculate differences
	toSubscribe := difference(latestPairs, currentPairs)
	toUnsubscribe := difference(currentPairs, latestPairs)

	// No changes needed
	if len(toSubscribe) == 0 && len(toUnsubscribe) == 0 {
		return nil
	}

	log.Printf("Config changed: subscribing to %d pairs, unsubscribing from %d pairs", len(toSubscribe), len(toUnsubscribe))

	// Unsubscribe first
	if len(toUnsubscribe) > 0 {
		if err := sm.client.Unsubscribe(toUnsubscribe); err != nil {
			log.Printf("Failed to unsubscribe: %v", err)
		}
	}

	// Then subscribe to new ones
	if len(toSubscribe) > 0 {
		if err := sm.client.Subscribe(toSubscribe); err != nil {
			log.Printf("Failed to subscribe: %v", err)
			return err
		}
	}

	return nil
}

// difference returns elements in a that are not in b
func difference(a, b []string) []string {
	mb := make(map[string]bool, len(b))
	for _, x := range b {
		mb[x] = true
	}

	var diff []string
	for _, x := range a {
		if !mb[x] {
			diff = append(diff, x)
		}
	}
	return diff
}
