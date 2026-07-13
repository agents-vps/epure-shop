// Package template implements the ports.Renderer interface using Go's html/template
// with embedded templates, Clone() pattern for thread safety, and FuncMap helpers.
package template

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	tpl "github.com/agents-vps/epure-shop/web/templates"

	"github.com/agents-vps/epure-shop/internal/core/domain"
	"github.com/agents-vps/epure-shop/internal/core/ports"
)

// Renderer implements ports.Renderer using html/template.
type Renderer struct {
	mu        sync.RWMutex
	templates map[string]*template.Template // layout name -> parsed template
	funcs     template.FuncMap
}

// New creates a new Renderer by parsing all templates from the embedded filesystem.
func New() (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		funcs:     buildFuncMap(),
	}

	// Read layout entries
	entries, err := tpl.FS.ReadDir("layouts")
	if err != nil {
		return nil, fmt.Errorf("template: failed to read layouts dir: %w", err)
	}

	// Collect all partials
	partials, err := readAllFiles(tpl.FS, "partials")
	if err != nil {
		return nil, fmt.Errorf("template: failed to read partials: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gohtml") {
			continue
		}

		layoutName := strings.TrimSuffix(entry.Name(), ".gohtml")
		layoutPath := path.Join("layouts", entry.Name())

		tmpl := template.New(layoutName).Funcs(r.funcs)

		// Parse layout
		layoutContent, err := tpl.FS.ReadFile(layoutPath)
		if err != nil {
			return nil, fmt.Errorf("template: failed to read layout %s: %w", layoutName, err)
		}
		if _, err := tmpl.Parse(string(layoutContent)); err != nil {
			return nil, fmt.Errorf("template: failed to parse layout %s: %w", layoutName, err)
		}

		// Parse partials into layout
		for _, p := range partials {
			if _, err := tmpl.Parse(string(p.content)); err != nil {
				return nil, fmt.Errorf("template: failed to parse partial %s into layout %s: %w", p.name, layoutName, err)
			}
		}

		r.templates[layoutName] = tmpl
	}

	return r, nil
}

type fileEntry struct {
	name    string
	content string
}

func readAllFiles(fs embed.FS, dir string) ([]fileEntry, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []fileEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gohtml") {
			continue
		}
		p := path.Join(dir, entry.Name())
		content, err := fs.ReadFile(p)
		if err != nil {
			return nil, err
		}
		result = append(result, fileEntry{name: entry.Name(), content: string(content)})
	}
	return result, nil
}

// Clone returns a copy of the layout template, ready to parse page content into.
// This ensures thread safety — each render gets its own template instance.
func (r *Renderer) Clone(layout string) (*template.Template, error) {
	r.mu.RLock()
	tmpl, ok := r.templates[layout]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("template: layout %q not found", layout)
	}

	cloned, err := tmpl.Clone()
	if err != nil {
		return nil, fmt.Errorf("template: failed to clone layout %q: %w", layout, err)
	}

	return cloned, nil
}

// renderPage loads and parses the given page template into the cloned layout, then executes it.
func (r *Renderer) renderPage(w io.Writer, layout, page string, data any) error {
	tmpl, err := r.Clone(layout)
	if err != nil {
		return err
	}

	// Determine page path — try pages/ first, then admin/
	pageContent, err := readPageFile(page)
	if err != nil {
		return err
	}

	if _, err := tmpl.Parse(string(pageContent)); err != nil {
		return fmt.Errorf("template: failed to parse page %q: %w", page, err)
	}

	// Wrap data to provide safe defaults for layout-level fields
	wrapped := wrapData(data)

	// Buffer-before-write pattern
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, layout, wrapped); err != nil {
		return fmt.Errorf("template: failed to execute layout %q with page %q: %w", layout, page, err)
	}

	_, err = io.Copy(w, &buf)
	return err
}

// wrapData ensures layout-level fields have safe defaults.
func wrapData(data any) map[string]any {
	m := make(map[string]any)
	m["CartCount"] = 0
	m["Query"] = ""
	m["Categories"] = []any{}
	m["ActiveCategory"] = ""

	// If data is already a map, merge it
	if dm, ok := data.(map[string]any); ok {
		for k, v := range dm {
			m[k] = v
		}
		return m
	}

	// Otherwise, embed the struct as "Data" and also flatten known fields via reflection
	m["Data"] = data

	// Use reflection to extract known fields
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.IsExported() {
				m[field.Name] = v.Field(i).Interface()
			}
		}
	}
	return m
}

