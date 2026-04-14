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

/*
*
# 止盈止损策略下单
POST /api/v5/trade/order-algo
body

	{
	    "instId":"BTC-USDT",
	    "tdMode":"cross",
	    "side":"buy",
	    "ordType":"conditional",
	    "sz":"2",
	    "tpTriggerPx":"15",
	    "tpOrdPx":"18"
	}

# 移动止盈止损策略下单
POST /api/v5/trade/order-algo
body

	{
	    "instId": "BTC-USDT-SWAP",
	    "tdMode": "cross",
	    "side": "buy",
	    "ordType": "move_order_stop",
	    "sz": "10",
	    "posSide": "net",
	    "callbackRatio": "0.05",
	    "reduceOnly": true
	}
*/
type PlaceAlgoOrderRequest struct {
	InstID      string `json:"instId"`
	TdMode      string `json:"tdMode"` //isolated/cross
	Ccy         string `json:"ccy,omitempty"`
	ClOrdID     string `json:"clOrdId"`
	Tag         string `json:"tag,omitempty"`
	Side        string `json:"side"`    //buy/sell
	PosSide     string `json:"posSide"` //long/short
	OrdType     string `json:"ordType"` //limit/post_only/fok/ioc/optimal_limit_ioc
	Sz          string `json:"sz"`
	TpTriggerPx string `json:"tpTriggerPx,omitempty"`
	SlTriggerPx string `json:"slTriggerPx,omitempty"`
	TpOrdPx     string `json:"tpOrdPx,omitempty"`
	SlOrdPx     string `json:"slOrdPx,omitempty"`
}

type OrdersPendingRequest struct {
	InstType string `json:"instType,omitempty"`
	InstID   string `json:"instId,omitempty"`
}

type OrdersPendingResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		InstType  string `json:"instType"`
		InstID    string `json:"instId"`
		OrdId     string `json:"ordId"`
		ClOrdId   string `json:"clOrdId"`
		Px        string `json:"px"`
		Sz        string `json:"sz"`
		Pnl       string `json:"pnl"`
		OrdType   string `json:"ordType"`
		Side      string `json:"side"`
		PosSide   string `json:"posSide"`
		AccFillSz string `json:"accFillSz"`
		FillPx    string `json:"fillPx"`
		FillSz    string `json:"fillSz"`
		State     string `json:"state"`
		UTime     string `json:"uTime"`
		CTime     string `json:"cTime"`
	} `json:"data"`
}

type CancelOrderRequest struct {
	InstID  string `json:"instId"`
	OrdId   string `json:"ordId"`
	ClOrdID string `json:"clOrdId,omitempty"`
}

type AmendOrderRequest struct {
	InstID         string              `json:"instId"`
	CxlOnFail      string              `json:"cxlOnFail,omitempty"`
	OrdId          string              `json:"ordId,omitempty"`
	ClOrdID        string              `json:"clOrdId,omitempty"`
	NewSz          string              `json:"newSz,omitempty"`
	NewPx          string              `json:"newPx,omitempty"`
	AttachAlgoOrds []map[string]string `json:"attachAlgoOrds,omitempty"`
}

// PlaceOrderResponse represents the response from placing an order
type OrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		OrdID          string              `json:"ordId"`
		ClOrdID        string              `json:"clOrdId"`
		Tag            string              `json:"tag"`
		SCode          string              `json:"sCode"`
		SMsg           string              `json:"sMsg"`
		AttachAlgoOrds []map[string]string `json:"attachAlgoOrds,omitempty"`
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
