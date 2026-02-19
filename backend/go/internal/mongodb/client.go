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

// Client wraps MongoDB client
type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

// ConfigItem represents a configuration item in the config collection
type ConfigItem struct {
	ID    string `bson:"_id,omitempty"`
	Item  string `bson:"item"`
	Key   string `bson:"key"`
	Value string `bson:"value"`
	Desc  string `bson:"desc"`
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

// Order represents an OKEx order
type Order struct {
	ID            string  `bson:"_id,omitempty"`
	InstID        string  `bson:"inst_id"`
	OrdID         string  `bson:"ord_id"`
	ClOrdID       string  `bson:"cl_ord_id"`
	Tag           string  `bson:"tag"`
	SignalID      string  `bson:"signal_id,omitempty"`
	Side          string  `bson:"side"`
	OrdType       string  `bson:"ord_type"`
	PosSide       string  `bson:"pos_side"`
	State         string  `bson:"state"`
	Sz            string  `bson:"sz"`
	Px            string  `bson:"px"`
	Lever         string  `bson:"lever"`
	Tm            string  `bson:"tm"`
	CTime         string  `bson:"c_time"`
	UTime         string  `bson:"u_time"`
	ReqID         string  `bson:"req_id,omitempty"`
	Fee           string  `bson:"fee,omitempty"`
	FillSz        string  `bson:"fill_sz"`
	FillPx        string  `bson:"fill_px"`
	FillTime      string  `bson:"fill_time"`
	FillNotionalUSD string `bson:"fill_notional_usd"`
	Pnl           string  `bson:"pnl,omitempty"`
	PnlRatio      string  `bson:"pnl_ratio,omitempty"`
	Category      string  `bson:"category"`
	Timestamp     int64   `bson:"timestamp"`
}

// Position represents an OKEx position
type Position struct {
	ID             string  `bson:"_id,omitempty"`
	InstID         string  `bson:"inst_id"`
	MgnMode        string  `bson:"mgn_mode"`
	PosID          string  `bson:"pos_id"`
	PosSide        string  `bson:"pos_side"`
	Pos            string  `bson:"pos"`
	BaseBal        string  `bson:"base_bal"`
	QuoteBal       string  `bson:"quote_bal"`
	PosCcy         string  `bson:"pos_ccy"`
	PnlRatio       string  `bson:"pnl_ratio"`
	Upl            string  `bson:"upl"`
	UplRatio       string  `bson:"upl_ratio"`
	Lever          string  `bson:"lever"`
	LiqPx          string  `bson:"liq_px"`
	MarkPx         string  `bson:"mark_px"`
	CTime          string  `bson:"c_time"`
	UTime          string  `bson:"u_time"`
	ADL            string  `bson:"adl"`
	NotionalUSD    string  `bson:"notional_usd"`
	Last           string  `bson:"last"`
	Timestamp      int64   `bson:"timestamp"`
}

// TradingSignal represents a trading signal
type TradingSignal struct {
	ID               string  `bson:"_id,omitempty"`
	SignalID         string  `bson:"signal_id"`
	StrategyName     string  `bson:"strategy_name"`
	InstID           string  `bson:"inst_id"`
	Side             string  `bson:"side"`
	OrdType          string  `bson:"ord_type"`
	PosSide          string  `bson:"pos_side"`
	Sz               string  `bson:"sz"`
	Px               string  `bson:"px"`
	ReduceOnly       bool    `bson:"reduce_only"`
	Status           string  `bson:"status"`
	OrdID            string  `bson:"ord_id,omitempty"`
	ClOrdID          string  `bson:"cl_ord_id,omitempty"`
	ErrorMsg         string  `bson:"error_msg,omitempty"`
	SignalTimestamp  int64   `bson:"signal_timestamp"`
	ProcessTimestamp int64   `bson:"process_timestamp"`
	OrderTimestamp   int64   `bson:"order_timestamp"`
	CreatedAt        string  `bson:"created_at"`
	UpdatedAt        string  `bson:"updated_at"`
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

// GetConfigValue retrieves a configuration value from the config collection
func (c *Client) GetConfigValue(item, key string) (string, error) {
	collection := c.database.Collection("config")

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
	collection := c.database.Collection("orders")

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
	collection := c.database.Collection("positions")

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
	collection := c.database.Collection("trading_signals")

	_, err := collection.InsertOne(context.Background(), signal)
	return err
}

// UpdateTradingSignal updates a trading signal record
func (c *Client) UpdateTradingSignal(signalID string, update bson.M) error {
	collection := c.database.Collection("trading_signals")

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
