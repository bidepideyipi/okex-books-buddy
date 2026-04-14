package common

/**
*  定义了两种信号源
*  1、从Redsi list中消费信号
*  2、从Redis Stream中消费信号
 */

const (
	//Redis key group
	LIST_KEY      = "orders"
	STREAM_KEY    = "signals"
	GROUP_NAME    = "gp-go"
	CONSUMER_NAME = "consumer-go"

	//OKE API group
	OKEX_API_BASE_URL     = "https://www.okx.com"
	PLACE_ORDER_PATH      = "/api/v5/trade/order"
	ORDERS_PENDING_PATH   = "/api/v5/trade/orders-pending"
	CANCEL_ORDER_PATH     = "/api/v5/trade/cancel-order"
	AMEND_ORDER_PATH      = "/api/v5/trade/amend-order"
	PLACE_ORDER_ALGO_PATH = "/api/v5/trade/order-algo"
)
