package handler

import (
	"log"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/common"
	rest "github.com/supermancell/okex-buddy/internal/http"
	"github.com/supermancell/okex-buddy/internal/mongodb"
)

var (
	MAX_SIZE float64 = 1.0
	PER_SIZE float64 = 0.1
	SIDE     int8    = 0
)

func NewRedisStreamMessageHandler(mongoClient *mongodb.Client, okRest *rest.OKExHTTPClient) common.StreamSignalHandler {
	return func(msg *common.StreamSignal) error {
		//信号延迟检测
		currentTime := time.Now()
		signalTime := time.UnixMilli(msg.Timestamp)
		timeDiff := currentTime.Sub(signalTime)

		if timeDiff > 2*time.Minute {
			log.Printf("[WARN] Signal delayed: signal_time(%v)=%v, current_time=%v, diff=%v",
				msg.Timestamp, signalTime, currentTime, timeDiff)
			return nil
		}

		//中立信号
		if "3" == msg.Prediction {
			log.Printf("中立信号: %s", msg.Prediction)
			return nil
		}

		//检测ActivePositions
		positions, err := mongoClient.GetActivePositions()
		if err != nil {
			log.Printf("查询MongoDb失败: %v", err)
		}

		var posSz float64 = 0
		for _, p := range positions {
			posSz += p.Pos
		}

		var NEW_SIDE int8 = 0
		//预测值处理
		switch msg.Prediction {
		case "5":
			NEW_SIDE = 1
		case "4":
			NEW_SIDE = 1
		case "2":
			NEW_SIDE = -1
		case "1":
			NEW_SIDE = -1
		default:
			NEW_SIDE = 0
		}

		//方向不同了
		if posSz > 0 && SIDE != 0 && NEW_SIDE != SIDE {
			log.Printf("[INFO] 方向不同了结束交易: inst=%s, prediction=%s, price=%f.2f, posSz=%f",
				msg.InstID, msg.Prediction, msg.Price, posSz)
			closeOrder(okRest, msg.InstID, SIDE, strconv.FormatFloat(posSz, 'f', 1, 64))
			return nil
		}
		//获取交易配置
		eth_max_size, eth_per_size, err := mongoClient.GetTradeSettingsConfig()
		if err != nil {
			log.Printf("查询交易配置失败: %v", err)
			return nil
		}

		//将eth_max_size和eth_per_size转换为float64
		MAX_SIZE, _ = strconv.ParseFloat(eth_max_size, 64)
		PER_SIZE, _ = strconv.ParseFloat(eth_per_size, 64)

		if posSz >= float64(MAX_SIZE) {
			log.Printf("仓位已满，posSz=: %f", posSz)
			return nil
		}

		//光判断Position是不够准确的，还需要判断PendingOrder数量；
		//改成market order后不需要判断pending order数量了
		// pendingOrders, err := okRest.OrdersPending(&rest.OrdersPendingRequest{
		// 	InstType: "SWAP",
		// })

		// if err != nil {
		// 	log.Printf("查询PendingOrders失败: %v", err)
		// 	return nil
		// }

		// pendingSz := float64(len(pendingOrders.Data)) * PER_SIZE
		// //判断PendingOrder数量是否超过最大数量
		// if pendingSz+posSz > MAX_SIZE {
		// 	log.Printf("PendingOrder+Position数量超过最大数量: %v", pendingOrders)
		// 	return nil
		// }

		if msg.Probabilities[msg.Prediction] < 0.65 {
			log.Printf("概率过低: %s, probability=%f", msg.Prediction, msg.Probabilities[msg.Prediction])
			return nil
		}

		//预测值处理
		switch msg.Prediction {
		case "5":
			SIDE = 1
			log.Printf("[INFO] 预测暴涨下多单: inst=%s, prediction=%s, probability=%f, price=%f.2f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2*2, msg.InstID, "long")
			return nil
		case "4":
			SIDE = 1
			log.Printf("[INFO] 预测涨下多单: inst=%s, prediction=%s, probability=%f, price=%f.2f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2, msg.InstID, "long")
			return nil
		case "2":
			SIDE = -1
			log.Printf("[INFO] 预测跌下空单: inst=%s, prediction=%s, probability=%f, price=%f.2f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2, msg.InstID, "short")
			return nil
		case "1":
			SIDE = -1
			log.Printf("[INFO] 预测暴跌下空单: inst=%s, prediction=%s, probability=%f, price=%f.2f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2*2, msg.InstID, "short")
			return nil
		default:
			log.Printf("[WARN] Unknown prediction: %s", msg.Prediction)
		}

		return nil
	}
}

func openOrder(okRest *rest.OKExHTTPClient, price float64, line float64, instId string, posSide string) {
	//如果不自己生成附加域的attachAlgoClOrdId，就要去查一遍pending才能知道attachAlgoOrdId
	attachAlgoClOrdId := rest.GenerateOrderID("ao")

	var slTriggerPx string
	var tpTriggerPx string
	if "long" == posSide {
		slTriggerPx = strconv.FormatFloat(price*(1-line), 'f', 2, 64)
		tpTriggerPx = strconv.FormatFloat(price*(1+line), 'f', 2, 64)
	} else {
		slTriggerPx = strconv.FormatFloat(price*(1+line), 'f', 2, 64)
		tpTriggerPx = strconv.FormatFloat(price*(1-line), 'f', 2, 64)
	}

	algoOrd := map[string]string{
		"attachAlgoClOrdId": attachAlgoClOrdId,
		"slTriggerPx":       slTriggerPx, //止损触发价
		"slOrdPx":           "-1",        //止损委托价
		"slTriggerPxType":   "last",
		"tpTriggerPx":       tpTriggerPx, //止盈触发价
		"tpOrdPx":           "-1",        //止盈委托价
		"tpTriggerPxType":   "last",
	}

	_, err := okRest.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  instId,
		TdMode:  "cross",
		Side:    "buy",
		PosSide: posSide,
		OrdType: "market",
		//Px:             strconv.FormatFloat(price, 'f', 2, 64),
		Sz:             strconv.FormatFloat(PER_SIZE, 'f', 1, 64),
		AttachAlgoOrds: []map[string]string{algoOrd},
	})

	if err != nil {
		log.Printf("开仓失败：%v\n", err)
	}
}

func closeOrder(okRest *rest.OKExHTTPClient, instId string, side int8, sz string) {

	var posSide string
	switch side {
	case 1:
		posSide = "long"
	case -1:
		posSide = "short"
	default:
		return
	}

	_, err := okRest.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  instId,
		TdMode:  "cross",
		Side:    "sell",
		PosSide: posSide,
		OrdType: "market",
		Sz:      sz,
	})

	if err != nil {
		log.Printf("平仓失败：%v\n", err)
	}
}

