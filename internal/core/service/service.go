// Package service implements business logic against port interfaces.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agents-vps/epure-shop/internal/core/domain"
	"github.com/agents-vps/epure-shop/internal/core/ports"
)

// ─── Catalog ───

type CatalogService struct {
	products  ports.ProductRepository
	categories ports.CategoryRepository
}

func NewCatalogService(products ports.ProductRepository, categories ports.CategoryRepository) *CatalogService {
	return &CatalogService{products: products, categories: categories}
}

func (s *CatalogService) HomeData(ctx context.Context) (*HomeData, error) {
	cats, err := s.categories.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog: home categories: %w", err)
	}
	newProducts, _, err := s.products.List(ctx, "", "newest", 0, 8)
	if err != nil {
		return nil, fmt.Errorf("catalog: home new products: %w", err)
	}
	bestProducts, _, err := s.products.List(ctx, "", "price-desc", 0, 8)
	if err != nil {
		return nil, fmt.Errorf("catalog: home best products: %w", err)
	}
	return &HomeData{
		Categories:      cats,
		NewProducts:     newProducts,
		BestProducts:    bestProducts,
	}, nil
}

type HomeData struct {
	Categories   []domain.Category
	NewProducts  []domain.Product
	BestProducts []domain.Product
}

func (s *CatalogService) CategoryData(ctx context.Context, slug string, sort string, page int) (*CategoryData, error) {
	cat, err := s.categories.BySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("catalog: category %q: %w", slug, err)
	}
	if cat == nil {
		return nil, ErrNotFound
	}
	const perPage = 12
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}
	products, total, err := s.products.List(ctx, slug, sort, offset, perPage)
	if err != nil {
		return nil, fmt.Errorf("catalog: list %q: %w", slug, err)
	}
	return &CategoryData{
		Category:  *cat,
		Products:  products,
		Total:     total,
		Page:      page,
		PerPage:   perPage,
		Sort:      sort,
	}, nil
}

type CategoryData struct {
	Category  domain.Category
	Products  []domain.Product
	Total     int
	Page      int
	PerPage   int
	Sort      string
}

func (d CategoryData) TotalPages() int {
	if d.PerPage <= 0 {
		return 1
	}
	return (d.Total + d.PerPage - 1) / d.PerPage
}

func (s *CatalogService) ProductData(ctx context.Context, slug string) (*ProductData, error) {
	p, err := s.products.BySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("catalog: product %q: %w", slug, err)
	}
	if p == nil || p.Status != "published" {
		return nil, ErrNotFound
	}
	similar, _, err := s.products.List(ctx, p.CategoryID, "newest", 0, 4)
	if err != nil {
		return nil, fmt.Errorf("catalog: similar products: %w", err)
	}
	return &ProductData{
		Product: *p,
		Similar: similar,
	}, nil
}

type ProductData struct {
	Product domain.Product
	Similar []domain.Product
}

func (s *CatalogService) Search(ctx context.Context, query string, page int) (*SearchData, error) {
	query = strings.TrimSpace(query)
	if len(query) > 200 {
		query = query[:200]
	}
	const perPage = 12
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}
	products, total, err := s.products.Search(ctx, query, offset, perPage)
	if err != nil {
		return nil, fmt.Errorf("catalog: search %q: %w", query, err)
	}
	return &SearchData{
		Query:    query,
		Products: products,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
	}, nil
}

type SearchData struct {
	Query    string
	Products []domain.Product
	Total    int
	Page     int
	PerPage  int
}

// ─── Cart ───

type CartService struct {
	cartRepo     ports.CartRepository
	productRepo  ports.ProductRepository
	discountRepo ports.DiscountRepository
}

func NewCartService(cr ports.CartRepository, pr ports.ProductRepository, dr ports.DiscountRepository) *CartService {
	return &CartService{cartRepo: cr, productRepo: pr, discountRepo: dr}
}

func (s *CartService) GetCart(ctx context.Context, token string) (*domain.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("cart: get: %w", err)
	}
	if cart == nil {
		cart = &domain.Cart{ID: token}
	}
	return cart, nil
}

