package handlers

import (
	"net/http"
	"strconv"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/middleware"
	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/unpoly"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

// AccountHandler serves authenticated account routes.
type AccountHandler struct {
	checkoutSvc *service.CheckoutService
	renderer    ports.Renderer
}

// NewAccountHandler creates an AccountHandler.
func NewAccountHandler(checkoutSvc *service.CheckoutService, renderer ports.Renderer) *AccountHandler {
	return &AccountHandler{
		checkoutSvc: checkoutSvc,
		renderer:    renderer,
	}
}

// Show displays the account overview page.
func (h *AccountHandler) Show(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	userID, _ := middleware.UserIDFromContext(r.Context())
	h.renderer.Render(w, "account", map[string]any{
		"UserID": userID,
	})
}

// Orders displays the user's order history.
func (h *AccountHandler) Orders(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	userID, _ := middleware.UserIDFromContext(r.Context())
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	// Note: CheckoutService doesn't expose order listing by user.
	// We pass basic data to the template and extend later when the
	// service gains a ListByUser method.
	h.renderer.Render(w, "account", map[string]any{
		"UserID":    userID,
		"Orders":    nil,
		"Page":      page,
		"ShowOrders": true,
	})
}
