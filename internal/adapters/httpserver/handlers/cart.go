package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/unpoly"
	"github.com/agents-vps/epure-shop/internal/core/domain/id"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

const cartCookieName = "cart"
const cartCookieMaxAge = 30 * 24 * time.Hour // 30 days

// CartHandler serves cart-related routes: view, add, update, remove, apply discount.
type CartHandler struct {
	cartSvc  *service.CartService
	renderer ports.Renderer
}

// NewCartHandler creates a CartHandler.
func NewCartHandler(cartSvc *service.CartService, renderer ports.Renderer) *CartHandler {
	return &CartHandler{cartSvc: cartSvc, renderer: renderer}
}

// View serves the cart page.
func (h *CartHandler) View(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, _ := h.cartToken(w, r)
	cart, err := h.cartSvc.GetCart(r.Context(), token)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		h.renderer.RenderPartial(w, "cart-body", cart)
		return
	}
	h.renderer.Render(w, "cart", cart)
}

// Add adds an item to the cart.
func (h *CartHandler) Add(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, isNew := h.cartToken(w, r)

	productID := r.FormValue("product_id")
	qty := 1
	if q, err := strconv.Atoi(r.FormValue("qty")); err == nil && q > 0 {
		qty = q
	}

	cart, err := h.cartSvc.AddItem(r.Context(), token, productID, qty)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			if unpoly.IsUnpolyRequest(r) {
				unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"product_id": "Produit introuvable"})
				return
			}
			http.Error(w, "Produit introuvable", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if isNew {
		h.setCartCookie(w, token, cartCookieMaxAge)
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "cart:updated", Target: ".cart-drawer"},
		})
		h.renderer.RenderPartial(w, "cart-body", cart)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// Update updates the quantity of a cart item.
func (h *CartHandler) Update(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, _ := h.cartToken(w, r)
	productID := r.PathValue("id")
	qty := 1
	if q, err := strconv.Atoi(r.FormValue("qty")); err == nil && q > 0 {
		qty = q
	}

	cart, err := h.cartSvc.UpdateQuantity(r.Context(), token, productID, qty)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "cart:updated", Target: ".cart-drawer"},
		})
		h.renderer.RenderPartial(w, "cart-body", cart)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// Remove removes an item from the cart.
func (h *CartHandler) Remove(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, _ := h.cartToken(w, r)
	productID := r.PathValue("id")

	cart, err := h.cartSvc.RemoveItem(r.Context(), token, productID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "cart:updated", Target: ".cart-drawer"},
		})
		h.renderer.RenderPartial(w, "cart-body", cart)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// ApplyDiscount applies a discount code to the cart.
func (h *CartHandler) ApplyDiscount(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, _ := h.cartToken(w, r)
	code := r.FormValue("code")

	cart, err := h.cartSvc.ApplyDiscount(r.Context(), token, code)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDiscount) || errors.Is(err, service.ErrNotFound) {
			if unpoly.IsUnpolyRequest(r) {
				unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"code": "Code promo invalide ou expiré"})
				return
			}
			// Re-render cart page with error for non-Unpoly
			cart, _ = h.cartSvc.GetCart(r.Context(), token)
			h.renderer.Render(w, "cart", map[string]any{
				"Cart":  cart,
				"Error": "Code promo invalide ou expiré",
			})
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "cart:updated", Target: ".cart-drawer"},
		})
		h.renderer.RenderPartial(w, "cart-summary", cart)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

// cartToken reads the cart cookie or creates a new token and sets the cookie.
// Returns the token and whether a new cookie was created.
func (h *CartHandler) cartToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cartCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value, false
	}
	token := id.NewV4()
	h.setCartCookie(w, token, cartCookieMaxAge)
	return token, true
}

// setCartCookie sets the cart cookie with standard attributes.
func (h *CartHandler) setCartCookie(w http.ResponseWriter, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cartCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
