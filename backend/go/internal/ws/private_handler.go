package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/mongodb"
)

// PrivateHandler handles messages from OKEx private WebSocket channels
type PrivateHandler struct {
	mongoClient *mongodb.Client
}

// NewPrivateHandler creates a new private channel handler
func NewPrivateHandler(mongoClient *mongodb.Client) *PrivateHandler {
	return &PrivateHandler{
		mongoClient: mongoClient,
	}
}

// HandleMessage processes incoming WebSocket messages
func (h *PrivateHandler) HandleMessage(message []byte) error {
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
		return h.handleOrders(data)
	case "positions":
		return h.handlePositions(data)
	case "account-greeks":
		return h.handleGreeks(data)
	default:
		log.Printf("Unknown private channel: %s", channel)
	}

	return nil
}

// handleOrders processes order channel data
func (h *PrivateHandler) handleOrders(data []interface{}) error {
	for _, item := range data {
		orderMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		order, err := h.parseOrder(orderMap)
		if err != nil {
			log.Printf("Failed to parse order: %v", err)
			continue
		}

		if err := h.mongoClient.InsertOrder(order); err != nil {
			log.Printf("Failed to insert order: %v", err)
		}
	}
	return nil
}

// parseOrder parses order data from WebSocket message
func (h *PrivateHandler) parseOrder(data map[string]interface{}) (*mongodb.Order, error) {
	order := &mongodb.Order{
		Timestamp: time.Now().UnixMilli(),
	}

	if instID, ok := data["instId"].(string); ok {
		order.InstID = instID
	}

	if ordID, ok := data["ordId"].(string); ok {
		order.OrdID = ordID
	} else if ordIDFloat, ok := data["ordId"].(float64); ok {
		order.OrdID = strconv.FormatFloat(ordIDFloat, 'f', -1, 64)
	}

	if clOrdID, ok := data["clOrdId"].(string); ok {
		order.ClOrdID = clOrdID
	}

	if tag, ok := data["tag"].(string); ok {
		order.Tag = tag
	}

	if side, ok := data["side"].(string); ok {
		order.Side = side
	}

	if ordType, ok := data["ordType"].(string); ok {
		order.OrdType = ordType
	}

	if posSide, ok := data["posSide"].(string); ok {
		order.PosSide = posSide
	}

	if state, ok := data["state"].(string); ok {
		order.State = state
	}

	if sz, ok := data["sz"].(string); ok {
		order.Sz = sz
	}

	if px, ok := data["px"].(string); ok {
		order.Px = px
	}

	if lever, ok := data["lever"].(string); ok {
		order.Lever = lever
	}

	if tm, ok := data["tm"].(string); ok {
		order.Tm = tm
	}

	if cTime, ok := data["cTime"].(string); ok {
		order.CTime = cTime
	}

	if uTime, ok := data["uTime"].(string); ok {
		order.UTime = uTime
	}

	if reqID, ok := data["reqId"].(string); ok {
		order.ReqID = reqID
	} else if reqIDFloat, ok := data["reqId"].(float64); ok {
		order.ReqID = strconv.FormatFloat(reqIDFloat, 'f', -1, 64)
	}

	if fee, ok := data["fee"].(string); ok {
		order.Fee = fee
	}

	if fillSz, ok := data["accFillSz"].(string); ok {
		order.FillSz = fillSz
	}

	if fillPx, ok := data["avgPx"].(string); ok {
		order.FillPx = fillPx
	}

	if fillTime, ok := data["fillTime"].(string); ok {
		order.FillTime = fillTime
	}

	if fillNotionalUSD, ok := data["fillNotionalUsd"].(string); ok {
		order.FillNotionalUSD = fillNotionalUSD
	}

	if pnl, ok := data["pnl"].(string); ok {
		order.Pnl = pnl
	}

	if pnlRatio, ok := data["pnlRatio"].(string); ok {
		order.PnlRatio = pnlRatio
	}

	if category, ok := data["category"].(string); ok {
		order.Category = category
	}

	order.ID = order.OrdID

	return order, nil
}

// handlePositions processes position channel data
func (h *PrivateHandler) handlePositions(data []interface{}) error {
	for _, item := range data {
		posMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		position, err := h.parsePosition(posMap)
		if err != nil {
			log.Printf("Failed to parse position: %v", err)
			continue
		}

		if err := h.mongoClient.InsertPosition(position); err != nil {
			log.Printf("Failed to insert position: %v", err)
		}
	}
	return nil
}

// parsePosition parses position data from WebSocket message
func (h *PrivateHandler) parsePosition(data map[string]interface{}) (*mongodb.Position, error) {
	position := &mongodb.Position{
		Timestamp: time.Now().UnixMilli(),
	}

	if instID, ok := data["instId"].(string); ok {
		position.InstID = instID
	}

	if mgnMode, ok := data["mgnMode"].(string); ok {
		position.MgnMode = mgnMode
	}

	if posID, ok := data["posId"].(string); ok {
		position.PosID = posID
	} else if posIDFloat, ok := data["posId"].(float64); ok {
		position.PosID = strconv.FormatFloat(posIDFloat, 'f', -1, 64)
	}

	if posSide, ok := data["posSide"].(string); ok {
		position.PosSide = posSide
	}

	if pos, ok := data["pos"].(string); ok {
		position.Pos = pos
	}

	if baseBal, ok := data["baseBal"].(string); ok {
		position.BaseBal = baseBal
	}

	if quoteBal, ok := data["quoteBal"].(string); ok {
		position.QuoteBal = quoteBal
	}

	if posCcy, ok := data["posCcy"].(string); ok {
		position.PosCcy = posCcy
	}

	if pnlRatio, ok := data["pnlRatio"].(string); ok {
		position.PnlRatio = pnlRatio
	}

	if upl, ok := data["upl"].(string); ok {
		position.Upl = upl
	}

	if uplRatio, ok := data["uplRatio"].(string); ok {
		position.UplRatio = uplRatio
	}

	if lever, ok := data["lever"].(string); ok {
		position.Lever = lever
	}

	if liqPx, ok := data["liqPx"].(string); ok {
		position.LiqPx = liqPx
	}

	if markPx, ok := data["markPx"].(string); ok {
		position.MarkPx = markPx
	}

	if cTime, ok := data["cTime"].(string); ok {
		position.CTime = cTime
	}

	if uTime, ok := data["uTime"].(string); ok {
		position.UTime = uTime
	}

	if adl, ok := data["adl"].(string); ok {
		position.ADL = adl
	}

	if notionalUSD, ok := data["notionalUsd"].(string); ok {
		position.NotionalUSD = notionalUSD
	}

	if last, ok := data["last"].(string); ok {
		position.Last = last
	}

	position.ID = fmt.Sprintf("%s_%s", position.InstID, position.PosID)

	return position, nil
}

// handleGreeks processes Greeks channel data
func (h *PrivateHandler) handleGreeks(data []interface{}) error {
	for _, item := range data {
		log.Printf("Greeks data: %v", item)
	}
	return nil
}
