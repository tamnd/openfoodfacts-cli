package openfoodfacts_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/openfoodfacts-cli/openfoodfacts"
)

// --- fake JSON responses ---

const fakeProductJSON = `{
  "status": 1,
  "product": {
    "product_name": "Nutella",
    "brands": "Ferrero",
    "categories": "Spreads",
    "quantity": "400 g",
    "nutrition_grades": "e",
    "ecoscore_grade": "d",
    "nutriments": {
      "energy-kcal_100g": 539,
      "fat_100g": 30.9,
      "saturated-fat_100g": 10.6,
      "carbohydrates_100g": 57.5,
      "sugars_100g": 56.3,
      "proteins_100g": 6.3,
      "salt_100g": 0.107,
      "fiber_100g": 0
    }
  }
}`

const fakeProductNotFoundJSON = `{"status": 0, "product": {}}`

const fakeSearchJSON = `{
  "count": 15234,
  "page": 1,
  "page_size": 3,
  "products": [
    {
      "code": "3046920022651",
      "product_name": "Extra fine dark chocolate",
      "brands": "Lindt",
      "nutrition_grades": "d",
      "ecoscore_grade": "c"
    },
    {
      "code": "3046920028950",
      "product_name": "Excellence 70% cacao",
      "brands": "Lindt",
      "nutrition_grades": "c",
      "ecoscore_grade": "b"
    }
  ]
}`

const fakeCategoryJSON = `{
  "count": 12345,
  "products": [
    {
      "code": "3017624010701",
      "product_name": "Nutella",
      "brands": "Ferrero",
      "nutrition_grades": "e",
      "ecoscore_grade": "d"
    },
    {
      "code": "3046920022651",
      "product_name": "Extra fine dark chocolate",
      "brands": "Lindt",
      "nutrition_grades": "d",
      "ecoscore_grade": "c"
    }
  ]
}`

func newTestClient(ts *httptest.Server) *openfoodfacts.Client {
	cfg := openfoodfacts.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return openfoodfacts.NewClient(cfg)
}

// --- product tests ---

func TestGetProductParsesFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.GetProduct(context.Background(), "3017624010701")
	if err != nil {
		t.Fatal(err)
	}
	if p.Barcode != "3017624010701" {
		t.Errorf("Barcode = %q, want 3017624010701", p.Barcode)
	}
	if p.Name != "Nutella" {
		t.Errorf("Name = %q, want Nutella", p.Name)
	}
	if p.Brands != "Ferrero" {
		t.Errorf("Brands = %q, want Ferrero", p.Brands)
	}
	if p.Quantity != "400 g" {
		t.Errorf("Quantity = %q, want 400 g", p.Quantity)
	}
	if p.NutritionGrade != "e" {
		t.Errorf("NutritionGrade = %q, want e", p.NutritionGrade)
	}
	if p.EcoScore != "d" {
		t.Errorf("EcoScore = %q, want d", p.EcoScore)
	}
}

func TestGetProductNutriments(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.GetProduct(context.Background(), "3017624010701")
	if err != nil {
		t.Fatal(err)
	}
	if p.EnergyKcal != "539.0" {
		t.Errorf("EnergyKcal = %q, want 539.0", p.EnergyKcal)
	}
	if p.Fat != "30.9" {
		t.Errorf("Fat = %q, want 30.9", p.Fat)
	}
	if p.Sugars != "56.3" {
		t.Errorf("Sugars = %q, want 56.3", p.Sugars)
	}
	if p.Proteins != "6.3" {
		t.Errorf("Proteins = %q, want 6.3", p.Proteins)
	}
	if p.Salt != "0.1" {
		t.Errorf("Salt = %q, want 0.1", p.Salt)
	}
}

func TestGetProductNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductNotFoundJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetProduct(context.Background(), "0000000000000")
	if err == nil {
		t.Error("expected error for not-found product, got nil")
	}
	if !strings.Contains(err.Error(), "product not found") {
		t.Errorf("error = %q, want 'product not found'", err.Error())
	}
}

func TestGetProductSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetProduct(context.Background(), "3017624010701")
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestGetProductUsesV2Endpoint(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.GetProduct(context.Background(), "3017624010701")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "/api/v2/product/") {
		t.Errorf("expected /api/v2/product/ in path, got %q", gotPath)
	}
}

// --- search tests ---

func TestSearchParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.Search(context.Background(), "chocolate", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	r0 := results[0]
	if r0.Name != "Extra fine dark chocolate" {
		t.Errorf("results[0].Name = %q, want Extra fine dark chocolate", r0.Name)
	}
	if r0.Barcode != "3046920022651" {
		t.Errorf("results[0].Barcode = %q, want 3046920022651", r0.Barcode)
	}
	if r0.NutritionGrade != "d" {
		t.Errorf("results[0].NutritionGrade = %q, want d", r0.NutritionGrade)
	}
	if r0.EcoScore != "c" {
		t.Errorf("results[0].EcoScore = %q, want c", r0.EcoScore)
	}
}

func TestSearchUsesSearchPl(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Search(context.Background(), "chocolate", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "search.pl") {
		t.Errorf("expected search.pl in path, got %q", gotPath)
	}
}

func TestSearchDefaultsLimit(t *testing.T) {
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Search(context.Background(), "milk", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "page_size=20") {
		t.Errorf("expected page_size=20 in query, got %q", gotQuery)
	}
}

func TestSearchRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	cfg := openfoodfacts.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := openfoodfacts.NewClient(cfg)

	_, err := c.Search(context.Background(), "chocolate", 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

// --- category tests ---

func TestCategoryParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeCategoryJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	results, err := c.Category(context.Background(), "chocolates", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	r0 := results[0]
	if r0.Name != "Nutella" {
		t.Errorf("results[0].Name = %q, want Nutella", r0.Name)
	}
	if r0.Barcode != "3017624010701" {
		t.Errorf("results[0].Barcode = %q, want 3017624010701", r0.Barcode)
	}
}

func TestCategoryUsesCorrectPath(t *testing.T) {
	var gotPath, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, fakeCategoryJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Category(context.Background(), "chocolates", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "search.pl") {
		t.Errorf("expected search.pl in path, got %q", gotPath)
	}
	if !strings.Contains(gotQuery, "chocolates") {
		t.Errorf("expected category name in query, got %q", gotQuery)
	}
}
