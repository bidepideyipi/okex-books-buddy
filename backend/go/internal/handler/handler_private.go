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

// NewPrivateMessageHandler creates a message handler for private WebSocket
func NewPrivateMessageHandler(mongoClient *mongodb.Client, orderProcessor *signal.OrderProcessor) common.MessageHandler {
	return func(msg []byte) error {
		if err := handlePrivateMessage(mongoClient, msg); err != nil {
			log.Printf("Failed to handle private message: %v", err)
			return err
		}

		if orderProcessor != nil {
			if err := orderProcessor.HandleOrderResponse(msg); err != nil {
				if err.Error() != "order ID not found in message" {
					log.Printf("Failed to handle order response: %v", err)
				}
			}

			if err := orderProcessor.HandleErrorResponse(msg); err != nil {
				log.Printf("Error in order response: %v", err)
			}
		}

		return nil
	}
}

// handlePrivateMessage processes private WebSocket messages without creating a handler object
func handlePrivateMessage(mongoClient *mongodb.Client, message []byte) error {
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
		return handleOrders(mongoClient, data)
	case "positions":
		return handlePositions(mongoClient, data)
	default:
		log.Printf("Unknown private channel: %s", channel)
	}

	return nil
}

// handleOrders processes order channel data
func handleOrders(mongoClient *mongodb.Client, data []interface{}) error {
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

// handlePositions processes position channel data
func handlePositions(mongoClient *mongodb.Client, data []interface{}) error {
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

		if err := mongoClient.InsertPosition(pos); err != nil {
			log.Printf("Failed to insert position: %v", err)
		}
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
