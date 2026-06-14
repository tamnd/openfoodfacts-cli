package openfoodfacts

import (
	"context"
	"unicode"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes openfoodfacts as a kit Domain driver.
//
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/openfoodfacts-cli/openfoodfacts"
//
// The same Domain also builds the standalone openfoodfacts binary (see cli.NewApp).
func init() { kit.Register(Domain{}) }

// Domain is the openfoodfacts driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "openfoodfacts",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "openfoodfacts",
			Short:  "Search food products and fetch nutritional data from Open Food Facts",
			Long: `openfoodfacts fetches food product data from the Open Food Facts API
(world.openfoodfacts.org), the Wikipedia of food.
No authentication required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/openfoodfacts-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// product: fetch a product by barcode
	kit.Handle(app, kit.OpMeta{
		Name:    "product",
		Group:   "read",
		Single:  true,
		Summary: "Fetch a food product by barcode (EAN-13 or UPC)",
		Args:    []kit.Arg{{Name: "barcode", Help: "product barcode (EAN-13, UPC-A, etc.)"}},
	}, productOp)

	// search: full-text search for food products
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search for food products by name",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, searchOp)

	// category: list products in a taxonomy category
	kit.Handle(app, kit.OpMeta{
		Name:    "category",
		Group:   "read",
		List:    true,
		Summary: "List food products in a category (e.g. chocolates, biscuits)",
		Args:    []kit.Arg{{Name: "category", Help: "Open Food Facts taxonomy category slug"}},
	}, categoryOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type productInput struct {
	Barcode string  `kit:"arg" help:"product barcode (EAN-13, UPC-A, etc.)"`
	Client  *Client `kit:"inject"`
}

type searchInput struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"20"`
	Page   int     `kit:"flag,inherit" help:"page number" default:"1"`
	Client *Client `kit:"inject"`
}

type categoryInput struct {
	Category string  `kit:"arg" help:"Open Food Facts category slug (e.g. chocolates)"`
	Limit    int     `kit:"flag,inherit" help:"max results" default:"20"`
	Client   *Client `kit:"inject"`
}

// --- handlers ---

func productOp(ctx context.Context, in productInput, emit func(Product) error) error {
	p, err := in.Client.GetProduct(ctx, in.Barcode)
	if err != nil {
		return err
	}
	return emit(*p)
}

func searchOp(ctx context.Context, in searchInput, emit func(SearchResult) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	results, err := in.Client.Search(ctx, in.Query, limit, page)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func categoryOp(ctx context.Context, in categoryInput, emit func(SearchResult) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	results, err := in.Client.Category(ctx, in.Category, limit)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// isBarcode returns true if s is 8 or more digit characters (EAN-8, EAN-13, UPC-A, etc.).
func isBarcode(s string) bool {
	if len(s) < 8 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// Classify turns an input into the canonical (type, id).
// 8+ digit strings are treated as barcodes ("product"), everything else as a search query ("query").
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty openfoodfacts reference")
	}
	if isBarcode(input) {
		return "product", input, nil
	}
	return "query", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "product":
		return "https://world.openfoodfacts.org/product/" + id, nil
	case "query":
		return "https://world.openfoodfacts.org/cgi/search.pl?search_terms=" + id + "&action=process", nil
	default:
		return "", errs.Usage("openfoodfacts has no resource type %q", uriType)
	}
}
