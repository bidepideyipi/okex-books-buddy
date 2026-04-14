package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GetConfigValue retrieves a configuration value from the config collection
func (c *Client) GetConfigValue(item, key string) (string, error) {
	collection := c.database.(*mongo.Database).Collection("config")

	filter := bson.M{
		"item": item,
		"key":  key,
	}

	var config ConfigItem
	err := collection.FindOne(context.Background(), filter).Decode(&config)
	if err != nil {
		return "", err
	}

	return config.Value, nil
}

// GetOKExConfig retrieves OKEx API credentials from MongoDB
func (c *Client) GetOKExConfig() (apiKey, secretKey, passphrase string, err error) {
	apiKey, err = c.GetConfigValue("okexAccount", "api_key")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get api_key: %w", err)
	}

	secretKey, err = c.GetConfigValue("okexAccount", "secret_key")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get secret_key: %w", err)
	}

	passphrase, err = c.GetConfigValue("okexAccount", "passphrase")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get passphrase: %w", err)
	}

	return apiKey, secretKey, passphrase, nil
}

func (c *Client) GetTradeSettingsConfig() (eth_max_size, eth_per_size string, err error) {
	eth_max_size, err = c.GetConfigValue("tradeSettings", "eth_max_size")
	if err != nil {
		return "", "", fmt.Errorf("failed to get eth_max_size: %w", err)
	}

	eth_per_size, err = c.GetConfigValue("tradeSettings", "eth_per_size")
	if err != nil {
		return "", "", fmt.Errorf("failed to get eth_per_size: %w", err)
	}

	return eth_max_size, eth_per_size, nil
}
