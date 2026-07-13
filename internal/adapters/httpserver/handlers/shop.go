// Package handlers implements HTTP handlers for the shop frontend.
package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/unpoly"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

// ShopHandler serves catalog-related pages: home, category, product, search, 404.
type ShopHandler struct {
	catalogSvc *service.CatalogService
	renderer   ports.Renderer
	logger     *slog.Logger
}

// NewShopHandler creates a ShopHandler.
func NewShopHandler(catalogSvc *service.CatalogService, renderer ports.Renderer) *ShopHandler {
	return &ShopHandler{
		catalogSvc: catalogSvc,
		renderer:   renderer,
		logger:     slog.New(slog.NewJSONHandler(os.Stderr, nil)),
	}
}

// Home serves the storefront home page.
func (h *ShopHandler) Home(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	data, err := h.catalogSvc.HomeData(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		h.renderer.RenderPartial(w, "product-grid", map[string]any{
			"Products": data.NewProducts,
			"Title":    "Nouveautés",
		})
		return
	}
	if err := h.renderer.Render(w, "home", data); err != nil {
		h.logger.Error("render home page", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Category serves a category page with product listing.
func (h *ShopHandler) Category(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	slug := r.PathValue("slug")
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "newest"
	}
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	data, err := h.catalogSvc.CategoryData(r.Context(), slug, sort, page)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.renderer.RenderStatus(w, http.StatusNotFound, "404", nil)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		h.renderer.RenderPartial(w, "product-grid", map[string]any{
			"Products":   data.Products,
			"Total":      data.Total,
			"Page":       data.Page,
			"TotalPages": data.TotalPages(),
		})
		return
	}
	h.renderer.Render(w, "category", data)
}

// Product serves a single product detail page.
func (h *ShopHandler) Product(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	slug := r.PathValue("slug")
	data, err := h.catalogSvc.ProductData(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			h.renderer.RenderStatus(w, http.StatusNotFound, "404", nil)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		h.renderer.RenderPartial(w, "product-card", data.Product)
		return
	}
	h.renderer.Render(w, "product", data)
}

// Search serves the search results page.
func (h *ShopHandler) Search(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	query := r.URL.Query().Get("q")
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	data, err := h.catalogSvc.Search(r.Context(), query, page)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		h.renderer.RenderPartial(w, "product-grid", map[string]any{
			"Products": data.Products,
			"Query":    data.Query,
			"Total":    data.Total,
			"Page":     data.Page,
		})
		return
	}
	h.renderer.Render(w, "search", data)
}

// NotFound serves the 404 page for unmatched routes.
func (h *ShopHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)
	w.WriteHeader(http.StatusNotFound)
	if unpoly.IsUnpolyRequest(r) {
		h.renderer.RenderPartial(w, "empty-state", map[string]any{
			"Title":   "Page introuvable",
			"Message": "La page que vous cherchez n'existe pas.",
		})
		return
	}
	h.renderer.RenderStatus(w, http.StatusNotFound, "404", nil)
}
