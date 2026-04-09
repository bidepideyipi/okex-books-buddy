package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertPosition inserts or updates a position record
func (c *Client) InsertPosition(position *Position) error {
	collection := c.database.(*mongo.Database).Collection("positions")

	filter := bson.M{
		"inst_id": position.InstID,
		"pos_id":  position.PosID,
	}

	position.Active = 1
	update := bson.M{"$set": position}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.Background(), filter, update, opts)
	return err
}

// SoftDeletePosition soft deletes a position by setting active to 0
func (c *Client) SoftDeletePosition(instID, posID string) error {
	collection := c.database.(*mongo.Database).Collection("positions")

	filter := bson.M{
		"inst_id": instID,
		"pos_id":  posID,
	}

	update := bson.M{"$set": bson.M{"active": 0}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	return err
}

// GetActivePositions retrieves all active positions (active = 1)
func (c *Client) GetActivePositions() ([]Position, error) {
	collection := c.database.(*mongo.Database).Collection("positions")

	filter := bson.M{"active": 1}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var positions []Position
	if err := cursor.All(context.Background(), &positions); err != nil {
		return nil, err
	}

	return positions, nil
}

// GetActivePositions retrieves all active positions (active = 1)
func (c *Client) GetActivePosByInstIDAndPoSide(instID string, posSide string) ([]Position, error) {
	collection := c.database.(*mongo.Database).Collection("positions")

	filter := bson.M{"active": 1, "inst_id": instID, "pos_side": posSide}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var positions []Position
	if err := cursor.All(context.Background(), &positions); err != nil {
		return nil, err
	}

	return positions, nil
}
