package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/mongodb"
)

func NewRedisListMessageHandler(mongoClient *mongodb.Client) common.MessageHandler {
	return func(msg []byte) error {
		var signal common.Signal
		if err := json.Unmarshal(msg, &signal); err != nil {
			return fmt.Errorf("failed to unmarshal signal: %w", err)
		}

		if err := validateSignal(&signal); err != nil {
			return fmt.Errorf("signal validation failed: %w", err)
		}

		tradingSignal := &mongodb.TradingSignal{
			ID:               fmt.Sprintf("signal_%s", signal.SignalID),
			SignalID:         signal.SignalID,
			InstID:           signal.InstID,
			Side:             signal.Side,
			OrdType:          signal.OrdType,
			PosSide:          signal.PosSide,
			Sz:               signal.Sz,
			Px:               signal.Px,
			ReduceOnly:       signal.ReduceOnly,
			Status:           "pending",
			SignalTimestamp:  signal.Timestamp,
			ProcessTimestamp: time.Now().UnixMilli(),
			CreatedAt:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			UpdatedAt:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		}

		if err := mongoClient.InsertTradingSignal(tradingSignal); err != nil {
			return fmt.Errorf("failed to insert trading signal: %w", err)
		}

		log.Printf("Signal recorded: %s (inst=%s, side=%s, type=%s)",
			signal.SignalID, signal.InstID, signal.Side, signal.OrdType)
		return nil
	}
}
