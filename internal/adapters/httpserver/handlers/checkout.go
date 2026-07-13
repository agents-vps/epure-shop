package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/middleware"
	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/unpoly"
	"github.com/agents-vps/epure-shop/internal/core/domain/id"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

// CheckoutHandler serves checkout-related routes: show, submit, confirmation.
type CheckoutHandler struct {
	checkoutSvc *service.CheckoutService
	cartSvc     *service.CartService
	renderer    ports.Renderer
}

// NewCheckoutHandler creates a CheckoutHandler.
func NewCheckoutHandler(checkoutSvc *service.CheckoutService, cartSvc *service.CartService, renderer ports.Renderer) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutSvc: checkoutSvc,
		cartSvc:     cartSvc,
		renderer:    renderer,
	}
}

// Show validates the cart and displays the checkout form.
func (h *CheckoutHandler) Show(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, _ := h.cartToken(r)
	cart, err := h.checkoutSvc.ValidateCheckout(r.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrEmptyCart) {
			http.Redirect(w, r, "/cart", http.StatusSeeOther)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Pre-fill email if user is logged in
	userID, _ := middleware.UserIDFromContext(r.Context())
	_ = userID // future: load user profile for pre-filling

	h.renderer.Render(w, "checkout", map[string]any{
		"Cart":    cart,
		"IsUnpoly": unpoly.IsUnpolyRequest(r),
	})
}

// Submit processes the checkout form and places the order.
func (h *CheckoutHandler) Submit(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	token, _ := h.cartToken(r)
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	shippingAddress := strings.TrimSpace(r.FormValue("shipping_address"))
	idempotencyKey := r.FormValue("idempotency_key")

	// Validate required fields
	if email == "" {
		h.renderCheckoutError(w, r, "L'email est requis")
		return
	}

	order, err := h.checkoutSvc.PlaceOrder(r.Context(), token, email, shippingAddress, idempotencyKey)
	if err != nil {
		if errors.Is(err, service.ErrEmptyCart) {
			http.Redirect(w, r, "/cart", http.StatusSeeOther)
			return
		}
		if errors.Is(err, service.ErrValidation) {
			h.renderCheckoutError(w, r, err.Error())
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Clear cart cookie
	h.clearCartCookie(w)

	// Redirect to confirmation — Unpoly follows via X-Up-Location
	confURL := "/orders/" + order.Ref + "/confirmation"
	if unpoly.IsUnpolyRequest(r) {
		unpoly.SetLocation(w, confURL, "GET")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, confURL, http.StatusSeeOther)
}

// Confirmation shows the order confirmation page.
func (h *CheckoutHandler) Confirmation(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	ref := r.PathValue("ref")
	// Note: CheckoutService doesn't expose ByRef. We render a minimal
	// confirmation page — the order reference is passed to the template.
	h.renderer.Render(w, "order-confirmation", map[string]any{
		"Ref": ref,
	})
}

// renderCheckoutError re-renders the checkout page with an error message.
func (h *CheckoutHandler) renderCheckoutError(w http.ResponseWriter, r *http.Request, msg string) {
	// Prevent infinite loops from gorilla/schema errors
	_ = r.ParseForm()

	token, _ := h.cartToken(r)
	cart, _ := h.cartSvc.GetCart(r.Context(), token)

	if unpoly.IsUnpolyRequest(r) {
		unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"email": msg})
		return
	}
	h.renderer.Render(w, "checkout", map[string]any{
		"Cart":  cart,
		"Error": msg,
	})
}

// cartToken reads the cart cookie used by the checkout handler.
func (h *CheckoutHandler) cartToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cartCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value, false
	}
	token := id.NewV4()
	return token, true
}

// clearCartCookie removes the cart cookie.
func (h *CheckoutHandler) clearCartCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cartCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
