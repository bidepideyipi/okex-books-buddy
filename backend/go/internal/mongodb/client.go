package mongodb

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewClient creates a new MongoDB client
func NewClient(addr string, dbName string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(addr))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	database := client.Database(dbName)
	log.Printf("Connected to MongoDB at %s, database: %s", addr, dbName)

	return &Client{
		client:   client,
		database: database,
	}, nil
}

// InsertCandlestick inserts or updates a candlestick record
func (c *Client) InsertCandlestick(candle *Candlestick) error {
	collection := c.database.(*mongo.Database).Collection("candlesticks")

	filter := bson.M{
		"inst_id":  candle.InstrumentID,
		"bar":      candle.Bar,
		"timestamp": candle.Timestamp,
	}

	update := bson.M{"$set": candle}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	return err
}

// Close closes the MongoDB connection
func (c *Client) Close() error {
	return c.client.(*mongo.Client).Disconnect(context.Background())
}

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

// InsertOrder inserts or updates an order record
func (c *Client) InsertOrder(order *Order) error {
	collection := c.database.(*mongo.Database).Collection("orders")

	filter := bson.M{
		"ord_id": order.OrdID,
	}

	update := bson.M{"$set": order}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	return err
}

// InsertPosition inserts or updates a position record
func (c *Client) InsertPosition(position *Position) error {
	collection := c.database.(*mongo.Database).Collection("positions")

	filter := bson.M{
		"inst_id": position.InstID,
		"pos_id":  position.PosID,
	}

	update := bson.M{"$set": position}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	return err
}

// InsertTradingSignal inserts a new trading signal record
func (c *Client) InsertTradingSignal(signal *TradingSignal) error {
	collection := c.database.(*mongo.Database).Collection("trading_signals")

	_, err := collection.InsertOne(context.Background(), signal)
	return err
}

// UpdateTradingSignal updates a trading signal record
func (c *Client) UpdateTradingSignal(signalID string, update bson.M) error {
	collection := c.database.(*mongo.Database).Collection("trading_signals")

	filter := bson.M{
		"signal_id": signalID,
	}

	_, err := collection.UpdateOne(context.Background(), filter, bson.M{"$set": update})
	return err
}

// UpdateSignalWithOrderID updates a signal with order ID and status
func (c *Client) UpdateSignalWithOrderID(signalID, ordID, clOrdID, status string) error {
	update := bson.M{
		"ord_id":    ordID,
		"cl_ord_id": clOrdID,
		"status":    status,
		"updated_at": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	return c.UpdateTradingSignal(signalID, update)
}

// UpdateSignalStatus updates only the status of a trading signal
func (c *Client) UpdateSignalStatus(signalID, status string) error {
	update := bson.M{
		"status":    status,
		"updated_at": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	return c.UpdateTradingSignal(signalID, update)
}

// UpdateSignalStatusWithError updates signal status with error message
func (c *Client) UpdateSignalStatusWithError(signalID, status, errorMsg string) error {
	update := bson.M{
		"status":    status,
		"error_msg": errorMsg,
		"updated_at": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	return c.UpdateTradingSignal(signalID, update)
}
