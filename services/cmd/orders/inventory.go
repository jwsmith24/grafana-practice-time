package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jwsmith24/grafana-practice-time/services/internal/auth"
	"github.com/jwsmith24/grafana-practice-time/services/internal/requestid"
)

// DefaultInventoryTimeout bounds each call to inventory.
const DefaultInventoryTimeout = 2 * time.Second

var (
	errNotFound    = errors.New("item not found")
	errUpstream5xx = errors.New("inventory server error")
)

type inventoryClient struct {
	baseURL string
	token   string
	timeout time.Duration
	http    *http.Client
}

func newInventoryClient(baseURL, token string, timeout time.Duration) *inventoryClient {
	return &inventoryClient{baseURL: baseURL, token: token, timeout: timeout, http: &http.Client{}}
}

// stock returns the stock level for itemID. Errors wrap errNotFound,
// errUpstream5xx or context.DeadlineExceeded where they apply.
func (c *inventoryClient) stock(ctx context.Context, reqID, itemID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/inventory/"+url.PathEscape(itemID), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set(requestid.Header, reqID)
	req.Header.Set(auth.Header, c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, context.DeadlineExceeded
		}
		return 0, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return 0, errNotFound
	case resp.StatusCode >= 500:
		return 0, fmt.Errorf("%w: status %d", errUpstream5xx, resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return 0, fmt.Errorf("inventory returned status %d", resp.StatusCode)
	}
	var body struct {
		Stock int `json:"stock"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("decode inventory response: %w", err)
	}
	return body.Stock, nil
}
