package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertCandlestick inserts or updates a candlestick record
func (c *Client) InsertCandlestick(candle *Candlestick) error {
	collection := c.database.(*mongo.Database).Collection("candlesticks")

	filter := bson.M{
		"inst_id":   candle.InstrumentID,
		"bar":       candle.Bar,
		"timestamp": candle.Timestamp,
	}

	update := bson.M{"$set": candle}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	return err
}