func readPageFile(page string) ([]byte, error) {
	pagePath := path.Join("pages", page+".gohtml")
	content, err := tpl.FS.ReadFile(pagePath)
	if err == nil {
		return content, nil
	}
	// Try admin path
	pagePath = path.Join("admin", page+".gohtml")
	return tpl.FS.ReadFile(pagePath)
}

// Render writes a full page using the given layout and page template.
func (r *Renderer) Render(w io.Writer, page string, data any) error {
	layout := inferLayout(page, data)
	return r.renderPage(w, layout, page, data)
}

// RenderPartial renders a single partial template. Partials are raw HTML fragments
// without {{define}} wrappers — they are parsed as-is and executed directly.
func (r *Renderer) RenderPartial(w io.Writer, partial string, data any) error {
	partialPath := path.Join("partials", partial+".gohtml")
	content, err := tpl.FS.ReadFile(partialPath)
	if err != nil {
		return fmt.Errorf("template: partial %q not found: %w", partial, err)
	}

	tmpl := template.New(partial).Funcs(r.funcs)
	if _, err := tmpl.Parse(string(content)); err != nil {
		return fmt.Errorf("template: failed to parse partial %q: %w", partial, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template: failed to execute partial %q: %w", partial, err)
	}

	_, err = io.Copy(w, &buf)
	return err
}

// RenderStatus renders a full page (status is expected to be set by the caller).
func (r *Renderer) RenderStatus(w io.Writer, status int, page string, data any) error {
	return r.Render(w, page, data)
}

// Ensure Renderer satisfies ports.Renderer.
var _ ports.Renderer = (*Renderer)(nil)

// inferLayout determines which layout to use based on the page.
func inferLayout(page string, data any) string {
	// Admin pages
	adminPages := []string{
		"dashboard", "orders", "order-detail",
		"products", "product-edit", "customers",
		"categories", "discounts", "settings",
	}
	for _, p := range adminPages {
		if strings.HasPrefix(page, p) {
			return "admin"
		}
	}

	// Auth pages
	if page == "login" || page == "register" {
		return "auth"
	}

	// Checkout layout for checkout, order-confirmation, 404
	if page == "checkout" || page == "order-confirmation" || page == "404" {
		return "checkout"
	}

	return "base"
}

// buildFuncMap returns the standard FuncMap with template helpers.
func buildFuncMap() template.FuncMap {
	return template.FuncMap{
		"money": func(v any) string {
			switch val := v.(type) {
			case domain.Money:
				return formatMoney(val.Cents())
			case *domain.Money:
				if val == nil {
					return "0,00 €"
				}
				return formatMoney(val.Cents())
			case int64:
				return formatMoney(val)
			case int:
				return formatMoney(int64(val))
			default:
				return "—"
			}
		},
		"date": func(v any) string {
			switch val := v.(type) {
			case time.Time:
				return val.Format("02/01/2006")
			case *time.Time:
				if val == nil {
					return "—"
				}
				return val.Format("02/01/2006")
			default:
				return "—"
			}
		},
		"plural": func(n int, singular, plural string) string {
			if n <= 1 {
				return singular
			}
			return plural
		},
		"seq": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict: odd number of arguments")
			}
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				key, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key %d is not a string", i)
				}
				m[key] = pairs[i+1]
			}
			return m, nil
		},
		"add": func(a, b int) int {
			return a + b
		},
		"multiply": func(a any, b int) int64 {
			switch v := a.(type) {
			case domain.Money:
				return v.Cents() * int64(b)
			default:
				return 0
			}
		},
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"slice": func(s string, start, end int) string {
			if start < 0 {
				start = 0
			}
			if end > len(s) {
				end = len(s)
			}
			if start > end {
				return ""
			}
			return s[start:end]
		},
		"default": func(def, val any) any {
			if val == nil {
				return def
			}
			switch v := val.(type) {
			case string:
				if v == "" {
					return def
				}
			case int:
				if v == 0 {
					return def
				}
			case int64:
				if v == 0 {
					return def
				}
			}
			return val
		},
	}
}

func formatMoney(cents int64) string {
	if cents < 0 {
		return fmt.Sprintf("-%s", formatMoney(-cents))
	}
	euros := cents / 100
	decPart := cents % 100

	intStr := formatIntWithSpaces(euros)
	return fmt.Sprintf("%s,%02d €", intStr, decPart)
}

func formatIntWithSpaces(n int64) string {
	if n == 0 {
		return "0"
	}
	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteByte(' ')
		}
		result.WriteRune(c)
	}
	return result.String()
}