func (s *CartService) AddItem(ctx context.Context, token string, productID string, qty int) (*domain.Cart, error) {
	p, err := s.productRepo.ByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("cart: product lookup: %w", err)
	}
	if p == nil || p.Status != "published" {
		return nil, ErrNotFound
	}
	if qty < 1 {
		qty = 1
	}
	// Check existing cart to cap at stock
	cart, err := s.cartRepo.Get(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("cart: get: %w", err)
	}
	if cart != nil {
		for _, item := range cart.Items {
			if item.ProductID == productID {
				newQty := item.Quantity + qty
				if newQty > p.Stock {
					newQty = p.Stock
				}
				if err := s.cartRepo.UpdateQuantity(ctx, token, productID, newQty); err != nil {
					return nil, fmt.Errorf("cart: update qty: %w", err)
				}
				cart, _ = s.cartRepo.Get(ctx, token)
				return cart, nil
			}
		}
	}
	if qty > p.Stock {
		qty = p.Stock
	}
	if err := s.cartRepo.AddItem(ctx, token, productID, qty); err != nil {
		return nil, fmt.Errorf("cart: add item: %w", err)
	}
	cart, _ = s.cartRepo.Get(ctx, token)
	return cart, nil
}

func (s *CartService) UpdateQuantity(ctx context.Context, token string, productID string, qty int) (*domain.Cart, error) {
	if qty < 1 {
		return s.RemoveItem(ctx, token, productID)
	}
	p, err := s.productRepo.ByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("cart: product lookup: %w", err)
	}
	if p != nil && qty > p.Stock {
		qty = p.Stock
	}
	if err := s.cartRepo.UpdateQuantity(ctx, token, productID, qty); err != nil {
		return nil, fmt.Errorf("cart: update qty: %w", err)
	}
	cart, _ := s.cartRepo.Get(ctx, token)
	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, token string, productID string) (*domain.Cart, error) {
	if err := s.cartRepo.RemoveItem(ctx, token, productID); err != nil {
		return nil, fmt.Errorf("cart: remove item: %w", err)
	}
	cart, _ := s.cartRepo.Get(ctx, token)
	return cart, nil
}

func (s *CartService) ApplyDiscount(ctx context.Context, token string, code string) (*domain.Cart, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	d, err := s.discountRepo.ByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("cart: discount lookup: %w", err)
	}
	if d == nil || !d.Active {
		return nil, ErrInvalidDiscount
	}
	cart, err := s.cartRepo.Get(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("cart: get: %w", err)
	}
	if cart == nil {
		return nil, ErrNotFound
	}
	cart.Discount = d
	if err := s.cartRepo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("cart: save discount: %w", err)
	}
	return cart, nil
}

func (s *CartService) MergeGuestCart(ctx context.Context, guestToken string, userID string) error {
	return s.cartRepo.MergeGuestCart(ctx, guestToken, userID)
}

// ─── Auth ───

type AuthService struct {
	users    ports.UserRepository
	sessions ports.SessionStore
	hasher   ports.PasswordHasher
}

func NewAuthService(users ports.UserRepository, sessions ports.SessionStore, hasher ports.PasswordHasher) *AuthService {
	return &AuthService{users: users, sessions: sessions, hasher: hasher}
}

