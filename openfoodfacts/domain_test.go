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

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("3017620422003")
	if err != nil || typ != "product" || id != "3017620422003" {
		t.Errorf("Classify(%q) = (%q, %q, %v), want (product, 3017620422003, nil)", "3017620422003", typ, id, err)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("product", "3017620422003")
	want := "https://world.openfoodfacts.org/product/3017620422003"
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
