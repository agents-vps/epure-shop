// Package handlers provides HTTP handler implementations for the Épure shop.
package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/unpoly"
	"github.com/agents-vps/epure-shop/internal/core/domain"
	"github.com/agents-vps/epure-shop/internal/core/domain/id"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

// AdminHandler serves all /admin routes.
type AdminHandler struct {
	AdminService *service.AdminService
	AuthService  *service.AuthService
	Renderer     ports.Renderer
	CSRF         func(r *http.Request) string // CSRF token provider
}

// ServeHTTP dispatches admin routes.
func (h *AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	// Login (no admin layout)
	case path == "/admin/login" && r.Method == http.MethodGet:
		h.ShowLogin(w, r)
	case path == "/admin/login" && r.Method == http.MethodPost:
		h.Login(w, r)

	// Dashboard
	case path == "/admin" && r.Method == http.MethodGet:
		h.Dashboard(w, r)

	// Orders
	case path == "/admin/orders" && r.Method == http.MethodGet:
		h.Orders(w, r)
	case matchPath(path, `/admin/orders/([^/]+)/status`) && r.Method == http.MethodPatch:
		id := extractPathSeg(path, `/admin/orders/([^/]+)/status`, 1)
		h.UpdateOrderStatus(w, r, id)
	case matchPath(path, `/admin/orders/([^/]+)`) && r.Method == http.MethodGet:
		id := extractPathSeg(path, `/admin/orders/([^/]+)`, 1)
		h.OrderDetail(w, r, id)

	// Products
	case path == "/admin/products" && r.Method == http.MethodGet:
		h.Products(w, r)
	case path == "/admin/products" && r.Method == http.MethodPost:
		h.ProductCreate(w, r)
	case matchPath(path, `/admin/products/([^/]+)`) && r.Method == http.MethodGet:
		id := extractPathSeg(path, `/admin/products/([^/]+)`, 1)
		h.ProductEdit(w, r, id)
	case matchPath(path, `/admin/products/([^/]+)`) && r.Method == http.MethodPut:
		id := extractPathSeg(path, `/admin/products/([^/]+)`, 1)
		h.ProductUpdate(w, r, id)
	case matchPath(path, `/admin/products/([^/]+)`) && r.Method == http.MethodDelete:
		id := extractPathSeg(path, `/admin/products/([^/]+)`, 1)
		h.ProductDelete(w, r, id)

	// Customers
	case path == "/admin/customers" && r.Method == http.MethodGet:
		h.Customers(w, r)

	// Discounts
	case path == "/admin/discounts" && r.Method == http.MethodGet:
		h.Discounts(w, r)
	case path == "/admin/discounts" && r.Method == http.MethodPost:
		h.DiscountCreate(w, r)
	case matchPath(path, `/admin/discounts/([^/]+)`) && r.Method == http.MethodDelete:
		id := extractPathSeg(path, `/admin/discounts/([^/]+)`, 1)
		h.DiscountDelete(w, r, id)

	// Categories
	case path == "/admin/categories" && r.Method == http.MethodGet:
		h.Categories(w, r)
	case path == "/admin/categories" && r.Method == http.MethodPost:
		h.CategoryCreate(w, r)
	case matchPath(path, `/admin/categories/([^/]+)`) && r.Method == http.MethodDelete:
		id := extractPathSeg(path, `/admin/categories/([^/]+)`, 1)
		h.CategoryDelete(w, r, id)

	// Settings
	case path == "/admin/settings" && r.Method == http.MethodGet:
		h.Settings(w, r)

	default:
		http.NotFound(w, r)
	}
}

// matchPath reports whether path matches the regexp pattern.
func matchPath(path, pattern string) bool {
	ok, _ := regexp.MatchString(`^`+pattern+`$`, path)
	return ok
}