func (s *AuthService) Register(ctx context.Context, email, name, password string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 {
		return nil, ErrValidation
	}
	existing, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("auth: check existing: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}
	user := &domain.User{
		Email:        email,
		Name:         strings.TrimSpace(name),
		PasswordHash: hash,
		Role:         "customer",
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("auth: create user: %w", err)
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	// Anti-timing: always hash something
	dummyHash, _ := s.hasher.Hash("dummy-constant-password-for-timing")

	user, err := s.users.ByEmail(ctx, email)
	if err != nil {
		s.hasher.Verify(password, dummyHash) // burn CPU
		return nil, "", fmt.Errorf("auth: user lookup: %w", err)
	}
	if user == nil {
		s.hasher.Verify(password, dummyHash) // burn CPU
		return nil, "", ErrInvalidCredentials
	}
	if !s.hasher.Verify(password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}
	token, err := s.sessions.Create(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("auth: create session: %w", err)
	}
	return user, token, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// ─── Checkout ───

type CheckoutService struct {
	cartRepo  ports.CartRepository
	orderRepo ports.OrderRepository
	productRepo ports.ProductRepository
}

func NewCheckoutService(cr ports.CartRepository, or ports.OrderRepository, pr ports.ProductRepository) *CheckoutService {
	return &CheckoutService{cartRepo: cr, orderRepo: or, productRepo: pr}
}

func (s *CheckoutService) ValidateCheckout(ctx context.Context, token string) (*domain.Cart, error) {
	cart, err := s.cartRepo.Get(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("checkout: get cart: %w", err)
	}
	if cart == nil || len(cart.Items) == 0 {
		return nil, ErrEmptyCart
	}
	// Verify all products still published
	for _, item := range cart.Items {
		if item.Product == nil || item.Product.Status != "published" {
			return nil, fmt.Errorf("%w: product %s unavailable", ErrValidation, item.ProductID)
		}
	}
	return cart, nil
}

func (s *CheckoutService) PlaceOrder(ctx context.Context, token string, email string, shippingAddress string, idempotencyKey string) (*domain.Order, error) {
	cart, err := s.ValidateCheckout(ctx, token)
	if err != nil {
		return nil, err
	}

	// Reload product prices from DB
	productIDs := make([]string, len(cart.Items))
	for i, item := range cart.Items {
		productIDs[i] = item.ProductID
	}
	products, err := s.productRepo.ListByIDs(ctx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("checkout: load products: %w", err)
	}
	priceMap := make(map[string]domain.Product)
	for _, p := range products {
		priceMap[p.ID] = p
	}

	var items []domain.OrderItem
	var subtotal int64
	for _, item := range cart.Items {
		p, ok := priceMap[item.ProductID]
		if !ok || p.Status != "published" {
			return nil, fmt.Errorf("%w: product %s unavailable", ErrValidation, item.ProductID)
		}
		unitPrice := p.Price
		items = append(items, domain.OrderItem{
			ProductID: item.ProductID,
			Name:      p.Name,
			UnitPrice: unitPrice,
			Quantity:  item.Quantity,
			ImageURL:  p.ImageURL,
		})
		subtotal += unitPrice.Cents() * int64(item.Quantity)
	}

	// Shipping: free above 75 EUR, otherwise 4.90 EUR
	var shipping int64 = 490
	if subtotal >= 7500 {
		shipping = 0
	}

	// Discount
	var discountAmount int64
	if cart.Discount != nil {
		discountAmount = subtotal * int64(cart.Discount.Percent) / 100
	}

	total := subtotal + shipping - discountAmount
	if total < 0 {
		total = 0
	}

	order := &domain.Order{
		Email:           email,
		Status:          "pending",
		Items:           items,
		Subtotal:        domain.Money(subtotal),
		ShippingCost:    domain.Money(shipping),
		DiscountAmount:  domain.Money(discountAmount),
		Total:           domain.Money(total),
		ShippingAddress: shippingAddress,
	}

	if err := s.orderRepo.Create(ctx, order, idempotencyKey); err != nil {
		return nil, fmt.Errorf("checkout: create order: %w", err)
	}

	// Clear cart
	if err := s.cartRepo.Delete(ctx, token); err != nil {
		return nil, fmt.Errorf("checkout: clear cart: %w", err)
	}

	return order, nil
}

func (s *CheckoutService) GetOrder(ctx context.Context, ref string) (*domain.Order, error) {
	order, err := s.orderRepo.ByRef(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("checkout: get order %q: %w", ref, err)
	}
	if order == nil {
		return nil, ErrNotFound
	}
	return order, nil
}

type AdminService struct {
	products   ports.ProductRepository
	orders     ports.OrderRepository
	users      ports.UserRepository
	categories ports.CategoryRepository
	discounts  ports.DiscountRepository
}

func NewAdminService(pr ports.ProductRepository, or ports.OrderRepository, ur ports.UserRepository, cr ports.CategoryRepository, dr ports.DiscountRepository) *AdminService {
	return &AdminService{products: pr, orders: or, users: ur, categories: cr, discounts: dr}
}

func (s *AdminService) DashboardData(ctx context.Context) (*AdminDashboard, error) {
	allOrders, _, _ := s.orders.List(ctx, "", 0, 1000)
	products, _, _ := s.products.List(ctx, "", "", 0, 1000)
	recentOrders, _, _ := s.orders.List(ctx, "", 0, 5)

	// Low stock products
	var lowStock []domain.Product
	for _, p := range products {
		if p.Stock < 5 && p.Status == "published" {
			lowStock = append(lowStock, p)
		}
	}

	// Best sellers placeholder
	bestSellers, _, _ := s.products.List(ctx, "", "newest", 0, 5)

	return &AdminDashboard{
		TotalRevenue:    domain.Money(125000), // placeholder
		OrderCount:      len(allOrders),
		AvgBasket:       domain.Money(4500),
		VisitorCount:    342,
		RecentOrders:    recentOrders,
		LowStock:        lowStock,
		BestSellers:     bestSellers,
	}, nil
}

type AdminDashboard struct {
	TotalRevenue    domain.Money
	OrderCount      int
	AvgBasket       domain.Money
	VisitorCount    int
	RecentOrders    []domain.Order
	LowStock        []domain.Product
	BestSellers     []domain.Product
}

func (s *AdminService) Orders(ctx context.Context, status string, page int) ([]domain.Order, int, error) {
	const perPage = 20
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}
	return s.orders.List(ctx, status, offset, perPage)
}

func (s *AdminService) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	valid := map[string]bool{"pending": true, "confirmed": true, "shipped": true, "delivered": true, "cancelled": true}
	if !valid[status] {
		return ErrValidation
	}
	return s.orders.UpdateStatus(ctx, orderID, status)
}

func (s *AdminService) Products(ctx context.Context) ([]domain.Product, int, error) {
	return s.products.List(ctx, "", "", 0, 100)
}

func (s *AdminService) CreateProduct(ctx context.Context, p *domain.Product) error {
	return s.products.Create(ctx, p)
}

func (s *AdminService) UpdateProduct(ctx context.Context, p *domain.Product) error {
	return s.products.Update(ctx, p)
}

func (s *AdminService) DeleteProduct(ctx context.Context, id string) error {
	return s.products.Delete(ctx, id)
}

func (s *AdminService) Customers(ctx context.Context, page int) ([]domain.User, int, error) {
	const perPage = 20
	offset := (page - 1) * perPage
	if offset < 0 {
		offset = 0
	}
	return s.users.List(ctx, offset, perPage)
}

func (s *AdminService) Discounts(ctx context.Context) ([]domain.Discount, error) {
	return s.discounts.List(ctx)
}

func (s *AdminService) CreateDiscount(ctx context.Context, d *domain.Discount) error {
	return s.discounts.Create(ctx, d)
}

func (s *AdminService) DeleteDiscount(ctx context.Context, id string) error {
	return s.discounts.Delete(ctx, id)
}

func (s *AdminService) Categories(ctx context.Context) ([]domain.Category, error) {
	return s.categories.List(ctx)
}

func (s *AdminService) CreateCategory(ctx context.Context, c *domain.Category) error {
	return s.categories.Create(ctx, c)
}

func (s *AdminService) DeleteCategory(ctx context.Context, id string) error {
	return s.categories.Delete(ctx, id)
}

// ─── Errors ───

var (
	ErrNotFound          = errors.New("not found")
	ErrValidation        = errors.New("validation error")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken        = errors.New("email already registered")
	ErrEmptyCart         = errors.New("cart is empty")
	ErrInvalidDiscount   = errors.New("invalid or expired discount code")
)
