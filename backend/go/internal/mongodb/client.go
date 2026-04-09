package mongodb

import (
	"context"
	"log"
	"time"

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

// Close closes the MongoDB connection
func (c *Client) Close() error {
	return c.client.(*mongo.Client).Disconnect(context.Background())
}
