package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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
		"ord_id":     ordID,
		"cl_ord_id":  clOrdID,
		"status":     status,
		"updated_at": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	return c.UpdateTradingSignal(signalID, update)
}

// UpdateSignalStatus updates only the status of a trading signal
func (c *Client) UpdateSignalStatus(signalID, status string) error {
	update := bson.M{
		"status":     status,
		"updated_at": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	return c.UpdateTradingSignal(signalID, update)
}

// UpdateSignalStatusWithError updates signal status with error message
func (c *Client) UpdateSignalStatusWithError(signalID, status, errorMsg string) error {
	update := bson.M{
		"status":     status,
		"error_msg":  errorMsg,
		"updated_at": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	return c.UpdateTradingSignal(signalID, update)
}
