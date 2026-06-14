// Package openfoodfacts is the library behind the openfoodfacts command line:
// the HTTP client, request shaping, and the typed data models for the Open
// Food Facts API (world.openfoodfacts.org).
//
// No authentication is required. The client sets a descriptive User-Agent and
// paces requests at 300ms minimum to be polite to the free community service.
package openfoodfacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "world.openfoodfacts.org"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://world.openfoodfacts.org",
		UserAgent: "openfoodfacts-cli/0.1.0 (github.com/tamnd/openfoodfacts-cli)",
		Rate:      300 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to Open Food Facts over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// wire types for JSON decode only

type productResponse struct {
	Status  int         `json:"status"` // 1=found, 0=not found
	Code    string      `json:"code"`
	Product wireProduct `json:"product"`
}

type searchResponse struct {
	Count    int           `json:"count"`
	Products []wireProduct `json:"products"`
}

type wireProduct struct {
	Code            string         `json:"code"`
	ProductName     string         `json:"product_name"`
	Brands          string         `json:"brands"`
	Categories      string         `json:"categories"`
	ImageURL        string         `json:"image_url"`
	NutriscoreGrade string         `json:"nutriscore_grade"`
	NovaGroup       int            `json:"nova_group"`
	Nutriments      wireNutriments `json:"nutriments"`
}

type wireNutriments struct {
	EnergyKcal100g    float64 `json:"energy-kcal_100g"`
	Fat100g           float64 `json:"fat_100g"`
	Carbohydrates100g float64 `json:"carbohydrates_100g"`
	Proteins100g      float64 `json:"proteins_100g"`
	Salt100g          float64 `json:"salt_100g"`
	Sugars100g        float64 `json:"sugars_100g"`
}

// GetProduct fetches the product with the given barcode.
// Returns an error if the product is not found (status == 0).
func (c *Client) GetProduct(ctx context.Context, barcode string) (*Product, error) {
	u := fmt.Sprintf(
		"%s/api/v2/product/%s.json",
		c.cfg.BaseURL, neturl.PathEscape(barcode),
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp productResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode product: %w", err)
	}
	if resp.Status == 0 {
		return nil, fmt.Errorf("product not found: %s", barcode)
	}
	p := toProduct(resp.Product)
	return &p, nil
}

// Search fetches products matching the given query string.
// limit defaults to 10 if <= 0.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Product, error) {
	n := limit
	if n <= 0 {
		n = 10
	}
	fields := "product_name,brands,categories,code,image_url,nutriscore_grade,nova_group,nutriments"
	u := fmt.Sprintf(
		"%s/cgi/search.pl?search_terms=%s&action=process&json=1&page_size=%d&fields=%s",
		c.cfg.BaseURL, neturl.QueryEscape(query), n, fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	products := make([]Product, 0, len(resp.Products))
	for _, r := range resp.Products {
		products = append(products, toProduct(r))
	}
	return products, nil
}

func toProduct(r wireProduct) Product {
	n := r.Nutriments
	return Product{
		Barcode:    r.Code,
		Name:       r.ProductName,
		Brands:     r.Brands,
		Categories: r.Categories,
		ImageURL:   r.ImageURL,
		NutriScore: r.NutriscoreGrade,
		NovaGroup:  r.NovaGroup,
		Nutriments: Nutriments{
			EnergyKcal100g:    n.EnergyKcal100g,
			Fat100g:           n.Fat100g,
			Carbohydrates100g: n.Carbohydrates100g,
			Proteins100g:      n.Proteins100g,
			Salt100g:          n.Salt100g,
			Sugars100g:        n.Sugars100g,
		},
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
