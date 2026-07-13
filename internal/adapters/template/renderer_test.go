package template

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/agents-vps/epure-shop/internal/core/domain"
)

func TestRenderAllPages(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	tests := []struct {
		page   string
		layout string
		data   any
	}{
		{"home", "base", homeTestData()},
		{"login", "auth", map[string]any{"CSRF": "test-csrf"}},
		{"checkout", "checkout", map[string]any{"CSRF": "test-csrf"}},
		{"dashboard", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "dashboard"}},
		{"404", "checkout", map[string]any{"CSRF": "test-csrf"}},
		{"category", "base", categoryTestData()},
		{"product", "base", productTestData()},
		{"cart", "base", map[string]any{"CSRF": "test-csrf", "Breadcrumbs": []domain.Category{}}},
		{"register", "auth", map[string]any{"CSRF": "test-csrf"}},
		{"account", "base", accountTestData()},
		{"search", "base", map[string]any{"CSRF": "test-csrf", "Query": "test", "TotalCount": 0, "Breadcrumbs": []domain.Category{}}},
		{"orders", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "orders", "Orders": []domain.Order{}}},
		{"order-detail", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "orders", "Order": orderTestData()}},
		{"products", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "products", "Products": []domain.Product{}}},
		{"product-edit", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "products"}},
		{"customers", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "customers", "Users": []domain.User{}}},
		{"categories", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "categories", "Categories": []domain.Category{}}},
		{"discounts", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "discounts", "Discounts": []domain.Discount{}}},
		{"settings", "admin", map[string]any{"CSRF": "test-csrf", "ActivePage": "settings"}},
		{"order-confirmation", "checkout", map[string]any{"CSRF": "test-csrf", "Order": orderTestData()}},
		{"login", "auth", map[string]any{"CSRF": "test-csrf", "Email": "admin@epure.fr"}},
	}

	for _, tt := range tests {
		t.Run(tt.page+"/"+tt.layout, func(t *testing.T) {
			var buf bytes.Buffer
			if err := r.Render(&buf, tt.page, tt.data); err != nil {
				t.Fatalf("Render error: %v", err)
			}
			out := buf.String()
			if len(out) < 100 {
				t.Fatalf("output too short: %d bytes", len(out))
			}
			if !strings.Contains(out, "<!DOCTYPE html>") {
				t.Fatal("missing DOCTYPE")
			}
			// All layouts should have CSRF meta
			if !strings.Contains(out, "csrf-token") {
				t.Fatal("missing csrf-token meta")
			}
			t.Logf("OK — %d bytes", len(out))
		})
	}
}

func TestRenderAllPartials(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	partials := []struct {
		name string
		data any
	}{
		{"header", map[string]any{"CartCount": 0, "Categories": sampleCats(), "ActiveCategory": ""}},
		{"footer", map[string]any{"Categories": sampleCats()}},
		{"product-card", &domain.Product{ID: "p1", Slug: "tasse-artisanale", Name: "Tasse artisanale", CategoryName: "Céramique", Price: 3400, Rating: 4.8, ReviewCount: 47}},
		{"product-grid", map[string]any{"Products": sampleProds(), "Columns": 3}},
		{"cart-item", map[string]any{"ProductID": "c1", "Product": &domain.Product{ID: "p1", Slug: "tasse-artisanale", Name: "Tasse artisanale", Price: 3400}, "Quantity": 2}},
		{"empty-state", map[string]string{"Title": "Test vide", "Text": "Rien à afficher."}},
		{"skeleton-card", nil},
	}

	for _, tt := range partials {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := r.RenderPartial(&buf, tt.name, tt.data); err != nil {
				t.Fatalf("RenderPartial error: %v", err)
			}
			if buf.Len() < 10 {
				t.Fatal("output too short")
			}
			t.Logf("OK — %d bytes", buf.Len())
		})
	}
}

func TestRenderPartialNotFound(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	var buf bytes.Buffer
	err = r.RenderPartial(&buf, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent partial")
	}
}

func TestRenderPageNotFound(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	var buf bytes.Buffer
	err = r.Render(&buf, "nonexistent", map[string]any{"CSRF": "test"})
	if err == nil {
		t.Fatal("expected error for nonexistent page")
	}
}

