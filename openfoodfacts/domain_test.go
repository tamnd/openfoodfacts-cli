package openfoodfacts

import (
	"testing"
)

// These tests exercise the URI driver's pure string functions offline.
// HTTP behaviour is covered in openfoodfacts_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "openfoodfacts" {
		t.Errorf("Scheme = %q, want openfoodfacts", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "openfoodfacts" {
		t.Errorf("Identity.Binary = %q, want openfoodfacts", info.Identity.Binary)
	}
}

func TestClassifyBarcode(t *testing.T) {
	typ, id, err := Domain{}.Classify("3017620422003")
	if err != nil || typ != "product" || id != "3017620422003" {
		t.Errorf("Classify(%q) = (%q, %q, %v), want (product, 3017620422003, nil)", "3017620422003", typ, id, err)
	}
}

func TestClassifyShortBarcode(t *testing.T) {
	// 8 digits is minimum barcode (EAN-8)
	typ, id, err := Domain{}.Classify("12345678")
	if err != nil || typ != "product" || id != "12345678" {
		t.Errorf("Classify(%q) = (%q, %q, %v), want (product, 12345678, nil)", "12345678", typ, id, err)
	}
}

func TestClassifyQuery(t *testing.T) {
	typ, id, err := Domain{}.Classify("chocolate")
	if err != nil || typ != "query" || id != "chocolate" {
		t.Errorf("Classify(%q) = (%q, %q, %v), want (query, chocolate, nil)", "chocolate", typ, id, err)
	}
}

func TestClassifyTooShortForBarcode(t *testing.T) {
	// 7 digits is not a barcode
	typ, id, err := Domain{}.Classify("1234567")
	if err != nil || typ != "query" || id != "1234567" {
		t.Errorf("Classify(%q) = (%q, %q, %v), want (query, 1234567, nil)", "1234567", typ, id, err)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestLocateProduct(t *testing.T) {
	got, err := Domain{}.Locate("product", "3017620422003")
	want := "https://world.openfoodfacts.org/product/3017620422003"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateQuery(t *testing.T) {
	got, err := Domain{}.Locate("query", "chocolate")
	want := "https://world.openfoodfacts.org/cgi/search.pl?search_terms=chocolate&action=process"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("unknown", "foo")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

func TestIsBarcode(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"3017620422003", true},   // EAN-13
		{"737628064502", true},    // UPC-A 12 digits
		{"12345678", true},        // EAN-8
		{"1234567", false},        // 7 digits, too short
		{"chocolate", false},      // text
		{"123abc456", false},      // mixed
		{"", false},               // empty
	}
	for _, tc := range cases {
		got := isBarcode(tc.input)
		if got != tc.want {
			t.Errorf("isBarcode(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
