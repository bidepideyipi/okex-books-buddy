package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/signal"
)

/**
 * NewPrivateMessageHandler creates a message handler for private WebSocket
 * @description Processes private WebSocket messages
 * @param mongoClient MongoDB client for inserting orders and positions
 * @param orderProcessor Order processor
 */
func NewPrivateMessageHandler(mongoClient *mongodb.Client, orderProcessor *signal.OrderProcessor, postionProcessor *signal.PositionProcessor) common.MessageHandler {
	return func(msg []byte) error {
		//在这里写入了数据 到 MongoDB
		if err := saveMessage(mongoClient, msg); err != nil {
			return err
		}

		//在这里处理了事件
		if orderProcessor != nil && postionProcessor != nil {
			if err := handleEvent(orderProcessor, postionProcessor, msg); err != nil {
				return err
			}
		}

		return nil
	}
}

/**
 * saveMessage processes private WebSocket messages without creating a handler object
 * @description Processes private WebSocket messages
 * @param mongoClient MongoDB client for inserting orders and positions
 * @param message Private WebSocket message
 * @return error If insertion fails
 */
func saveMessage(mongoClient *mongodb.Client, message []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	arg, ok := msg["arg"].(map[string]interface{})
	if !ok {
		return nil
	}

	channel, ok := arg["channel"].(string)
	if !ok {
		return nil
	}

	data, ok := msg["data"].([]interface{})
	if !ok {
		return nil
	}

	switch channel {
	case "orders":
		return saveOrders(mongoClient, data)
	case "positions":
		return savePositions(mongoClient, data)
	default:
		log.Printf("Unknown private channel: %s", channel)
	}

	return nil
}

/**
 * saveOrders processes order channel data
 * @description Inserts orders into MongoDB
 * @param mongoClient MongoDB client for inserting orders
 * @param data Order channel data
 * @return error If insertion fails
 */
func saveOrders(mongoClient *mongodb.Client, data []interface{}) error {
	for _, item := range data {
		orderMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		order, err := parseOrder(orderMap)
		if err != nil {
			log.Printf("Failed to parse order: %v", err)
			continue
		}

		if err := mongoClient.InsertOrder(order); err != nil {
			log.Printf("Failed to insert order: %v", err)
		}
	}
	return nil
}

// parseOrder parses order data from WebSocket message
func parseOrder(orderMap map[string]interface{}) (*mongodb.Order, error) {
	order := &mongodb.Order{
		Timestamp: time.Now().UnixMilli(),
	}

	if instID, ok := orderMap["instId"].(string); ok {
		order.InstID = instID
	}

	if ordID, ok := orderMap["ordId"].(string); ok {
		order.OrdID = ordID
	}

	if clOrdID, ok := orderMap["clOrdId"].(string); ok {
		order.ClOrdID = clOrdID
	}

	if tag, ok := orderMap["tag"].(string); ok {
		order.Tag = tag
	}

	if side, ok := orderMap["side"].(string); ok {
		order.Side = side
	}

	if ordType, ok := orderMap["ordType"].(string); ok {
		order.OrdType = ordType
	}

	if posSide, ok := orderMap["posSide"].(string); ok {
		order.PosSide = posSide
	}

	if state, ok := orderMap["state"].(string); ok {
		order.State = state
	}

	if sz, ok := orderMap["sz"].(string); ok {
		order.Sz = sz
	}

	if px, ok := orderMap["px"].(string); ok {
		order.Px = px
	}

	if lever, ok := orderMap["lever"].(string); ok {
		order.Lever = lever
	}

	if tm, ok := orderMap["tm"].(string); ok {
		order.Tm = tm
	}

	if cTime, ok := orderMap["cTime"].(string); ok {
		order.CTime = cTime
	}

	if uTime, ok := orderMap["uTime"].(string); ok {
		order.UTime = uTime
	}

	if reqID, ok := orderMap["reqId"].(string); ok {
		order.ReqID = reqID
	}

	if fee, ok := orderMap["fee"].(string); ok {
		order.Fee = fee
	}

	if fillSz, ok := orderMap["accFillSz"].(string); ok {
		order.FillSz = fillSz
	}

	if fillPx, ok := orderMap["avgPx"].(string); ok {
		order.FillPx = fillPx
	}

	if fillTime, ok := orderMap["fillTime"].(string); ok {
		order.FillTime = fillTime
	}

	if fillNotionalUsd, ok := orderMap["fillNotionalUsd"].(string); ok {
		order.FillNotionalUSD = fillNotionalUsd
	}

	if pnl, ok := orderMap["pnl"].(string); ok {
		order.Pnl = pnl
	}

	if pnlRatio, ok := orderMap["pnlRatio"].(string); ok {
		order.PnlRatio = pnlRatio
	}

	if category, ok := orderMap["category"].(string); ok {
		order.Category = category
	}

	order.ID = order.OrdID

	return order, nil
}

