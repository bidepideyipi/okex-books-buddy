package ws

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ServerTimeResponse represents OKEx server time response
type ServerTimeResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		Timestamp string `json:"ts"`
	} `json:"data"`
}

// GetServerTime fetches current server time from OKEx with optional HTTP proxy
func GetServerTime(proxyAddr string) (int64, error) {
	transport := &http.Transport{}

	if proxyAddr != "" {
		proxyURL, err := url.Parse("http://" + proxyAddr)
		if err != nil {
			return 0, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		log.Printf("Using HTTP proxy for time sync: %s", proxyAddr)
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	resp, err := client.Get("https://www.okx.com/api/v5/public/time")
	if err != nil {
		return 0, fmt.Errorf("failed to fetch server time: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var serverTimeResp ServerTimeResponse
	if err := json.Unmarshal(body, &serverTimeResp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if serverTimeResp.Code != "0" {
		return 0, fmt.Errorf("server returned error: %s - %s", serverTimeResp.Code, serverTimeResp.Msg)
	}

	if len(serverTimeResp.Data) == 0 {
		return 0, fmt.Errorf("no server time data in response")
	}

	serverTimeMs, err := strconv.ParseInt(serverTimeResp.Data[0].Timestamp, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse server time: %w", err)
	}

	log.Printf("OKEx server time: %d ms", serverTimeMs)
	return serverTimeMs, nil
}

// SyncServerTime calculates time offset from server with optional HTTP proxy
func SyncServerTime(proxyAddr string) (int64, error) {
	serverTime, err := GetServerTime(proxyAddr)
	if err != nil {
		return 0, err
	}

	localTime := time.Now().UnixMilli()
	offset := serverTime - localTime

	log.Printf("Server time: %d ms, Local time: %d ms, Offset: %d ms", serverTime, localTime, offset)

	return offset, nil
}
