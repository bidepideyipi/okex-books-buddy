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
)

func NewRedisStreamMessageHandler(mongoClient *mongodb.Client, okxClient *rest.OKExHTTPClient) common.StreamSignalHandler {
	return func(msg *common.StreamSignal) error {
		//信号延迟检测
		currentTime := time.Now()

		timestampMs, err := strconv.ParseInt(msg.Timestamp, 10, 64)
		if err != nil {
			log.Printf("[ERROR] Failed to parse timestamp: %v", err)
			return err
		}

		signalTime := time.UnixMilli(timestampMs)
		timeDiff := currentTime.Sub(signalTime)

		if timeDiff > 1*time.Minute {
			log.Printf("[WARN] Signal delayed: signal_time=%v, current_time=%v, diff=%v",
				signalTime, currentTime, timeDiff)
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

		var sz float64 = 0
		for _, p := range positions {
			pos, err := strconv.ParseFloat(p.Pos, 64)
			if err != nil {
				log.Printf("Failed to parse position: %v", err)
				continue
			}
			sz += pos
		}

		if sz > float64(MAX_SIZE) {
			log.Printf("仓位已满，sz=: %f", sz)
			return nil
		}

		//空仓逻辑
		if sz == 0 {
			//预测值处理
			switch msg.Prediction {
			case "5":
				log.Printf("[INFO] 下多单: inst=%s, prediction=%s, price=%s",
					msg.InstID, msg.Prediction, msg.Price)
			case "4":
				log.Printf("[INFO] 下空单: inst=%s, prediction=%s, price=%s",
					msg.InstID, msg.Prediction, msg.Price)
			case "2":
				log.Printf("[INFO] 横盘，跳过: inst=%s, prediction=%s, price=%s",
					msg.InstID, msg.Prediction, msg.Price)
				return nil
			case "1":
				log.Printf("[INFO] 横盘，跳过: inst=%s, prediction=%s, price=%s",
					msg.InstID, msg.Prediction, msg.Price)
				return nil
			default:
				log.Printf("[WARN] Unknown prediction: %s", msg.Prediction)
			}

		} else { //有仓逻辑
			//TODO
		}

		return nil
	}
}