/**
 * savePositions processes position channel data
 * @description Inserts positions into MongoDB
 * @param mongoClient MongoDB client for inserting positions
 * @param data Position channel data
 * @return error If insertion fails
 */
func savePositions(mongoClient *mongodb.Client, data []interface{}) error {
	positions, _ := mongoClient.GetActivePositions()

	for _, item := range data {
		posMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		pos, err := parsePosition(posMap)
		if err != nil {
			log.Printf("Failed to parse position: %v", err)
			continue
		}

		//写入DB
		if err := mongoClient.InsertPosition(pos); err != nil {
			log.Printf("Failed to insert position: %v", err)
		}

		//排除当前确实是Active的Position
		for i, dbPos := range positions {
			if pos.PosID == dbPos.PosID {
				// 用最后一个元素替换当前元素
				positions[i] = positions[len(positions)-1]
				positions = positions[:len(positions)-1]
			}
		}
	}

	//排除后的就是!Active的Position
	for _, dbPos := range positions {
		mongoClient.SoftDeletePosition(dbPos.InstID, dbPos.PosID)
	}

	return nil
}

// parsePosition parses position data from WebSocket message
func parsePosition(posMap map[string]interface{}) (*mongodb.Position, error) {
	position := &mongodb.Position{
		Timestamp: time.Now().UnixMilli(),
	}

	if instID, ok := posMap["instId"].(string); ok {
		position.InstID = instID
	}

	if mgnMode, ok := posMap["mgnMode"].(string); ok {
		position.MgnMode = mgnMode
	}

	if posID, ok := posMap["posId"].(string); ok {
		position.PosID = posID
	}

	if posSide, ok := posMap["posSide"].(string); ok {
		position.PosSide = posSide
	}

	if pos, ok := posMap["pos"].(string); ok {
		position.Pos = pos
	}

	if availPos, ok := posMap["availPos"].(string); ok {
		position.AvailPos = availPos
	}

	if baseBal, ok := posMap["baseBal"].(string); ok {
		position.BaseBal = baseBal
	}

	if quoteBal, ok := posMap["quoteBal"].(string); ok {
		position.QuoteBal = quoteBal
	}

	if posCcy, ok := posMap["posCcy"].(string); ok {
		position.PosCcy = posCcy
	}

	if pnlRatio, ok := posMap["pnlRatio"].(string); ok {
		position.PnlRatio = pnlRatio
	}

	if upl, ok := posMap["upl"].(string); ok {
		position.Upl = upl
	}

	if uplRatio, ok := posMap["uplRatio"].(string); ok {
		position.UplRatio = uplRatio
	}

	if lever, ok := posMap["lever"].(string); ok {
		position.Lever = lever
	}

	if liqPx, ok := posMap["liqPx"].(string); ok {
		position.LiqPx = liqPx
	}

	if markPx, ok := posMap["markPx"].(string); ok {
		position.MarkPx = markPx
	}

	if cTime, ok := posMap["cTime"].(string); ok {
		position.CTime = cTime
	}

	if uTime, ok := posMap["uTime"].(string); ok {
		position.UTime = uTime
	}

	if adl, ok := posMap["adl"].(string); ok {
		position.ADL = adl
	}

	if notionalUSD, ok := posMap["notionalUsd"].(string); ok {
		position.NotionalUSD = notionalUSD
	}

	if last, ok := posMap["last"].(string); ok {
		position.Last = last
	}

	position.ID = fmt.Sprintf("%s_%s", position.InstID, position.PosID)

	return position, nil
}

func handleEvent(orderProcessor *signal.OrderProcessor, postionProcessor *signal.PositionProcessor, message []byte) error {
	//log.Printf("[INFO] %s", string(message))
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if event, ok := msg["event"].(string); ok {
		switch event {
		case "channel-conn-count":
			log.Printf("[INFO] %s", string(message))
		case "subscribe":
			if arg, ok := msg["arg"].(map[string]interface{}); ok {
				if channel, ok := arg["channel"].(string); ok {
					log.Printf("[INFO] Subscribe successful: channel=%s", channel)
				}
			}
		case "unsubscribe":
			if arg, ok := msg["arg"].(map[string]interface{}); ok {
				if channel, ok := arg["channel"].(string); ok {
					log.Printf("[INFO] Unsubscribe successful: channel=%s", channel)
				}
			}
		case "error":
			//TODO Test error response
			log.Printf("[ERROR] Event error: %v", msg)
			orderProcessor.HandleErrorResponse(message)
		default:
			log.Printf("[WARNING] Unknown event: %s", event)
		}
		return nil
	}

	arg, ok := msg["arg"].(map[string]interface{})
	if !ok {
		return nil
	}

	channel, ok := arg["channel"].(string)
	if !ok {
		return nil
	}

	data, ok := msg["data"].([]interface{})
	if !ok {
		return nil
	}

	switch channel {
	case "orders":
		return orderProcessor.HandleOrderEvent(data)
	case "positions":
		return postionProcessor.HandleEvent(data)
	default:
		log.Printf("Unknown private channel: %s", channel)
	}

	return nil
}
