package common

/**
* @brief Represents a trading signal from Redis list
* 使用 brpop命令 消费信号，拼接trading_signals
* strategy_name 做为
* 接收到的是一个具体的下单信号, inst_id、方向、数量、价格等
 */
type Signal struct {
	SignalID        string `json:"signal_id"`
	InstID          string `json:"inst_id"`
	Side            string `json:"side"`
	OrdType         string `json:"ord_type"`
	PosSide         string `json:"pos_side"`
	Sz              string `json:"sz"`
	Px              string `json:"px"`
	ReduceOnly      bool   `json:"reduce_only"`
	TPTriggerPx     string `json:"tp_trigger_px"`
	TPTriggerPxType string `json:"tp_trigger_px_type"`
	SlTriggerPx     string `json:"sl_trigger_px"`
	SlTriggerPxType string `json:"sl_trigger_px_type"`
	Ccy             string `json:"ccy"`
	Tag             string `json:"tag"`
	Timestamp       int64  `json:"timestamp"`
}

/**
* @brief Represents a trading signal from Redis Stream
* 接收到的是一个上游分类预测系统给出的值, 包含预测结果、预测概率等
 */
type StreamSignal struct {
	Timestamp           int64
	InstID              string
	Bar                 string
	Prediction          string
	PredictionLabel     string
	PredictionHigh      string
	PredictionHighLabel string
	PredictionLow       string
	PredictionLowLabel  string
	Probabilities       map[string]float64
	ProbabilitiesHigh   map[string]float64
	ProbabilitiesLow    map[string]float64
	FeaturesCount       string
	Price               float64
	Line1               float64
	Line2               float64
}

type WsData struct {
	ID      string                 `json:"id"`
	Op      string                 `json:"op"`
	Code    string                 `json:"code"`
	Msg     string                 `json:"msg"`
	Event   string                 `json:"event"` //有Event就没有Data
	Arg     map[string]interface{} `json:"arg"`
	Data    []interface{}          `json:"data"`
	InTime  string                 `json:"inTime"`
	OutTime string                 `json:"outTime"`
	SCode   string                 `json:"sCode"`
	SMsg    string                 `json:"sMsg"`
}

type WsEvent struct {
	Event     string                 `json:"event"` //有Event就没有Data
	Arg       map[string]interface{} `json:"arg"`
	ConnID    string                 `json:"connId"`
	ConnCount string                 `json:"connCount"`
}

// MessageHandler processes incoming messages
type MessageHandler func(msg []byte) error
type StreamSignalHandler func(sig *StreamSignal) error
type EventHandler func(event *WsEvent) error
type MapHandler func(data []interface{}) error

// WSClientInterface defines the common interface for WebSocket clients
type WSClientInterface interface {
	Subscribe(params interface{}) error
	Unsubscribe(params interface{}) error
	GetSubscribed() []string
}
