# Backend Go + Unpoly — Épure E-commerce

> **For Hermes:** Execute this plan phase by phase using subagent-driven-development. Each phase produces a compilable increment.

**Goal:** Go 1.27 backend for Épure e-commerce — hexagonal architecture, Unpoly progressive enhancement, SQLite, html/template, stdlib only (except sqlite driver + bcrypt).

**Tech Stack:** Go 1.27, `net/http`, `html/template`, `database/sql` + `modernc.org/sqlite`, `golang.org/x/crypto/bcrypt`, Unpoly 3.14.x (vendored in `web/static/vendor/`).

**Architecture:** Hexagonal (ports & adapters). Core → Ports → Adapters. Front-end under `web/`.

---

## Phase 1: core/domain + core/ports

### Task 1.1: Domain entities
- `internal/core/domain/product.go` — Product struct
- `internal/core/domain/user.go` — User struct
- `internal/core/domain/cart.go` — Cart, CartItem structs
- `internal/core/domain/order.go` — Order, OrderItem structs
- `internal/core/domain/discount.go` — Discount struct
- `internal/core/domain/category.go` — Category struct
- `internal/core/domain/money.go` — Money type (int64 centimes), `ParseMoney()`, `String()`

### Task 1.2: Port interfaces
- `internal/core/ports/repositories.go` — ProductRepository, UserRepository, CartRepository, OrderRepository, DiscountRepository interfaces
- `internal/core/ports/services.go` — PasswordHasher, Clock, IDGenerator, Renderer interfaces
- `internal/core/ports/sessions.go` — SessionStore interface

---

## Phase 2: adapters/sqlite

### Task 2.1: Schema + migrations
- `migrations/001_init.sql` — Full schema
- `seed.sql` — 1 admin, 8 categories, 40 products

### Task 2.2: SQLite repositories
- `internal/adapters/sqlite/db.go` — Open(), migration runner
- `internal/adapters/sqlite/product_repo.go`
- `internal/adapters/sqlite/user_repo.go`
- `internal/adapters/sqlite/cart_repo.go`
- `internal/adapters/sqlite/order_repo.go`
- `internal/adapters/sqlite/discount_repo.go`
- `internal/adapters/sqlite/session_store.go`

---

## Phase 3: core/service

### Task 3.1: Business services (tested with fake implementations)
- `internal/core/service/catalog.go` — CatalogService
- `internal/core/service/cart.go` — CartService
- `internal/core/service/checkout.go` — CheckoutService
- `internal/core/service/auth.go` — AuthService
- `internal/core/service/admin.go` — AdminCatalogService, AdminOrderService

---

## Phase 4: adapters/template

### Task 4.1: Convert static HTML to Go templates
- Move `shop/` front-end files to `web/templates/` and `web/static/`
- Rename `.html` → `.gohtml`
- Extract layouts: `web/templates/layouts/base.gohtml`, `admin.gohtml`, `checkout.gohtml`
- Create partials: `web/templates/partials/product-grid.gohtml`, `cart-item.gohtml`, etc.
- Add `{{.CSRF}}` meta tag to base layout
- Add Unpoly vendor files to `web/static/vendor/`

### Task 4.2: Renderer
- `internal/adapters/template/renderer.go` — implements Renderer port
- `Render(w, req, status, page, data)` and `RenderPartial(w, req, status, partial, data)`
- Buffer-before-write pattern
- `IsUnpolyRequest(r)` → check `X-Up-Version` header
- FuncMap: `money`, `date`, `plural`, `seq`, `dict`
- Clone() pattern for page isolation

---

## Phase 5: adapters/httpserver/unpoly

### Task 5.1: Unpoly protocol package
- `internal/adapters/httpserver/unpoly/unpoly.go`
- Constants: all `X-Up-*` header names
- `IsUnpolyRequest(r) bool` — checks `X-Up-Version`
- `IsValidationRequest(r) bool` → `X-Up-Validate != ""`
- `ValidationField(r) string` → parses `X-Up-Validate` header
- `Target(r) string` → reads `X-Up-Target`
- `SetTarget(w, selector)` → sets response `X-Up-Target`
- `SetLocation(w, url, method)` → sets `X-Up-Location` + `X-Up-Method`
- `AcceptLayer(w, value)` → sets `X-Up-Accept-Layer`
- `DismissLayer(w)` → sets `X-Up-Dismiss-Layer: null`
- `EmitEvents(w, events)` → marshals JSON to `X-Up-Events`
- Middleware: `VaryHeader` → adds `Vary: X-Up-Version, X-Up-Target` to responses

---

## Phase 6: adapters/httpserver

### Task 6.1: Middleware stack
- `internal/adapters/httpserver/middleware/request_id.go`
- `internal/adapters/httpserver/middleware/recover.go`
- `internal/adapters/httpserver/middleware/logger.go` (slog)
- `internal/adapters/httpserver/middleware/session.go` (SessionAuth)
- `internal/adapters/httpserver/middleware/csrf.go` (http.CrossOriginProtection)
- `internal/adapters/httpserver/middleware/rate_limit.go`
- `internal/adapters/httpserver/middleware/auth.go` (RequireAuth, RequireAdmin)

### Task 6.2: Shop handlers
- `internal/adapters/httpserver/handlers/shop.go` — Home, Category, Product
- `internal/adapters/httpserver/handlers/cart.go` — Cart, AddToCart, UpdateCart, RemoveCart, ApplyDiscount
- `internal/adapters/httpserver/handlers/checkout.go` — Checkout GET/POST, confirmation
- `internal/adapters/httpserver/handlers/auth.go` — Login, Register, Logout
- `internal/adapters/httpserver/handlers/account.go` — Account, Orders

### Task 6.3: Admin handlers
- `internal/adapters/httpserver/handlers/admin_auth.go` — Admin login
- `internal/adapters/httpserver/handlers/admin_dashboard.go`
- `internal/adapters/httpserver/handlers/admin_orders.go` — List, detail, status update
- `internal/adapters/httpserver/handlers/admin_products.go` — CRUD
- `internal/adapters/httpserver/handlers/admin_customers.go`
- `internal/adapters/httpserver/handlers/admin_discounts.go`
- `internal/adapters/httpserver/handlers/admin_settings.go`

### Task 6.4: Router
- `internal/adapters/httpserver/router.go` — All routes per CDC §8

---

## Phase 7: cmd/server/main.go

- Wire all dependencies
- `go build` success
- `go run ./cmd/server -dev` serves the full site

---

## Phase 8: Security + Edge Cases + Audit

- §6 security checklist
- §7 all 22 edge cases
- go vet + go build clean
- HTML template harness smoke test
- Unpoly fallback: all forms work without JS
