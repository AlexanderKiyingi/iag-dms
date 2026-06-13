// Package financeclient calls iag-finance for distribution finance hub data.
package financeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	platformserviceauth "github.com/alvor-technologies/iag-platform-go/serviceauth"
)

// Client fetches finance summaries and AR items from iag-finance.
type Client struct {
	baseURL    string
	httpClient *http.Client
	sa         *platformserviceauth.Client
}

// Config wires an optional finance upstream.
type Config struct {
	BaseURL         string
	TokenURL        string
	ServiceClientID string
	ServiceSecret   string
}

// New returns a Client. When BaseURL is empty the client is disabled.
func New(cfg Config) *Client {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return &Client{}
	}
	var sa *platformserviceauth.Client
	if cfg.ServiceSecret != "" {
		sa = platformserviceauth.NewClient(platformserviceauth.Options{
			TokenURL:     cfg.TokenURL,
			ClientID:     cfg.ServiceClientID,
			ClientSecret: cfg.ServiceSecret,
			Audience:     "iag.finance",
		})
	}
	return &Client{
		baseURL:    base,
		httpClient: &http.Client{Timeout: 8 * time.Second},
		sa:         sa,
	}
}

func (c *Client) Enabled() bool { return c != nil && c.baseURL != "" }

type Summary struct {
	ARBalance   string `json:"arBalance"`
	Overdue     string `json:"overdue"`
	Collected   string `json:"collected"`
	OpenItems   int    `json:"openItems"`
	GeneratedAt string `json:"generatedAt"`
}

type Invoice struct {
	No       string  `json:"no"`
	Customer string  `json:"customer"`
	Total    float64 `json:"total"`
	Balance  float64 `json:"balance"`
	Status   string  `json:"status"`
	Due      string  `json:"due"`
}

func (c *Client) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	err := c.getJSON(ctx, "/v1/finance/summary", &out)
	return out, err
}

func (c *Client) GetInvoice(ctx context.Context, no string) (Invoice, error) {
	var out Invoice
	err := c.getJSON(ctx, "/v1/invoices/"+no, &out)
	return out, err
}

func (c *Client) ListInvoices(ctx context.Context, limit int) ([]Invoice, error) {
	if limit <= 0 {
		limit = 50
	}
	var payload struct {
		Items []Invoice `json:"items"`
	}
	err := c.getJSON(ctx, fmt.Sprintf("/v1/invoices?limit=%d", limit), &payload)
	return payload.Items, err
}

// CreateInvoiceRequest is the body iag-finance expects on POST /v1/invoices.
type CreateInvoiceRequest struct {
	Customer string  `json:"customer"`
	Total    float64 `json:"total"`
	Due      string  `json:"due,omitempty"` // YYYY-MM-DD
	Status   string  `json:"status,omitempty"`
}

// CreateInvoice records the invoice in the finance service of record and
// returns the canonical invoice (with the finance-assigned number).
func (c *Client) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (Invoice, error) {
	var out Invoice
	err := c.postJSON(ctx, "/v1/invoices", req, &out)
	return out, err
}

func (c *Client) getJSON(ctx context.Context, path string, dest any) error {
	return c.do(ctx, http.MethodGet, path, nil, dest)
}

func (c *Client) postJSON(ctx context.Context, path string, payload, dest any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, body, dest)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, dest any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.sa != nil {
		tok, err := c.sa.Token(ctx)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("finance %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(respBody, dest)
}
