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
		UserAgent: "openfoodfacts-cli/0.1 (tamnd87@gmail.com)",
		Rate:      300 * time.Millisecond,
		Timeout:   20 * time.Second,
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

// wire types — used only for JSON decode

type productResponse struct {
	Status  int         `json:"status"` // 1=found, 0=not found
	Product wireProduct `json:"product"`
}

type searchResponse struct {
	Count    int           `json:"count"`
	Products []wireProduct `json:"products"`
}

type wireProduct struct {
	Code            string                     `json:"code"`
	ProductName     string                     `json:"product_name"`
	Brands          string                     `json:"brands"`
	Categories      string                     `json:"categories"`
	Quantity        string                     `json:"quantity"`
	NutritionGrades string                     `json:"nutrition_grades"`
	EcoscoreGrade   string                     `json:"ecoscore_grade"`
	Nutriments      map[string]json.RawMessage `json:"nutriments"`
}

func nutriFloat(m map[string]json.RawMessage, key string) float64 {
	if m == nil {
		return 0
	}
	raw, ok := m[key]
	if !ok {
		return 0
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0
	}
	return v
}

func fmtFloat(v float64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f", v)
}

// GetProduct fetches the product with the given barcode.
// Returns an error if the product is not found (status == 0).
func (c *Client) GetProduct(ctx context.Context, barcode string) (*Product, error) {
	fields := "product_name,brands,categories,quantity,nutrition_grades,ecoscore_grade,nutriments"
	u := fmt.Sprintf(
		"%s/api/v2/product/%s.json?fields=%s",
		c.cfg.BaseURL, neturl.PathEscape(barcode), fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp productResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode product: %w", err)
	}
	if resp.Status != 1 {
		return nil, fmt.Errorf("product not found: %s", barcode)
	}
	p := toProduct(barcode, resp.Product)
	return &p, nil
}

// Search fetches products matching the given query string.
func (c *Client) Search(ctx context.Context, query string, limit, page int) ([]SearchResult, error) {
	n := limit
	if n <= 0 {
		n = 20
	}
	pg := page
	if pg <= 0 {
		pg = 1
	}
	fields := "product_name,brands,nutrition_grades,ecoscore_grade,code"
	u := fmt.Sprintf(
		"%s/cgi/search.pl?search_terms=%s&search_simple=1&action=process&json=1&page_size=%d&page=%d&fields=%s",
		c.cfg.BaseURL, neturl.QueryEscape(query), n, pg, fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode search: %w", err)
	}
	results := make([]SearchResult, 0, len(resp.Products))
	for _, r := range resp.Products {
		results = append(results, toSearchResult(r))
	}
	return results, nil
}

// Category lists products in the given Open Food Facts taxonomy category.
// It uses the search CGI with a category tag filter, which is more reliable
// than the facet browsing endpoint that is sometimes unavailable.
func (c *Client) Category(ctx context.Context, category string, limit int) ([]SearchResult, error) {
	n := limit
	if n <= 0 {
		n = 20
	}
	fields := "product_name,brands,nutrition_grades,ecoscore_grade,code"
	u := fmt.Sprintf(
		"%s/cgi/search.pl?tagtype_0=categories&tag_contains_0=contains&tag_0=%s&action=process&json=1&page_size=%d&fields=%s",
		c.cfg.BaseURL, neturl.QueryEscape(category), n, fields,
	)
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode category: %w", err)
	}
	results := make([]SearchResult, 0, len(resp.Products))
	for _, r := range resp.Products {
		results = append(results, toSearchResult(r))
	}
	return results, nil
}

func toProduct(barcode string, r wireProduct) Product {
	n := r.Nutriments
	return Product{
		Barcode:        barcode,
		Name:           r.ProductName,
		Brands:         r.Brands,
		Categories:     r.Categories,
		Quantity:       r.Quantity,
		NutritionGrade: r.NutritionGrades,
		EcoScore:       r.EcoscoreGrade,
		EnergyKcal:     fmtFloat(nutriFloat(n, "energy-kcal_100g")),
		Fat:            fmtFloat(nutriFloat(n, "fat_100g")),
		Sugars:         fmtFloat(nutriFloat(n, "sugars_100g")),
		Proteins:       fmtFloat(nutriFloat(n, "proteins_100g")),
		Salt:           fmtFloat(nutriFloat(n, "salt_100g")),
	}
}

func toSearchResult(r wireProduct) SearchResult {
	return SearchResult{
		Barcode:        r.Code,
		Name:           r.ProductName,
		Brands:         r.Brands,
		NutritionGrade: r.NutritionGrades,
		EcoScore:       r.EcoscoreGrade,
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
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		return 5 * time.Second
	}
	return d
}