func TestMoneyFormatter(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{0, "0,00 €"},
		{100, "1,00 €"},
		{3400, "34,00 €"},
		{12000, "120,00 €"},
		{145000, "1 450,00 €"},
		{1950000, "19 500,00 €"},
	}

	buf := new(bytes.Buffer)
	w := new(strings.Builder)

	for _, c := range cases {
		buf.Reset()
		w.Reset()
		// Use RenderPartial to test money formatting indirectly
		// Actually, just test formatMoney directly
		result := formatMoney(c.cents)
		if result != c.want {
			t.Errorf("formatMoney(%d) = %q, want %q", c.cents, result, c.want)
		}
		_ = buf
	}
}

// --- test data helpers ---

func homeTestData() map[string]any {
	return map[string]any{
		"CSRF":             "test-csrf",
		"Categories":       sampleCats(),
		"FeaturedProducts": sampleProds(),
		"BestSellers":      sampleProds(),
		"CartCount":        0,
	}
}

func categoryTestData() map[string]any {
	return map[string]any{
		"CSRF": "test-csrf",
		"Category": map[string]any{
			"Name":        "Céramique",
			"Slug":        "ceramique",
			"Description": "Pièces uniques",
		},
		"Categories":   sampleCats(),
		"Products":     sampleProds(),
		"TotalCount":   3,
		"Breadcrumbs":  []domain.Category{},
		"Filters":      map[string]int{"PriceMin": 0, "PriceMax": 500},
		"Sort":         "relevance",
		"ActiveCategory": "ceramique",
		"CartCount":    0,
		"Pagination": map[string]any{
			"HasPrev": false, "HasNext": false, "Pages": []map[string]any{},
		},
	}
}

func productTestData() map[string]any {
	return map[string]any{
		"CSRF": "test-csrf",
		"Product": &domain.Product{
			ID:           "p1",
			Slug:         "tasse-artisanale",
			Name:         "Tasse artisanale — Bleu Azur",
			CategoryName: "Céramique",
			Description:  "Tasse artisanale en grès émaillé.",
			Price:        domain.Money(3400),
			Rating:       4.8,
			ReviewCount:  47,
			Status:       "published",
		},
		"Categories":  sampleCats(),
		"Breadcrumbs": []domain.Category{},
		"CartCount":   0,
	}
}

func accountTestData() map[string]any {
	return map[string]any{
		"CSRF":       "test-csrf",
		"Categories": sampleCats(),
		"ActiveTab":  "profile",
		"User": map[string]string{
			"Name":  "Marie",
			"Surname": "Dupont",
			"Email": "marie@exemple.com",
		},
		"Breadcrumbs": []domain.Category{},
		"CartCount":   0,
	}
}

func orderTestData() any {
	return map[string]any{
		"CustomerName": "Marie Dupont",
		"Email":        "marie@exemple.com",
		"Ref":          "ABC12345",
		"Subtotal":     domain.Money(12300),
		"ShippingCost": domain.Money(490),
		"Total":        domain.Money(12790),
		"Items": []struct {
			Name      string
			Quantity  int
			UnitPrice domain.Money
		}{
			{Name: "Tasse artisanale", Quantity: 1, UnitPrice: 3400},
			{Name: "Vase en grès", Quantity: 1, UnitPrice: 8900},
		},
		"DeliveryMethod": "Standard (3-5 jours)",
		"ShippingAddress": "123 rue de la Céramique, 75001 Paris",
		"Status":         "confirmed",
		"StatusBadge":    "success",
	}
}

func sampleCats() []domain.Category {
	return []domain.Category{
		{ID: "1", Slug: "ceramique", Name: "Céramique", ProductCount: 24},
		{ID: "2", Slug: "textile", Name: "Textile", ProductCount: 15},
		{ID: "3", Slug: "decoration", Name: "Décoration", ProductCount: 18},
		{ID: "4", Slug: "art-de-la-table", Name: "Art de la table", ProductCount: 12},
		{ID: "5", Slug: "luminaires", Name: "Luminaires", ProductCount: 8},
		{ID: "6", Slug: "jardin", Name: "Jardin", ProductCount: 5},
	}
}

func sampleProds() []domain.Product {
	return []domain.Product{
		{ID: "p1", Slug: "tasse-artisanale", Name: "Tasse artisanale — Bleu Azur", CategoryName: "Céramique", Price: 3400, Rating: 4.8, ReviewCount: 47, Status: "published"},
		{ID: "p2", Slug: "plaid-laine", Name: "Plaid en laine mérinos", CategoryName: "Textile", Price: 12000, Rating: 5.0, ReviewCount: 8, Status: "published"},
		{ID: "p3", Slug: "vase-gres", Name: "Vase en grès — Grand modèle", CategoryName: "Décoration", Price: 8900, Rating: 4.5, ReviewCount: 16, Status: "published"},
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
