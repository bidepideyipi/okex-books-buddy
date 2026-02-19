package common

// MessageHandler processes incoming messages
type MessageHandler func(msg []byte) error

// WSClientInterface defines the common interface for WebSocket clients
type WSClientInterface interface {
	Subscribe(params interface{}) error
	Unsubscribe(params interface{}) error
	GetSubscribed() []string
}
