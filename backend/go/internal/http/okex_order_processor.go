package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// OKExHTTPClient handles HTTP requests to OKEx API
type OKExHTTPClient struct {
	baseURL    string
	httpClient *http.Client
	config     ws.OKExConfig
	useProxy   bool
	proxyAddr  string
	timeOffset int64
}

// NewOKExHTTPClient creates a new OKEx HTTP client
func NewOKExHTTPClient(config ws.OKExConfig) *OKExHTTPClient {
	return &OKExHTTPClient{
		baseURL:    common.OKEX_API_BASE_URL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		config:     config,
		useProxy:   false,
		timeOffset: 0,
	}
}

// NewOKExHTTPClientWithProxy creates a new OKEx HTTP client with proxy support
func NewOKExHTTPClientWithProxy(config ws.OKExConfig, proxyAddr string) *OKExHTTPClient {

	log.Printf("useProxy: %v, proxyAddr: %s", true, proxyAddr)
	client := &OKExHTTPClient{
		baseURL:    common.OKEX_API_BASE_URL,
		config:     config,
		useProxy:   true,
		proxyAddr:  proxyAddr,
		timeOffset: 0,
	}

	if proxyAddr != "" {
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			log.Printf("Failed to parse proxy URL: %v", err)
			client.httpClient = &http.Client{Timeout: 30 * time.Second}
		} else {
			client.httpClient = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: 30 * time.Second,
			}
			log.Printf("Using HTTP proxy for API requests: %s", proxyAddr)
		}
	} else {
		client.httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return client
}

// SetTimeOffset sets the time offset from server
func (c *OKExHTTPClient) SetTimeOffset(offset int64) {
	c.timeOffset = offset
}

// generateSignature generates the signature for OKEx API authentication
func (c *OKExHTTPClient) generateSignature(timestamp, method, requestPath, body string) string {
	signStr := timestamp + method + requestPath + body
	h := hmac.New(sha256.New, []byte(c.config.SecretKey))
	h.Write([]byte(signStr))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// makeRequest makes an authenticated HTTP request to OKEx API
func (c *OKExHTTPClient) makeRequest(method, path string, params map[string]interface{}, body interface{}) ([]byte, error) {
	var requestBody string
	var err error

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = string(bodyBytes)
	}

	// timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	timestamp := time.Now().UTC().Add(time.Duration(c.timeOffset) * time.Millisecond).Format("2006-01-02T15:04:05.000Z")
	signature := c.generateSignature(timestamp, method, path, requestBody)
	log.Printf("[DEBUG] [%s] path is %s,reqBody is %s", method, path, requestBody)

	requestURL := c.baseURL + path
	if method == "GET" && len(params) > 0 {
		queryParams := url.Values{}
		for k, v := range params {
			queryParams.Set(k, fmt.Sprintf("%v", v))
		}
		requestURL += "?" + queryParams.Encode()
	}

	var req *http.Request
	if method == "GET" || method == "DELETE" {
		req, err = http.NewRequest(method, requestURL, nil)
	} else {
		req, err = http.NewRequest(method, requestURL, bytes.NewBufferString(requestBody))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("OK-ACCESS-KEY", c.config.APIKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", c.config.Passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	log.Printf("respBody=%s", respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// PlaceOrder places a batch order via OKEx API
func (c *OKExHTTPClient) PlaceOrder(orderReq *PlaceOrderRequest) (*OrderResponse, error) {

	respBody, err := c.makeRequest("POST", common.PLACE_ORDER_PATH, nil, orderReq)
	if err != nil {
		return nil, err
	}

	var response OrderResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Code != "0" {
		return nil, fmt.Errorf("order placement failed: code=%s, msg=%s", response.Code, response.Msg)
	}

	return &response, nil
}

// PlaceAlgoOrder places a batch order via OKEx API
func (c *OKExHTTPClient) PlaceAlgoOrder(orderReq *PlaceAlgoOrderRequest) (*OrderResponse, error) {

	respBody, err := c.makeRequest("POST", common.PLACE_ORDER_ALGO_PATH, nil, orderReq)
	if err != nil {
		return nil, err
	}

	var response OrderResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Code != "0" {
		return nil, fmt.Errorf("order placement failed: code=%s, msg=%s", response.Code, response.Msg)
	}

	return &response, nil
}

func (c *OKExHTTPClient) OrdersPending(orderReq *OrdersPendingRequest) (*OrdersPendingResponse, error) {
	respBody, err := c.makeRequest("GET", common.ORDERS_PENDING_PATH, nil, nil)
	if err != nil {
		return nil, err
	}

	var response OrdersPendingResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Code != "0" {
		return nil, fmt.Errorf("order placement failed: code=%s, msg=%s", response.Code, response.Msg)
	}

	return &response, nil
}

func (c *OKExHTTPClient) CancelOrder(orderReq *CancelOrderRequest) (*OrderResponse, error) {
	respBody, err := c.makeRequest("POST", common.CANCEL_ORDER_PATH, nil, orderReq)
	if err != nil {
		return nil, err
	}

	var response OrderResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Code != "0" {
		return nil, fmt.Errorf("order placement failed: code=%s, msg=%s", response.Code, response.Msg)
	}

	return &response, nil
}

func (c *OKExHTTPClient) AmendOrder(orderReq *AmendOrderRequest) (*OrderResponse, error) {
	respBody, err := c.makeRequest("POST", common.AMEND_ORDER_PATH, nil, orderReq)
	if err != nil {
		return nil, err
	}

	var response OrderResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Code != "0" {
		return nil, fmt.Errorf("order placement failed: code=%s, msg=%s", response.Code, response.Msg)
	}

	return &response, nil
}

// SyncServerTime syncs server time and returns the offset
func (c *OKExHTTPClient) SyncServerTime() (int64, error) {
	requestURL := c.baseURL + "/api/v5/public/time"

	var client *http.Client
	if c.useProxy && c.proxyAddr != "" {
		proxyURL, err := url.Parse("http://" + c.proxyAddr)
		if err == nil {
			client = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: 10 * time.Second,
			}
		}
	}

	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := client.Get(requestURL)
	if err != nil {
		return 0, fmt.Errorf("failed to get server time: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var timeResponse ServerTimeResponse
	if err := json.Unmarshal(body, &timeResponse); err != nil {
		return 0, fmt.Errorf("failed to unmarshal time response: %w", err)
	}

	if len(timeResponse.Data) == 0 {
		return 0, fmt.Errorf("empty server time response")
	}

	serverTimeStr := timeResponse.Data[0].Ts
	serverTime, err := strconv.ParseInt(serverTimeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse server time: %w", err)
	}

	localTime := time.Now().UnixMilli()
	offset := serverTime - localTime

	log.Printf("OKEx server time: %d ms, Local time: %d ms, Offset: %d ms", serverTime, localTime, offset)

	return offset, nil
}

func GenerateOrderID(head string) string {
	// 生成随机字母 a-z
	randomLetter := 'a' + rune(rand.Intn(26))

	// 获取当前时间戳（毫秒）
	timestamp := time.Now().UnixMilli()

	// 格式化：ao + 随机字母 + 时间戳
	return fmt.Sprintf("%s%d%c", head, timestamp, randomLetter)
}
