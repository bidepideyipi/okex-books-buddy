package http

// HealthCheckResponse represents the health check response structure
type HealthCheckResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PublicWebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"public_websocket"`
		PrivateWebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"private_websocket"`
		BusinessWebSocket struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"business_websocket"`
		Redis struct {
			Status    string `json:"status"`
			Message   string `json:"message"`
			Timestamp int64  `json:"timestamp"`
		} `json:"redis"`
	} `json:"data"`
}
