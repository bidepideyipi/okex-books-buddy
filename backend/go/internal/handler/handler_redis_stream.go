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
	//推荐的配置，当数据库不存在交易配置时，默认使用该配置
	//遵循了棋经十诀中的“慎勿轻速”，避免因为交易速度过快导致仓位风险变重
	MAX_SIZE    float64 = 2.0
	PER_SIZE    float64 = 0.5
	POS_SIDE    string  = ""
	PER_PRICE   float64 = 0.0
	GRIDE_RTIDO float64 = 0.001 //每网格间距0.1%
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
		var instID string = "ETH-USDT-SWAP"
		var winRatio float64 = 0.0

		for _, p := range positions {
			//判断是否是当前InstID的Position
			if p.InstID != instID {
				continue
			}

			POS_SIDE = p.PosSide
			posSz = p.Pos
			winRatio = p.UplRatio / float64(p.Lever)
			break
		}

		var NEW_SIDE string = ""
		//预测值处理
		switch msg.Prediction {
		case "5":
			NEW_SIDE = "long"
		case "4":
			NEW_SIDE = "long"
		case "2":
			NEW_SIDE = "short"
		case "1":
			NEW_SIDE = "short"
		default:
			NEW_SIDE = POS_SIDE
		}

		//方向不同了
		if posSz > 0 && POS_SIDE != NEW_SIDE {

			//遵循了棋经十诀中的“逢危需弃”，但可能影响到趋势的持续性
			log.Printf("[INFO] 方向不同了结束交易: inst=%s, prediction=%s, price=%f.2f, posSz=%f",
				msg.InstID, msg.Prediction, msg.Price, posSz)
			if CloseOrder(okRest, msg.InstID, POS_SIDE, strconv.FormatFloat(posSz, 'f', 1, 64)) {
				//平仓成功后，重置PER_PRICE
				PER_PRICE = 0.0
			}
			return nil
		}

		//获取交易配置
		eth_max_size, eth_per_size, err := mongoClient.GetTradeSettingsConfig()
		if err != nil {
			log.Printf("查询交易配置失败，使用默认配置: %v", err)
		} else {
			//将eth_max_size和eth_per_size转换为float64
			MAX_SIZE, _ = strconv.ParseFloat(eth_max_size, 64)
			PER_SIZE, _ = strconv.ParseFloat(eth_per_size, 64)
		}

		if posSz >= float64(MAX_SIZE) {
			log.Printf("仓位已满，posSz=: %f", posSz)

			if winRatio >= GRIDE_RTIDO*2 { //减轻压力，增加操作空间
				if CloseOrder(okRest, msg.InstID, POS_SIDE, strconv.FormatFloat(posSz/2, 'f', 1, 64)) {
					//平仓成功后，重置PER_PRICE
					PER_PRICE = 0.0
				}
			}
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

		// 反向相同的情况下消除噪声
		if msg.Probabilities[msg.Prediction] < 0.5 {
			log.Printf("概率过低: %s, probability=%f", msg.Prediction, msg.Probabilities[msg.Prediction])
			return nil
		}

		if msg.Probabilities[msg.Prediction] < 0.75 {
			PER_SIZE = PER_SIZE / 4
			log.Printf("概率低需要谨慎交易: %s, probability=%f, 调低PER_SIZE=%s", msg.Prediction, msg.Probabilities[msg.Prediction], strconv.FormatFloat(PER_SIZE, 'f', 2, 64))
		}

		if POS_SIDE == "long" && PER_PRICE != 0.0 && msg.Price > PER_PRICE*(1-GRIDE_RTIDO) {
			//防止加码追高
			log.Printf("Price=%s 防止频繁加码追多", strconv.FormatFloat(msg.Price, 'f', 2, 64))
			return nil
		}

		if POS_SIDE == "short" && PER_PRICE != 0.0 && msg.Price < PER_PRICE*(1+GRIDE_RTIDO) {
			//防止加码追高
			log.Printf("Price=%s 防止频繁加码追空", strconv.FormatFloat(msg.Price, 'f', 2, 64))
			return nil
		}

		//预测值处理
		switch msg.Prediction {
		case "5":
			log.Printf("[INFO] 预测暴涨下多单: inst=%s, prediction=%s, probability=%f, price=%f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2*2, msg.InstID, "long")
			PER_PRICE = msg.Price
			return nil
		case "4":
			log.Printf("[INFO] 预测涨下多单: inst=%s, prediction=%s, probability=%f, price=%f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2, msg.InstID, "long")
			PER_PRICE = msg.Price
			return nil
		case "2":
			log.Printf("[INFO] 预测跌下空单: inst=%s, prediction=%s, probability=%f, price=%f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2, msg.InstID, "short")
			PER_PRICE = msg.Price
			return nil
		case "1":
			log.Printf("[INFO] 预测暴跌下空单: inst=%s, prediction=%s, probability=%f, price=%f",
				msg.InstID, msg.Prediction, msg.Probabilities[msg.Prediction], msg.Price)
			openOrder(okRest, msg.Price, msg.Line2*2, msg.InstID, "short")
			PER_PRICE = msg.Price
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

	side := "buy"
	if "short" == posSide {
		side = "sell"
	}

	_, err := okRest.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  instId,
		TdMode:  "cross",
		Side:    side,
		PosSide: posSide,
		OrdType: "market",
		//Px:             strconv.FormatFloat(price, 'f', 2, 64),
		Sz:             strconv.FormatFloat(PER_SIZE, 'f', 1, 64),
		AttachAlgoOrds: []map[string]string{algoOrd},
	})

	if err != nil {
		log.Printf("[ERROR] 开仓失败：%v\n", err)
	}

	log.Printf("[NOTICE] 开仓成功: inst=%s, posSide=%s, sz=%s", instId, posSide, strconv.FormatFloat(PER_SIZE, 'f', 1, 64))
}

func CloseOrder(okRest *rest.OKExHTTPClient, instId string, posSide string, sz string) bool {

	log.Printf("[INFO] 平仓: inst=%s, posSide=%s, sz=%s", instId, posSide, sz)

	side := "sell"
	if "short" == posSide {
		side = "buy"
	}

	_, err := okRest.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  instId,
		TdMode:  "cross",
		Side:    side,
		PosSide: posSide,
		OrdType: "market",
		Sz:      sz,
	})

	if err != nil {
		log.Printf("[ERROR] 平仓失败：%v\n", err)
		return false
	}

	log.Printf("[NOTICE] 平仓成功: inst=%s, posSide=%s, sz=%s", instId, posSide, sz)
	return true
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