// extractPathSeg extracts a capture group from path by pattern.
func extractPathSeg(path, pattern string, group int) string {
	re := regexp.MustCompile(`^` + pattern + `$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) > group {
		return matches[group]
	}
	return ""
}

// ─── Helpers ───

// adminData returns the base template data for admin pages.
func (h *AdminHandler) adminData(r *http.Request, activePage string) map[string]any {
	return map[string]any{
		"ActivePage":        activePage,
		"CSRF":              h.csrfToken(r),
		"NotificationCount": 0,
		"AdminInitials":     "AD",
	}
}

func (h *AdminHandler) csrfToken(r *http.Request) string {
	if h.CSRF != nil {
		return h.CSRF(r)
	}
	return ""
}

// renderPage renders a page with Unpoly-awareness: uses admin layout for full
// page requests, sets Vary for caching correctness.
func (h *AdminHandler) renderPage(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	unpoly.Vary(w)
	if err := h.Renderer.Render(w, page, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// renderAdmin renders an admin page with the given active page name.
func (h *AdminHandler) renderAdmin(w http.ResponseWriter, r *http.Request, page string, activePage string, extras map[string]any) {
	data := h.adminData(r, activePage)
	for k, v := range extras {
		data[k] = v
	}
	h.renderPage(w, r, page, data)
}

// redirectAfterAction returns appropriate response for destructive actions.
// For Unpoly requests, it sets AcceptLayer to close the confirmation modal and
// emits a reload event. For regular requests, it redirects.
func (h *AdminHandler) redirectAfterAction(w http.ResponseWriter, r *http.Request, redirectURL string) {
	if unpoly.IsUnpolyRequest(r) {
		unpoly.AcceptLayer(w, "true")
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "up:reload", Target: "main"},
		})
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// parsePage parses the "page" query parameter, defaulting to 1.
// Clamped to [1, maxPage].
func parsePage(r *http.Request) int {
	p, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || p < 1 {
		return 1
	}
	const maxPage = 60
	if p > maxPage {
		return maxPage
	}
	return p
}

// ─── Login ───

// ShowLogin displays the admin login form.
func (h *AdminHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	unpoly.Vary(w)
	data := map[string]any{
		"CSRF": h.csrfToken(r),
	}
	if err := h.Renderer.Render(w, "login", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// Login authenticates an admin user.
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLoginError(w, r, "Données de formulaire invalides", "")
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.renderLoginError(w, r, "Email et mot de passe requis", email)
		return
	}

	user, token, err := h.AuthService.Login(r.Context(), email, password)
	if err != nil {
		h.renderLoginError(w, r, "Email ou mot de passe incorrect", email)
		return
	}

	if user.Role != "admin" {
		h.renderLoginError(w, r, "Accès réservé aux administrateurs", email)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})

	if unpoly.IsUnpolyRequest(r) {
		unpoly.SetLocation(w, "/admin", "GET")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) renderLoginError(w http.ResponseWriter, r *http.Request, errMsg, email string) {
	if unpoly.IsUnpolyRequest(r) {
		// For Unpoly validation, return 422 with error
		unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"email": errMsg})
		return
	}
	unpoly.Vary(w)
	data := map[string]any{
		"CSRF":  h.csrfToken(r),
		"Error": errMsg,
		"Email": email,
	}
	if err := h.Renderer.Render(w, "login", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// ─── Dashboard ───

// Dashboard displays the admin dashboard.
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	dash, err := h.AdminService.DashboardData(r.Context())
	if err != nil {
		http.Error(w, "dashboard error", http.StatusInternalServerError)
		return
	}

	extras := map[string]any{
		"Revenue":          dash.TotalRevenue,
		"OrderCount":       dash.OrderCount,
		"AverageOrder":     dash.AvgBasket,
		"VisitorCount":     dash.VisitorCount,
		"RecentOrders":     dash.RecentOrders,
		"BestSellers":      dash.BestSellers,
		"LowStockProducts": dash.LowStock,
	}
	h.renderAdmin(w, r, "dashboard", "dashboard", extras)
}

// ─── Orders ───

// Orders displays the orders list with optional status filter and pagination.
func (h *AdminHandler) Orders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page := parsePage(r)

	orders, total, err := h.AdminService.Orders(r.Context(), status, page)
	if err != nil {
		http.Error(w, "orders error", http.StatusInternalServerError)
		return
	}

	extras := map[string]any{
		"Orders":        orders,
		"Status":        status,
		"Page":          page,
		"Total":         total,
		"PerPage":       20,
		"TotalPages":    (total + 19) / 20,
		"OrderStatuses": []string{"pending", "confirmed", "shipped", "delivered", "cancelled"},
	}
	h.renderAdmin(w, r, "orders", "orders", extras)
}

// OrderDetail displays a single order.
func (h *AdminHandler) OrderDetail(w http.ResponseWriter, r *http.Request, orderID string) {
	// Orders method returns list by status; we fetch all and find by ID.
	// The AdminService doesn't have a ByID method for orders directly,
	// so we iterate. For a real implementation, you'd add GetOrder.
	orders, _, err := h.AdminService.Orders(r.Context(), "", 1)
	if err != nil {
		http.Error(w, "order error", http.StatusInternalServerError)
		return
	}

	var order *domain.Order
	for i := range orders {
		if orders[i].ID == orderID || orders[i].Ref == orderID {
			order = &orders[i]
			break
		}
	}

	if order == nil {
		http.NotFound(w, r)
		return
	}

	extras := map[string]any{
		"Order":          order,
		"OrderStatuses":  []string{"pending", "confirmed", "shipped", "delivered", "cancelled"},
	}
	h.renderAdmin(w, r, "order-detail", "orders", extras)
}

// UpdateOrderStatus updates an order's status via PATCH (Unpoly layer confirmation).
func (h *AdminHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request, orderID string) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if body.Status == "" {
		http.Error(w, "status required", http.StatusBadRequest)
		return
	}

	if err := h.AdminService.UpdateOrderStatus(r.Context(), orderID, body.Status); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	h.redirectAfterAction(w, r, "/admin/orders/"+orderID)
}

// ─── Products ───

// Products displays the product list.
func (h *AdminHandler) Products(w http.ResponseWriter, r *http.Request) {
	products, _, err := h.AdminService.Products(r.Context())
	if err != nil {
		http.Error(w, "products error", http.StatusInternalServerError)
		return
	}

	extras := map[string]any{
		"Products": products,
	}
	h.renderAdmin(w, r, "products", "products", extras)
}

// ProductEdit displays the product edit form.
func (h *AdminHandler) ProductEdit(w http.ResponseWriter, r *http.Request, productID string) {
	products, _, err := h.AdminService.Products(r.Context())
	if err != nil {
		http.Error(w, "products error", http.StatusInternalServerError)
		return
	}

	var product *domain.Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}

	if product == nil {
		http.NotFound(w, r)
		return
	}

	categories, _ := h.AdminService.Categories(r.Context())

	extras := map[string]any{
		"Product":    product,
		"Categories": categories,
	}
	h.renderAdmin(w, r, "product-edit", "products", extras)
}

// ProductCreate creates a new product.
func (h *AdminHandler) ProductCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		h.renderProductEditError(w, r, "", "Le nom est requis", nil)
		return
	}

	priceStr := r.FormValue("price")
	priceCents, err := parsePriceCents(priceStr)
	if err != nil {
		h.renderProductEditError(w, r, name, "Prix invalide", parseProductForm(r))
		return
	}

	stock, _ := strconv.Atoi(r.FormValue("stock"))
	if stock < 0 {
		stock = 0
	}

	comparePriceStr := r.FormValue("compare_price")
	var comparePrice *domain.Money
	if comparePriceStr != "" {
		cp, err := parsePriceCents(comparePriceStr)
		if err == nil {
			m := domain.Money(cp)
			comparePrice = &m
		}
	}

	description := strings.TrimSpace(r.FormValue("description"))
	imageURL := strings.TrimSpace(r.FormValue("image_url"))
	categoryID := strings.TrimSpace(r.FormValue("category_id"))

	slug := generateSlug(name)

	product := &domain.Product{
		ID:           id.NewV4(),
		Slug:         slug,
		CategoryID:   categoryID,
		Name:         name,
		Description:  description,
		Price:        domain.Money(priceCents),
		ComparePrice: comparePrice,
		Stock:        stock,
		Status:       "draft",
		ImageURL:     imageURL,
		CreatedAt:    time.Now(),
	}

	if err := h.AdminService.CreateProduct(r.Context(), product); err != nil {
		h.renderProductEditError(w, r, name, "Erreur lors de la création: "+err.Error(), parseProductForm(r))
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.AcceptLayer(w, "true")
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "up:reload", Target: "main"},
		})
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
}

// ProductUpdate updates an existing product.
func (h *AdminHandler) ProductUpdate(w http.ResponseWriter, r *http.Request, productID string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	products, _, err := h.AdminService.Products(r.Context())
	if err != nil {
		http.Error(w, "products error", http.StatusInternalServerError)
		return
	}

	var product *domain.Product
	for i := range products {
		if products[i].ID == productID {
			product = &products[i]
			break
		}
	}

	if product == nil {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name != "" {
		product.Name = name
		product.Slug = generateSlug(name)
	}

	if priceStr := r.FormValue("price"); priceStr != "" {
		priceCents, err := parsePriceCents(priceStr)
		if err == nil {
			product.Price = domain.Money(priceCents)
		}
	}

	if comparePriceStr := r.FormValue("compare_price"); comparePriceStr != "" {
		cp, err := parsePriceCents(comparePriceStr)
		if err == nil {
			m := domain.Money(cp)
			product.ComparePrice = &m
		}
	} else if r.Form.Has("compare_price") {
		product.ComparePrice = nil
	}

	if stockStr := r.FormValue("stock"); stockStr != "" {
		stock, _ := strconv.Atoi(stockStr)
		if stock >= 0 {
			product.Stock = stock
		}
	}

	if desc := r.FormValue("description"); r.Form.Has("description") {
		product.Description = strings.TrimSpace(desc)
	}

	if imgURL := r.FormValue("image_url"); r.Form.Has("image_url") {
		product.ImageURL = strings.TrimSpace(imgURL)
	}

	if catID := r.FormValue("category_id"); r.Form.Has("category_id") {
		product.CategoryID = strings.TrimSpace(catID)
	}

	if status := r.FormValue("status"); isValidStatus(status) {
		product.Status = status
	}

	if err := h.AdminService.UpdateProduct(r.Context(), product); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}

	if unpoly.IsUnpolyRequest(r) {
		unpoly.AcceptLayer(w, "true")
		unpoly.EmitEvents(w, []unpoly.Event{
			{Type: "up:reload", Target: "main"},
		})
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/admin/products/"+productID, http.StatusSeeOther)
}

// ProductDelete deletes a product.
func (h *AdminHandler) ProductDelete(w http.ResponseWriter, r *http.Request, productID string) {
	if err := h.AdminService.DeleteProduct(r.Context(), productID); err != nil {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.AcceptLayer(w, "false")
			unpoly.EmitEvents(w, []unpoly.Event{
				{Type: "up:notify", Options: map[string]any{"text": "Erreur suppression: " + err.Error()}},
			})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	h.redirectAfterAction(w, r, "/admin/products")
}

func (h *AdminHandler) renderProductEditError(w http.ResponseWriter, r *http.Request, name, errMsg string, form map[string]any) {
	categories, _ := h.AdminService.Categories(r.Context())

	extras := map[string]any{
		"Error":      errMsg,
		"Categories": categories,
		"Form":       form,
	}
	if form != nil {
		for k, v := range form {
			extras[k] = v
		}
	}
	h.renderAdmin(w, r, "product-edit", "products", extras)
}

func parseProductForm(r *http.Request) map[string]any {
	return map[string]any{
		"Name":         r.FormValue("name"),
		"Price":        r.FormValue("price"),
		"ComparePrice": r.FormValue("compare_price"),
		"Stock":        r.FormValue("stock"),
		"Description":  r.FormValue("description"),
		"ImageURL":     r.FormValue("image_url"),
		"CategoryID":   r.FormValue("category_id"),
		"Status":       r.FormValue("status"),
	}
}

// ─── Customers ───

// Customers displays the customer list with pagination.
func (h *AdminHandler) Customers(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)

	customers, total, err := h.AdminService.Customers(r.Context(), page)
	if err != nil {
		http.Error(w, "customers error", http.StatusInternalServerError)
		return
	}

	extras := map[string]any{
		"Customers":  customers,
		"Page":       page,
		"Total":      total,
		"PerPage":    20,
		"TotalPages": (total + 19) / 20,
	}
	h.renderAdmin(w, r, "customers", "customers", extras)
}

// ─── Discounts ───

// Discounts displays the discount list.
func (h *AdminHandler) Discounts(w http.ResponseWriter, r *http.Request) {
	discounts, err := h.AdminService.Discounts(r.Context())
	if err != nil {
		http.Error(w, "discounts error", http.StatusInternalServerError)
		return
	}

	extras := map[string]any{
		"Discounts": discounts,
	}
	h.renderAdmin(w, r, "discounts", "discounts", extras)
}

// DiscountCreate creates a new discount code.
func (h *AdminHandler) DiscountCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))
	if code == "" {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"code": "Code requis"})
			return
		}
		http.Error(w, "code required", http.StatusBadRequest)
		return
	}

	percent, err := strconv.Atoi(r.FormValue("percent"))
	if err != nil || percent < 1 || percent > 100 {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"percent": "Pourcentage invalide (1-100)"})
			return
		}
		http.Error(w, "invalid percent", http.StatusBadRequest)
		return
	}

	discount := &domain.Discount{
		ID:      id.NewV4(),
		Code:    code,
		Percent: percent,
		Active:  true,
	}

	// Parse optional expiration
	if expStr := r.FormValue("expires_at"); expStr != "" {
		if t, err := time.Parse("2006-01-02", expStr); err == nil {
			discount.ExpiresAt = &t
		}
	}

	if err := h.AdminService.CreateDiscount(r.Context(), discount); err != nil {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.AcceptLayer(w, "false")
			unpoly.EmitEvents(w, []unpoly.Event{
				{Type: "up:notify", Options: map[string]any{"text": "Erreur: " + err.Error()}},
			})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.Error(w, "create discount failed", http.StatusInternalServerError)
		return
	}

	h.redirectAfterAction(w, r, "/admin/discounts")
}

// DiscountDelete deletes a discount.
func (h *AdminHandler) DiscountDelete(w http.ResponseWriter, r *http.Request, discountID string) {
	if err := h.AdminService.DeleteDiscount(r.Context(), discountID); err != nil {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.AcceptLayer(w, "false")
			unpoly.EmitEvents(w, []unpoly.Event{
				{Type: "up:notify", Options: map[string]any{"text": "Erreur suppression: " + err.Error()}},
			})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	h.redirectAfterAction(w, r, "/admin/discounts")
}

// ─── Categories ───

// Categories displays the category list.
func (h *AdminHandler) Categories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.AdminService.Categories(r.Context())
	if err != nil {
		http.Error(w, "categories error", http.StatusInternalServerError)
		return
	}

	extras := map[string]any{
		"Categories": categories,
	}
	h.renderAdmin(w, r, "categories", "categories", extras)
}

// CategoryCreate creates a new category.
func (h *AdminHandler) CategoryCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.WriteValidationErrors(w, unpoly.ValidationErrors{"name": "Nom requis"})
			return
		}
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	slug := generateSlug(name)
	category := &domain.Category{
		ID:   id.NewV4(),
		Slug: slug,
		Name: name,
	}

	if err := h.AdminService.CreateCategory(r.Context(), category); err != nil {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.AcceptLayer(w, "false")
			unpoly.EmitEvents(w, []unpoly.Event{
				{Type: "up:notify", Options: map[string]any{"text": "Erreur: " + err.Error()}},
			})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.Error(w, "create category failed", http.StatusInternalServerError)
		return
	}

	h.redirectAfterAction(w, r, "/admin/categories")
}

// CategoryDelete deletes a category.
func (h *AdminHandler) CategoryDelete(w http.ResponseWriter, r *http.Request, categoryID string) {
	if err := h.AdminService.DeleteCategory(r.Context(), categoryID); err != nil {
		if unpoly.IsUnpolyRequest(r) {
			unpoly.AcceptLayer(w, "false")
			unpoly.EmitEvents(w, []unpoly.Event{
				{Type: "up:notify", Options: map[string]any{"text": "Erreur suppression: " + err.Error()}},
			})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	h.redirectAfterAction(w, r, "/admin/categories")
}

// ─── Settings ───

// Settings displays the admin settings page.
func (h *AdminHandler) Settings(w http.ResponseWriter, r *http.Request) {
	h.renderAdmin(w, r, "settings", "settings", map[string]any{})
}

// ─── Utilities ───

// validStatuses defines allowed product status values.
var validStatuses = map[string]bool{
	"draft":     true,
	"published": true,
	"archived":  true,
}

func isValidStatus(s string) bool {
	return validStatuses[s]
}

// parsePriceCents parses a price string like "19.99" or "42" into cents.
func parsePriceCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(f * 100), nil
}

// nonSlugRe matches characters that should be replaced with hyphens in a slug.
var nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// generateSlug converts a product/category name into a URL-safe slug.
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = nonSlugRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "item"
	}
	return slug
}
