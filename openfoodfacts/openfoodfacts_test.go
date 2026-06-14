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

const fakeSearchJSON = `{"count":31300,"page":1,"page_size":10,"products":[
  {"code":"3046920022651","product_name":"Extra fine dark chocolate","brands":"Lindt & Sprüngli","categories":"Chocolates","nutriscore_grade":"d","ecoscore_grade":"c"},
  {"code":"3046920028950","product_name":"Excellence 70% cacao noir intense","brands":"Lindt","categories":"Dark chocolates","nutriscore_grade":"c","ecoscore_grade":"b"}
]}`

const fakeProductJSON = `{"status":1,"status_verbose":"product found","product":{"code":"3017620422003","product_name":"Nutella","brands":"Nutella","categories":"Spreads","ingredients_text":"Sugar, palm oil, hazelnuts","nutriscore_grade":"e","ecoscore_grade":"d","nutriments":{"energy-kcal_100g":539,"fat_100g":30.9,"saturated-fat_100g":10.6,"carbohydrates_100g":57.5,"sugars_100g":56.3,"proteins_100g":6.3,"salt_100g":0.107,"fiber_100g":0}}}`

const fakeCategoryJSON = `{"count":5000,"page":1,"page_size":10,"products":[
  {"code":"5449000000996","product_name":"Coca-Cola","brands":"Coca-Cola","categories":"en:beverages","nutriscore_grade":"e"},
  {"code":"5000112548167","product_name":"Diet Coke","brands":"Coca-Cola","categories":"en:beverages","nutriscore_grade":"b"}
]}`

const fakeProductNotFoundJSON = `{"status":0,"status_verbose":"product not found","product":{}}`

func newTestClient(ts *httptest.Server) *openfoodfacts.Client {
	cfg := openfoodfacts.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return openfoodfacts.NewClient(cfg)
}

func TestSearchSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Search(context.Background(), "chocolate", 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestSearchParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeSearchJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	products, err := c.Search(context.Background(), "chocolate", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}

	p0 := products[0]
	if p0.Name != "Extra fine dark chocolate" {
		t.Errorf("products[0].Name = %q, want %q", p0.Name, "Extra fine dark chocolate")
	}
	if p0.Brands == "" {
		t.Error("products[0].Brands is empty")
	}
	if p0.NutriScore != "d" {
		t.Errorf("products[0].NutriScore = %q, want d", p0.NutriScore)
	}

	p1 := products[1]
	if p1.Name != "Excellence 70% cacao noir intense" {
		t.Errorf("products[1].Name = %q, want %q", p1.Name, "Excellence 70% cacao noir intense")
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
	_, err := c.Search(context.Background(), "nutella", 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "search.pl") {
		t.Errorf("expected search.pl endpoint, got path %q", gotPath)
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

	_, err := c.Search(context.Background(), "chocolate", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestProductParses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.Product(context.Background(), "3017620422003")
	if err != nil {
		t.Fatal(err)
	}
	if p.Barcode != "3017620422003" {
		t.Errorf("p.Barcode = %q, want 3017620422003", p.Barcode)
	}
	if p.Name != "Nutella" {
		t.Errorf("p.Name = %q, want Nutella", p.Name)
	}
	if p.Brands != "Nutella" {
		t.Errorf("p.Brands = %q, want Nutella", p.Brands)
	}
	if p.NutriScore != "e" {
		t.Errorf("p.NutriScore = %q, want e", p.NutriScore)
	}
	if p.EcoScore != "d" {
		t.Errorf("p.EcoScore = %q, want d", p.EcoScore)
	}
	if p.Nutrients == nil {
		t.Fatal("p.Nutrients is nil")
	}
	if p.Nutrients.EnergyKcal != 539 {
		t.Errorf("p.Nutrients.EnergyKcal = %v, want 539", p.Nutrients.EnergyKcal)
	}
	if p.Nutrients.Fat != 30.9 {
		t.Errorf("p.Nutrients.Fat = %v, want 30.9", p.Nutrients.Fat)
	}
}

func TestProductNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductNotFoundJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Product(context.Background(), "0000000000000")
	if err == nil {
		t.Error("expected error for not-found product, got nil")
	}
	if !strings.Contains(err.Error(), "product not found") {
		t.Errorf("error = %q, want 'product not found'", err.Error())
	}
}

func TestCategoryParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeCategoryJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	products, err := c.Category(context.Background(), "en:beverages", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(products))
	}
	if products[0].Name != "Coca-Cola" {
		t.Errorf("products[0].Name = %q, want Coca-Cola", products[0].Name)
	}
}

func TestNutrientsParsesNutriments(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeProductJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	p, err := c.Nutrients(context.Background(), "3017620422003")
	if err != nil {
		t.Fatal(err)
	}
	if p.Nutrients == nil {
		t.Fatal("p.Nutrients is nil")
	}
	if p.Nutrients.Protein != 6.3 {
		t.Errorf("p.Nutrients.Protein = %v, want 6.3", p.Nutrients.Protein)
	}
	if p.Nutrients.Sugars != 56.3 {
		t.Errorf("p.Nutrients.Sugars = %v, want 56.3", p.Nutrients.Sugars)
	}
	if p.Nutrients.Salt != 0.107 {
		t.Errorf("p.Nutrients.Salt = %v, want 0.107", p.Nutrients.Salt)
	}
}
