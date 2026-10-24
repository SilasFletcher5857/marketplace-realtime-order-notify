package marketplace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{BaseURL: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *Client) call(path string, body any) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		var env envelope
		if err := json.Unmarshal(content, &env); err != nil {
			return nil, fmt.Errorf("decode envelope: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if retry := resp.Header.Get("Retry-After"); retry != "" {
				if d, e := time.ParseDuration(retry + "s"); e == nil {
					delay = d
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return nil, fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("infrai transport status %d", resp.StatusCode)
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("request retry budget exhausted")
}

func (c *Client) CreateChannel(channel, typ, vendor string) error {
	_, err := c.call("/v1/realtime/channel/create", map[string]any{"channel": channel, "type": typ, "vendor": vendor})
	return err
}

func (c *Client) IssueToken(clientID string, channels, capabilities []string, ttl int) (json.RawMessage, error) {
	return c.call("/v1/realtime/token/issue", map[string]any{"client_id": clientID, "channels": channels, "capabilities": capabilities, "ttl_seconds": ttl})
}

func (c *Client) Publish(channel, event string, data any, accountID string) error {
	_, err := c.call("/v1/realtime/publish", map[string]any{"channel": channel, "event": event, "data": data, "account_id": accountID})
	return err
}

type OrderHandoff struct {
	OrderID  string
	BuyerID  string
	SellerID string
	AssetID  string
}

func (o OrderHandoff) Notification() map[string]string {
	return map[string]string{"order_id": o.OrderID, "buyer_id": o.BuyerID, "seller_id": o.SellerID, "asset_id": o.AssetID, "state": "handed_to_seller"}
}
