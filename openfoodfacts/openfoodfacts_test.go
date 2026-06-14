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

const fakeProductJSON = `{
  "status": 1,
  "code": "737628064502",
  "product": {
    "code": "737628064502",
    "product_name": "Organic Whole Milk",
    "brands": "Horizon Organic",
    "categories": "Milks, Cow milks",
    "image_url": "https://images.openfoodfacts.org/images/products/073/762/806/4502/front.jpg",
    "nutriscore_grade": "b",
    "nova_group": 1,
    "nutriments": {
      "energy-kcal_100g": 61,
      "fat_100g": 3.25,
      "carbohydrates_100g": 4.8,
      "proteins_100g": 3.15,
      "salt_100g": 0.1,
      "sugars_100g": 4.8
    }
  }
}`

const fakeProductNotFoundJSON = `{"status": 0, "code": "0000000000000", "product": {}}`

const fakeSearchJSON = `{
  "count": 124867,
  "products": [
    {
      "code": "3046920022651",
      "product_name": "Extra fine dark chocolate",
      "brands": "Lindt & Sprungli",
      "categories": "Chocolates",
      "image_url": "https://images.openfoodfacts.org/images/products/1.jpg",
      "nutriscore_grade": "d",
      "nova_group": 4,
      "nutriments": {
        "energy-kcal_100g": 546,
        "fat_100g": 36,
        "carbohydrates_100g": 46,
        "proteins_100g": 5,
        "salt_100g": 0.01,
        "sugars_100g": 44
      }
    },
    {
      "code": "3046920028950",
      "product_name": "Excellence 70% cacao",
      "brands": "Lindt",
      "categories": "Dark chocolates",
      "image_url": "",
      "nutriscore_grade": "c",
      "nova_group": 3,
      "nutriments": {
        "energy-kcal_100g": 580,
        "fat_100g": 42,
        "carbohydrates_100g": 35,
        "proteins_100g": 8,
        "salt_100g": 0.005,
        "sugars_100g": 27
      }
    }
  ],
  "skip": 0,
  "page_size": 5,
  "page": 1
}`

func newTestClient(ts *httptest.Server) *openfoodfacts.Client {
	cfg := openfoodfacts.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return openfoodfacts.NewClient(cfg)
}

func TestGetProductParsesFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.GetProduct(context.Background(), "737628064502")
	if err != nil {
		t.Fatal(err)
	}
	if p.Barcode != "737628064502" {
		t.Errorf("Barcode = %q, want 737628064502", p.Barcode)
	}
	if p.Name != "Organic Whole Milk" {
		t.Errorf("Name = %q, want Organic Whole Milk", p.Name)
	}
	if p.Brands != "Horizon Organic" {
		t.Errorf("Brands = %q, want Horizon Organic", p.Brands)
	}
	if p.NutriScore != "b" {
		t.Errorf("NutriScore = %q, want b", p.NutriScore)
	}
	if p.NovaGroup != 1 {
		t.Errorf("NovaGroup = %d, want 1", p.NovaGroup)
	}
	if p.ImageURL == "" {
		t.Error("ImageURL is empty")
	}
}

func TestGetProductNutriments(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.GetProduct(context.Background(), "737628064502")
	if err != nil {
		t.Fatal(err)
	}
	n := p.Nutriments
	if n.EnergyKcal100g != 61 {
		t.Errorf("EnergyKcal100g = %v, want 61", n.EnergyKcal100g)
	}
	if n.Fat100g != 3.25 {
		t.Errorf("Fat100g = %v, want 3.25", n.Fat100g)
	}
	if n.Carbohydrates100g != 4.8 {
		t.Errorf("Carbohydrates100g = %v, want 4.8", n.Carbohydrates100g)
	}
	if n.Proteins100g != 3.15 {
		t.Errorf("Proteins100g = %v, want 3.15", n.Proteins100g)
	}
	if n.Salt100g != 0.1 {
		t.Errorf("Salt100g = %v, want 0.1", n.Salt100g)
	}
	if n.Sugars100g != 4.8 {
		t.Errorf("Sugars100g = %v, want 4.8", n.Sugars100g)
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
	_, err := c.GetProduct(context.Background(), "737628064502")
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
	_, err := c.GetProduct(context.Background(), "737628064502")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "/api/v2/product/") {
		t.Errorf("expected /api/v2/product/ endpoint, got path %q", gotPath)
	}
}

func TestSearchParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	products, err := c.Search(context.Background(), "chocolate", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}

	p0 := products[0]
	if p0.Name != "Extra fine dark chocolate" {
		t.Errorf("products[0].Name = %q, want Extra fine dark chocolate", p0.Name)
	}
	if p0.Brands == "" {
		t.Error("products[0].Brands is empty")
	}
	if p0.NutriScore != "d" {
		t.Errorf("products[0].NutriScore = %q, want d", p0.NutriScore)
	}
	if p0.NovaGroup != 4 {
		t.Errorf("products[0].NovaGroup = %d, want 4", p0.NovaGroup)
	}

	p1 := products[1]
	if p1.Name != "Excellence 70% cacao" {
		t.Errorf("products[1].Name = %q, want Excellence 70%% cacao", p1.Name)
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
	_, err := c.Search(context.Background(), "chocolate", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "search.pl") {
		t.Errorf("expected search.pl endpoint, got path %q", gotPath)
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
	_, err := c.Search(context.Background(), "milk", 0) // limit=0 → default 10
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "page_size=10") {
		t.Errorf("expected page_size=10 in query, got %q", gotQuery)
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

	_, err := c.Search(context.Background(), "chocolate", 5)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
