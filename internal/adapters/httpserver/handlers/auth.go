package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/middleware"
	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/unpoly"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

const sessionCookieName = "sid"

// AuthHandler serves authentication routes: login, register, logout.
type AuthHandler struct {
	authSvc  *service.AuthService
	cartSvc  *service.CartService
	renderer ports.Renderer
	isDev    bool
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authSvc *service.AuthService, cartSvc *service.CartService, renderer ports.Renderer, isDev bool) *AuthHandler {
	return &AuthHandler{
		authSvc:  authSvc,
		cartSvc:  cartSvc,
		renderer: renderer,
		isDev:    isDev,
	}
}

// ShowLogin displays the login form.
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	// Already authenticated? Redirect to account or next
	if _, ok := middleware.UserIDFromContext(r.Context()); ok {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}

	h.renderer.Render(w, "login", map[string]any{
		"Next": r.URL.Query().Get("next"),
	})
}

// Login processes the login form.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")

	user, sessionToken, err := h.authSvc.Login(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			if unpoly.IsUnpolyRequest(r) {
				unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"email": "Email ou mot de passe incorrect"})
				return
			}
			h.renderer.Render(w, "login", map[string]any{
				"Error": "Email ou mot de passe incorrect",
				"Email": email,
			})
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Set session cookie
	h.setSessionCookie(w, sessionToken, 7*24*time.Hour)

	// Merge guest cart with user account
	guestToken, _ := h.cartToken(r)
	if guestToken != "" && user != nil {
		_ = h.cartSvc.MergeGuestCart(r.Context(), guestToken, user.ID)
	}
	h.clearCartCookie(w)

	// Determine redirect target
	next := r.FormValue("next")
	if next == "" {
		next = "/account"
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.SetLocation(w, next, "GET")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// ShowRegister displays the registration form.
func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	if _, ok := middleware.UserIDFromContext(r.Context()); ok {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}

	h.renderer.Render(w, "register", nil)
}

// Register processes the registration form.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	name := strings.TrimSpace(r.FormValue("name"))
	password := r.FormValue("password")

	// Basic validation
	validationErrors := unpoly.ValidationErrors{}
	if email == "" || !strings.Contains(email, "@") {
		validationErrors["email"] = "Email requis"
	}
	if name == "" {
		validationErrors["name"] = "Nom requis"
	}
	if len(password) < 8 {
		validationErrors["password"] = "8 caractères minimum"
	}
	if len(validationErrors) > 0 {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.WriteValidationErrors(w, validationErrors)
			return
		}
		h.renderer.Render(w, "register", map[string]any{
			"Errors": validationErrors,
			"Email":  email,
			"Name":   name,
		})
		return
	}

	user, err := h.authSvc.Register(r.Context(), email, name, password)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			if unpoly.IsUnpolyRequest(r) {
				unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"email": "Cet email est déjà utilisé"})
				return
			}
			h.renderer.Render(w, "register", map[string]any{
				"Error": "Cet email est déjà utilisé",
				"Email": email,
				"Name":  name,
			})
			return
		}
		if errors.Is(err, service.ErrValidation) {
			if unpoly.IsUnpolyRequest(r) {
				unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"password": "Mot de passe invalide"})
				return
			}
			h.renderer.Render(w, "register", map[string]any{
				"Error": "Données invalides",
				"Email": email,
				"Name":  name,
			})
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Auto-login after registration: create session
	_, sessionToken, err := h.authSvc.Login(r.Context(), email, password)
	if err == nil {
		h.setSessionCookie(w, sessionToken, 7*24*time.Hour)
	}

	// Merge guest cart
	guestToken, _ := h.cartToken(r)
	if guestToken != "" && user != nil {
		_ = h.cartSvc.MergeGuestCart(r.Context(), guestToken, user.ID)
	}
	h.clearCartCookie(w)

	if unpoly.IsUnpolyRequest(r) {
		unpoly.SetLocation(w, "/account", "GET")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/account", http.StatusSeeOther)
}

// Logout destroys the session and redirects to home.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.authSvc.Logout(r.Context(), cookie.Value)
	}

	// Clear session cookie
	h.clearSessionCookie(w)

	if unpoly.IsUnpolyRequest(r) {
		unpoly.SetLocation(w, "/", "GET")
		w.WriteHeader(http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// setSessionCookie sets the session cookie with security attributes.
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   !h.isDev,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie removes the session cookie.
func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !h.isDev,
		SameSite: http.SameSiteLaxMode,
	})
}

// cartToken reads the cart cookie (reuses the const from cart.go).
func (h *AuthHandler) cartToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cartCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value, false
	}
	return "", false
}

// clearCartCookie removes the cart cookie.
func (h *AuthHandler) clearCartCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cartCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
