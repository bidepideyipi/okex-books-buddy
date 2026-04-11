package common

/**
*  定义了两种信号源
*  1、从Redsi list中消费信号
*  2、从Redis Stream中消费信号
 */

const (
	LIST_KEY      = "orders"
	STREAM_KEY    = "signals"
	GROUP_NAME    = "gp-go"
	CONSUMER_NAME = "consumer-go"
)
