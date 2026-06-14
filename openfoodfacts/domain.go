package openfoodfacts

import (
	"context"
	"time"

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
		Args:    []kit.Arg{{Name: "barcode", Help: "barcode string (e.g. 3017620422003)"}},
	}, productOp)

	// search: full-text search for food products
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search for food products by name",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, searchOp)

	// category: browse products in a category
	kit.Handle(app, kit.OpMeta{
		Name:    "category",
		Group:   "read",
		List:    true,
		Summary: "List products in a category (e.g. en:beverages)",
		Args:    []kit.Arg{{Name: "name", Help: "category tag (e.g. en:beverages)"}},
	}, categoryOp)

	// nutrients: nutritional info for a product by barcode
	kit.Handle(app, kit.OpMeta{
		Name:    "nutrients",
		Group:   "read",
		Single:  true,
		Summary: "Fetch nutritional information for a product by barcode",
		Args:    []kit.Arg{{Name: "barcode", Help: "barcode string (e.g. 3017620422003)"}},
	}, nutrientsOp)
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
	Barcode string        `kit:"arg" help:"barcode string (e.g. 3017620422003)"`
	Delay   time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client  *Client       `kit:"inject"`
}

type searchInput struct {
	Query  string        `kit:"arg" help:"search query"`
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Page   int           `kit:"flag" help:"page number (1-based)"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type categoryInput struct {
	Name   string        `kit:"arg" help:"category tag (e.g. en:beverages)"`
	Limit  int           `kit:"flag,inherit" help:"max results"`
	Delay  time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client *Client       `kit:"inject"`
}

type nutrientsInput struct {
	Barcode string        `kit:"arg" help:"barcode string (e.g. 3017620422003)"`
	Delay   time.Duration `kit:"flag,inherit" help:"minimum spacing between requests"`
	Client  *Client       `kit:"inject"`
}

// --- handlers ---

func productOp(ctx context.Context, in productInput, emit func(Product) error) error {
	p, err := in.Client.Product(ctx, in.Barcode)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

func searchOp(ctx context.Context, in searchInput, emit func(Product) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	products, err := in.Client.Search(ctx, in.Query, limit, in.Page)
	if err != nil {
		return mapErr(err)
	}
	for _, p := range products {
		if err := emit(p); err != nil {
			return err
		}
	}
	return nil
}

func categoryOp(ctx context.Context, in categoryInput, emit func(Product) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	products, err := in.Client.Category(ctx, in.Name, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, p := range products {
		if err := emit(p); err != nil {
			return err
		}
	}
	return nil
}

func nutrientsOp(ctx context.Context, in nutrientsInput, emit func(Product) error) error {
	p, err := in.Client.Nutrients(ctx, in.Barcode)
	if err != nil {
		return mapErr(err)
	}
	return emit(p)
}

// --- Resolver ---

// Classify turns an input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty openfoodfacts reference")
	}
	return "product", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "product":
		return "https://world.openfoodfacts.org/product/" + id, nil
	default:
		return "", errs.Usage("openfoodfacts has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind.
func mapErr(err error) error {
	return err
}