// func createAlgoOrd(okRest *rest.OKExHTTPClient, price float64, line float64, instId string, posSide string, sz string) {
// 	var slTriggerPx string
// 	var tpTriggerPx string
// 	if "long" == posSide {
// 		slTriggerPx = strconv.FormatFloat(price*(1-line), 'f', 2, 64)
// 		tpTriggerPx = strconv.FormatFloat(price*(1+line), 'f', 2, 64)
// 	} else {
// 		slTriggerPx = strconv.FormatFloat(price*(1+line), 'f', 2, 64)
// 		tpTriggerPx = strconv.FormatFloat(price*(1-line), 'f', 2, 64)
// 	}

// 	_, err := okRest.PlaceAlgoOrder(&rest.PlaceAlgoOrderRequest{
// 		InstID:      instId,
// 		TdMode:      "cross",
// 		Side:        "sell",
// 		PosSide:     posSide,
// 		OrdType:     "conditional",
// 		Sz:          sz,
// 		TpTriggerPx: tpTriggerPx,
// 		TpOrdPx:     "-1",
// 		SlTriggerPx: slTriggerPx,
// 		SlOrdPx:     "-1",
// 	})

// 	if err != nil {
// 		log.Printf("创建止盈止损订单失败：%v\n", err)
// 	}
// }
