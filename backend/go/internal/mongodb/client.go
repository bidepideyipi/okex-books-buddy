package mongodb

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client wraps MongoDB client
type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

// Candlestick represents a candlestick data point
type Candlestick struct {
	ID             string  `bson:"_id,omitempty"`
	Bar            string  `bson:"bar"`
	InstrumentID   string  `bson:"inst_id"`
	Timestamp      int64   `bson:"timestamp"`
	Close          float64 `bson:"close"`
	Confirm        int     `bson:"confirm"`
	DayOfWeek      int     `bson:"day_of_week"`
	High           float64 `bson:"high"`
	Low            float64 `bson:"low"`
	Open           float64 `bson:"open"`
	RecordDT       string  `bson:"record_dt"`
	RecordHour     int     `bson:"record_hour"`
	VolCcy         float64 `bson:"vol_ccy"`
	VolCcyQuote    float64 `bson:"vol_ccy_quote"`
	Volume         float64 `bson:"volume"`
}

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
	collection := c.database.Collection("candlesticks")

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
	return c.client.Disconnect(context.Background())
}
