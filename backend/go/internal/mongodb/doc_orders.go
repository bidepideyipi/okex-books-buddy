package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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