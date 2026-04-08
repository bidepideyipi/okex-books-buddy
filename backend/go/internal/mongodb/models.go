package mongodb

// Client wraps MongoDB client
type Client struct {
	client   interface{} // mongo.Client
	database interface{} // mongo.Database
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
	ID           string  `bson:"_id,omitempty"`
	Bar          string  `bson:"bar"`
	InstrumentID string  `bson:"inst_id"`
	Timestamp    int64   `bson:"timestamp"`
	Close        float64 `bson:"close"`
	Confirm      int     `bson:"confirm"`
	DayOfWeek    int     `bson:"day_of_week"`
	High         float64 `bson:"high"`
	Low          float64 `bson:"low"`
	Open         float64 `bson:"open"`
	RecordDT     string  `bson:"record_dt"`
	RecordHour   int     `bson:"record_hour"`
	VolCcy       float64 `bson:"vol_ccy"`
	VolCcyQuote  float64 `bson:"vol_ccy_quote"`
	Volume       float64 `bson:"volume"`
}

// Order represents an OKEx order
type Order struct {
	ID              string `bson:"_id,omitempty"`
	InstID          string `bson:"inst_id"`
	OrdID           string `bson:"ord_id"`
	ClOrdID         string `bson:"cl_ord_id"`
	Tag             string `bson:"tag"`
	SignalID        string `bson:"signal_id,omitempty"`
	Side            string `bson:"side"`
	OrdType         string `bson:"ord_type"`
	PosSide         string `bson:"pos_side"`
	State           string `bson:"state"`
	Sz              string `bson:"sz"`
	Px              string `bson:"px"`
	Lever           string `bson:"lever"`
	Tm              string `bson:"tm"`
	CTime           string `bson:"c_time"`
	UTime           string `bson:"u_time"`
	ReqID           string `bson:"req_id,omitempty"`
	Fee             string `bson:"fee,omitempty"`
	FillSz          string `bson:"fill_sz"`
	FillPx          string `bson:"fill_px"`
	FillTime        string `bson:"fill_time"`
	FillNotionalUSD string `bson:"fill_notional_usd"`
	Pnl             string `bson:"pnl,omitempty"`
	PnlRatio        string `bson:"pnl_ratio,omitempty"`
	Category        string `bson:"category"`
	Timestamp       int64  `bson:"timestamp"`
}

// Position represents an OKEx position
type Position struct {
	ID          string `bson:"_id,omitempty"`
	InstID      string `bson:"inst_id"`
	MgnMode     string `bson:"mgn_mode"`
	PosID       string `bson:"pos_id"`
	PosSide     string `bson:"pos_side"`
	Pos         string `bson:"pos"`
	BaseBal     string `bson:"base_bal"`
	QuoteBal    string `bson:"quote_bal"`
	PosCcy      string `bson:"pos_ccy"`
	PnlRatio    string `bson:"pnl_ratio"`
	Upl         string `bson:"upl"`
	UplRatio    string `bson:"upl_ratio"`
	Lever       string `bson:"lever"`
	LiqPx       string `bson:"liq_px"`
	MarkPx      string `bson:"mark_px"`
	CTime       string `bson:"c_time"`
	UTime       string `bson:"u_time"`
	ADL         string `bson:"adl"`
	NotionalUSD string `bson:"notional_usd"`
	Last        string `bson:"last"`
	Timestamp   int64  `bson:"timestamp"`
}

// TradingSignal represents a trading signal
type TradingSignal struct {
	ID               string `bson:"_id,omitempty"`
	SignalID         string `bson:"signal_id"`
	StrategyName     string `bson:"strategy_name"`
	InstID           string `bson:"inst_id"`
	Side             string `bson:"side"`
	OrdType          string `bson:"ord_type"`
	PosSide          string `bson:"pos_side"`
	Sz               string `bson:"sz"`
	Px               string `bson:"px"`
	ReduceOnly       bool   `bson:"reduce_only"`
	Status           string `bson:"status"`
	OrdID            string `bson:"ord_id,omitempty"`
	ClOrdID          string `bson:"cl_ord_id,omitempty"`
	ErrorMsg         string `bson:"error_msg,omitempty"`
	SignalTimestamp  int64  `bson:"signal_timestamp"`
	ProcessTimestamp int64  `bson:"process_timestamp"`
	OrderTimestamp   int64  `bson:"order_timestamp"`
	CreatedAt        string `bson:"created_at"`
	UpdatedAt        string `bson:"updated_at"`
}
