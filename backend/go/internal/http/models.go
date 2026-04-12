package http

// HealthCheckResponse represents the health check response structure
type HealthCheckResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PublicWebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"public_websocket"`
		PrivateWebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"private_websocket"`
		BusinessWebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"business_websocket"`
		Redis struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"redis"`
	} `json:"data"`
}

// PlaceOrderRequest represents the request body for placing an order
type PlaceOrderRequest struct {
	InstID         string              `json:"instId"`
	TdMode         string              `json:"tdMode"` //isolated/cross
	Ccy            string              `json:"ccy,omitempty"`
	ClOrdID        string              `json:"clOrdId"`
	Tag            string              `json:"tag,omitempty"`
	Side           string              `json:"side"`    //buy/sell
	PosSide        string              `json:"posSide"` //long/short
	OrdType        string              `json:"ordType"` //limit/post_only/fok/ioc/optimal_limit_ioc
	Sz             string              `json:"sz"`
	Px             string              `json:"px,omitempty"`
	ReduceOnly     string              `json:"reduceOnly,omitempty"`
	AttachAlgoOrds []map[string]string `json:"attachAlgoOrds,omitempty"`
}

// PlaceOrderResponse represents the response from placing an order
type PlaceOrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		OrdID   string `json:"ordId"`
		ClOrdID string `json:"clOrdId"`
		Tag     string `json:"tag"`
		SCode   string `json:"sCode"`
		SMsg    string `json:"sMsg"`
	} `json:"data"`
}

// ServerTimeResponse represents the response from server time endpoint
type ServerTimeResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Ts string `json:"ts"`
	} `json:"data"`
}

// PlaceOrderHTTPRequest represents the HTTP request for placing an order
type PlaceOrderHTTPRequest struct {
	InstID     string `json:"instId"`
	TdMode     string `json:"tdMode"`
	Ccy        string `json:"ccy,omitempty"`
	ClOrdID    string `json:"clOrdId"`
	Tag        string `json:"tag,omitempty"`
	Side       string `json:"side"`
	OrdType    string `json:"ordType"`
	Sz         string `json:"sz"`
	Px         string `json:"px,omitempty"`
	ReduceOnly string `json:"reduceOnly,omitempty"`
}

// PlaceOrderHTTPResponse represents the HTTP response for placing an order
type PlaceOrderHTTPResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    *PlaceOrderResponse `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}
