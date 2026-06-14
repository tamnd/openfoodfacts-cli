// Package openfoodfacts is the library behind the openfoodfacts command line:
// the HTTP client, request shaping, and the typed data models for the Open
// Food Facts API (world.openfoodfacts.org).
//
// No authentication is required. The client sets a descriptive User-Agent and
// paces requests at 200ms minimum to be polite to the free community service.
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
		Rate:      200 * time.Millisecond,
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

// Product fetches the product with the given barcode.
// Returns an error if the product is not found (status == 0).
func (c *Client) Product(ctx context.Context, barcode string) (Product, error) {
	fields := "product_name,brands,categories,ingredients_text,nutriments,ecoscore_grade,nutriscore_grade,code"
	u := fmt.Sprintf(
		"%s/api/v2/product/%s.json?fields=%s",
		c.cfg.BaseURL, neturl.PathEscape(barcode), fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return Product{}, err
	}
	var resp wireProductResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return Product{}, fmt.Errorf("decode product: %w", err)
	}
	if resp.Status == 0 {
		return Product{}, fmt.Errorf("product not found: %s", barcode)
	}
	return toProduct(resp.Product), nil
}

// Search fetches products matching the given query string.
// page is 1-based; pass 0 for page 1. limit defaults to 10 if <= 0.
func (c *Client) Search(ctx context.Context, query string, limit, page int) ([]Product, error) {
	n := limit
	if n <= 0 {
		n = 10
	}
	p := page
	if p <= 0 {
		p = 1
	}
	fields := "product_name,brands,categories,nutriscore_grade,ecoscore_grade,code"
	u := fmt.Sprintf(
		"%s/cgi/search.pl?search_terms=%s&search_simple=1&action=process&json=1&page_size=%d&page=%d&fields=%s",
		c.cfg.BaseURL, neturl.QueryEscape(query), n, p, fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp wireSearchResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	products := make([]Product, 0, len(resp.Products))
	for _, r := range resp.Products {
		products = append(products, toProduct(r))
	}
	return products, nil
}

// Category fetches products in the given category tag (e.g. "en:beverages").
// limit defaults to 10 if <= 0.
func (c *Client) Category(ctx context.Context, category string, limit int) ([]Product, error) {
	n := limit
	if n <= 0 {
		n = 10
	}
	fields := "product_name,brands,categories,nutriscore_grade,code"
	u := fmt.Sprintf(
		"%s/api/v2/search?categories_tags=%s&page_size=%d&fields=%s",
		c.cfg.BaseURL, neturl.QueryEscape(category), n, fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp wireSearchResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode category search: %w", err)
	}
	products := make([]Product, 0, len(resp.Products))
	for _, r := range resp.Products {
		products = append(products, toProduct(r))
	}
	return products, nil
}

// Nutrients fetches nutritional information for the product with the given barcode.
func (c *Client) Nutrients(ctx context.Context, barcode string) (Product, error) {
	fields := "product_name,brands,nutriments,nutriscore_grade,ecoscore_grade,code"
	u := fmt.Sprintf(
		"%s/api/v2/product/%s.json?fields=%s",
		c.cfg.BaseURL, neturl.PathEscape(barcode), fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return Product{}, err
	}
	var resp wireProductResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return Product{}, fmt.Errorf("decode product: %w", err)
	}
	if resp.Status == 0 {
		return Product{}, fmt.Errorf("product not found: %s", barcode)
	}
	return toProduct(resp.Product), nil
}

func toProduct(r wireProduct) Product {
	n := r.Nutriments
	nutrients := &Nutrients{
		EnergyKcal: n.EnergyKcal,
		Fat:        n.Fat,
		SatFat:     n.SatFat,
		Carbs:      n.Carbs,
		Sugars:     n.Sugars,
		Protein:    n.Proteins,
		Salt:       n.Salt,
		Fiber:      n.Fiber,
	}
	return Product{
		Barcode:     r.Code,
		Name:        r.ProductName,
		Brands:      r.Brands,
		Categories:  r.Categories,
		Ingredients: r.IngredientsText,
		NutriScore:  r.NutriscoreGrade,
		EcoScore:    r.EcoscoreGrade,
		Nutrients:   nutrients,
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
