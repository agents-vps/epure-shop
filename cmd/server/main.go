package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/handlers"
	"github.com/agents-vps/epure-shop/internal/adapters/httpserver/middleware"
	"github.com/agents-vps/epure-shop/internal/adapters/sqlite"
	tmpl "github.com/agents-vps/epure-shop/internal/adapters/template"
	"github.com/agents-vps/epure-shop/internal/core/ports"
	"github.com/agents-vps/epure-shop/internal/core/service"
)

func main() {
	dev := flag.Bool("dev", false, "Development mode")
	port := flag.Int("port", 8090, "HTTP port")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// ─── Database ───
	db, err := sqlite.Open("shop.db")
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// ─── Renderer ───
	renderer, err := tmpl.New()
	if err != nil {
		logger.Error("failed to initialize renderer", "error", err)
		os.Exit(1)
	}

	// ─── Infrastructure ───
	hasher := sqlite.NewBcryptHasher()

	// ─── Repositories ───
	products := sqlite.NewProductRepo(db)
	users := sqlite.NewUserRepo(db)
	carts := sqlite.NewCartRepo(db)
	orders := sqlite.NewOrderRepo(db)
	discounts := sqlite.NewDiscountRepo(db)
	categories := sqlite.NewCategoryRepo(db)
	sessions := sqlite.NewSessionStore(db)

	// ─── Services ───
	catalogSvc := service.NewCatalogService(products, categories)
	cartSvc := service.NewCartService(carts, products, discounts)
	authSvc := service.NewAuthService(users, sessions, hasher)
	checkoutSvc := service.NewCheckoutService(carts, orders, products)
	adminSvc := service.NewAdminService(products, orders, users, categories, discounts)

	// ─── Handlers ───
	shopH := handlers.NewShopHandler(catalogSvc, renderer)
	cartH := handlers.NewCartHandler(cartSvc, renderer)
	checkoutH := handlers.NewCheckoutHandler(checkoutSvc, cartSvc, renderer)
	authH := handlers.NewAuthHandler(authSvc, cartSvc, renderer, *dev)
	acctH := handlers.NewAccountHandler(checkoutSvc, renderer)
	adminH := &handlers.AdminHandler{
		AdminService: adminSvc,
		AuthService:  authSvc,
		Renderer:     renderer,
	}

	// ─── Router ───
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Shop
	mux.HandleFunc("GET /{$}", shopH.Home)
	mux.HandleFunc("GET /c/{slug}", shopH.Category)
	mux.HandleFunc("GET /p/{slug}", shopH.Product)
	mux.HandleFunc("GET /search", shopH.Search)
	mux.HandleFunc("GET /cart", cartH.View)
	mux.HandleFunc("POST /cart/items", cartH.Add)
	mux.HandleFunc("PATCH /cart/items/{id}", cartH.Update)
	mux.HandleFunc("DELETE /cart/items/{id}", cartH.Remove)
	mux.HandleFunc("POST /cart/discount", cartH.ApplyDiscount)
	mux.HandleFunc("GET /checkout", checkoutH.Show)
	mux.HandleFunc("POST /checkout", checkoutH.Submit)
	mux.HandleFunc("GET /orders/{ref}/confirmation", checkoutH.Confirmation)
	mux.HandleFunc("GET /login", authH.ShowLogin)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("GET /register", authH.ShowRegister)
	mux.HandleFunc("POST /register", authH.Register)
	mux.HandleFunc("POST /logout", authH.Logout)

	// Account (protected)
	mux.HandleFunc("GET /account", withAuth(acctH.Show))
	mux.HandleFunc("GET /account/orders", withAuth(acctH.Orders))

	// Admin login (public)
	mux.HandleFunc("GET /admin/login", adminH.ShowLogin)
	mux.HandleFunc("POST /admin/login", adminH.Login)

	// Admin routes (protected)
	mux.HandleFunc("GET /admin", withAdmin(adminH.Dashboard))
	mux.HandleFunc("GET /admin/orders", withAdmin(adminH.Orders))
	mux.HandleFunc("GET /admin/orders/{id}", withAdminID(adminH.OrderDetail))
	mux.HandleFunc("PATCH /admin/orders/{id}/status", withAdminID(adminH.UpdateOrderStatus))
	mux.HandleFunc("GET /admin/products", withAdmin(adminH.Products))
	mux.HandleFunc("GET /admin/products/{id}", withAdminID(adminH.ProductEdit))
	mux.HandleFunc("POST /admin/products", withAdmin(adminH.ProductCreate))
	mux.HandleFunc("PUT /admin/products/{id}", withAdminID(adminH.ProductUpdate))
	mux.HandleFunc("DELETE /admin/products/{id}", withAdminID(adminH.ProductDelete))
	mux.HandleFunc("GET /admin/customers", withAdmin(adminH.Customers))
	mux.HandleFunc("GET /admin/discounts", withAdmin(adminH.Discounts))
	mux.HandleFunc("POST /admin/discounts", withAdmin(adminH.DiscountCreate))
	mux.HandleFunc("DELETE /admin/discounts/{id}", withAdminID(adminH.DiscountDelete))
	mux.HandleFunc("GET /admin/categories", withAdmin(adminH.Categories))
	mux.HandleFunc("POST /admin/categories", withAdmin(adminH.CategoryCreate))
	mux.HandleFunc("DELETE /admin/categories/{id}", withAdminID(adminH.CategoryDelete))
	mux.HandleFunc("GET /admin/settings", withAdmin(adminH.Settings))

	// 404
	mux.HandleFunc("GET /", shopH.NotFound)

	// ─── Middleware stack ───
	var h http.Handler = mux
	h = middleware.CSRF()(h)
	h = middleware.NewSessionAuth(sessions, logger).Middleware(h)
	h = middleware.Recover(logger)(h)
	h = middleware.Logger(logger)(h)
	h = timeoutHandler(30 * time.Second)(h)

	// ─── Server ───
	addr := fmt.Sprintf(":%d", *port)
	logger.Info("starting server", "addr", addr, "dev", *dev)
	if err := http.ListenAndServe(addr, h); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// withAuth wraps a handler with RequireAuth middleware.
func withAuth(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	auth := middleware.RequireAuth()
	return auth(http.HandlerFunc(fn)).ServeHTTP
}

// withAdmin wraps a handler with RequireAdmin middleware.
func withAdmin(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	admin := middleware.RequireAdmin()
	return admin(http.HandlerFunc(fn)).ServeHTTP
}

// withAdminID wraps a handler that takes an ID parameter with RequireAdmin.
func withAdminID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	admin := middleware.RequireAdmin()
	return admin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		fn(w, r, id)
	})).ServeHTTP
}

func timeoutHandler(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "request timed out")
	}
}

// Ensure that we satisfy the Clock interface
var _ ports.Clock = (*realClock)(nil)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
