package handler

import (
	"fmt"
	"strconv"

	"github.com/supermancell/okex-buddy/internal/common"
)

// validateSignal validates the trading signal
func validateSignal(signal *common.Signal) error {
	if signal.SignalID == "" {
		return fmt.Errorf("signal_id is required")
	}

	if signal.InstID == "" {
		return fmt.Errorf("inst_id is required")
	}

	if signal.Side != "buy" && signal.Side != "sell" {
		return fmt.Errorf("side must be 'buy' or 'sell'")
	}

	if signal.OrdType == "" {
		return fmt.Errorf("ord_type is required")
	}

	if signal.PosSide != "long" && signal.PosSide != "short" && signal.PosSide != "net" {
		return fmt.Errorf("pos_side must be 'long', 'short', or 'net'")
	}

	if signal.Sz == "" {
		return fmt.Errorf("sz (size) is required")
	}

	if sz, err := strconv.ParseFloat(signal.Sz, 64); err != nil || sz <= 0 {
		return fmt.Errorf("sz must be a positive number")
	}

	if signal.OrdType == "limit" && signal.Px == "" {
		return fmt.Errorf("px (price) is required for limit orders")
	}

	if signal.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}

	return nil
}
