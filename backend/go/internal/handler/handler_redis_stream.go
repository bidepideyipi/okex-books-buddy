package handler

import (
	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/mongodb"
)

func NewRedisStreamMessageHandler(mongoClient *mongodb.Client) common.StreamSignalHandler {
	return func(msg *common.StreamSignal) error {
		return nil
	}
}
