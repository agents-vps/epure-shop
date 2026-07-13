package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/agents-vps/epure-shop/internal/core/domain"
	"github.com/agents-vps/epure-shop/internal/core/domain/id"
	"github.com/agents-vps/epure-shop/internal/core/ports"
)

// ─── helpers ────────────────────────────────────────────────────────────

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func nullInt64Ptr(m *domain.Money) interface{} {
	if m == nil {
		return nil
	}
	return m.Cents()
}

// ─── ProductRepository ──────────────────────────────────────────────────

type productRepo struct{ db *sql.DB }

func NewProductRepo(db *sql.DB) ports.ProductRepository { return &productRepo{db} }

var productSorts = map[string]string{
	"price":       "p.price",
	"-price":      "p.price DESC",
	"name":        "p.name",
	"-name":       "p.name DESC",
	"rating":      "p.rating",
	"-rating":     "p.rating DESC",
	"created_at":  "p.created_at",
	"-created_at": "p.created_at DESC",
	"stock":       "p.stock",
	"-stock":      "p.stock DESC",
}

const productCols = `p.id, p.slug, p.category_id, c.name, p.name, p.description,
	p.price, p.compare_price, p.stock, p.status, p.image_url,
	p.rating, p.review_count, p.created_at`

func scanProduct(scanner interface{ Scan(...interface{}) error }) (domain.Product, error) {
	var p domain.Product
	var catName string
	var comparePrice sql.NullInt64
	var createdAt string
	if err := scanner.Scan(
		&p.ID, &p.Slug, &p.CategoryID, &catName, &p.Name, &p.Description,
		(*int64)(&p.Price), &comparePrice, &p.Stock, &p.Status, &p.ImageURL,
		&p.Rating, &p.ReviewCount, &createdAt,
	); err != nil {
		return p, err
	}
	p.CategoryName = catName
	if comparePrice.Valid {
		cp := domain.Money(comparePrice.Int64)
		p.ComparePrice = &cp
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return p, nil
}

func (r *productRepo) List(ctx context.Context, categorySlug string, sort string, offset, limit int) ([]domain.Product, int, error) {
	orderBy, ok := productSorts[sort]
	if !ok {
		orderBy = "p.created_at DESC"
	}

	var countQuery string
	var countArgs []interface{}
	if categorySlug != "" {
		countQuery = `SELECT COUNT(*) FROM products p JOIN categories c ON p.category_id=c.id WHERE c.slug=? AND p.status='published'`
		countArgs = []interface{}{categorySlug}
	} else {
		countQuery = `SELECT COUNT(*) FROM products WHERE status='published'`
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("productRepo.List count: %w", err)
	}

	var listQuery string
	var listArgs []interface{}
	if categorySlug != "" {
		listQuery = fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id WHERE c.slug=? AND p.status='published' ORDER BY %s LIMIT ? OFFSET ?`, productCols, orderBy)
		listArgs = []interface{}{categorySlug, limit, offset}
	} else {
		listQuery = fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id WHERE p.status='published' ORDER BY %s LIMIT ? OFFSET ?`, productCols, orderBy)
		listArgs = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("productRepo.List query: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("productRepo.List scan: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *productRepo) BySlug(ctx context.Context, slug string) (*domain.Product, error) {
	query := fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id WHERE p.slug=?`, productCols)
	p, err := scanProduct(r.db.QueryRowContext(ctx, query, slug))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("productRepo.BySlug: %w", err)
	}
	return &p, nil
}

func (r *productRepo) ByID(ctx context.Context, productID string) (*domain.Product, error) {
	query := fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id WHERE p.id=?`, productCols)
	p, err := scanProduct(r.db.QueryRowContext(ctx, query, productID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("productRepo.ByID: %w", err)
	}
	return &p, nil
}

func (r *productRepo) Create(ctx context.Context, p *domain.Product) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO products (id, slug, category_id, name, description, price, compare_price, stock, status, image_url, rating, review_count, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Slug, p.CategoryID, p.Name, p.Description, p.Price.Cents(),
		nullInt64Ptr(p.ComparePrice), p.Stock, p.Status, p.ImageURL,
		p.Rating, p.ReviewCount, p.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("productRepo.Create: %w", err)
	}
	return nil
}

func (r *productRepo) Update(ctx context.Context, p *domain.Product) error {
	_, err := r.db.ExecContext(ctx, `UPDATE products SET slug=?, category_id=?, name=?, description=?, price=?, compare_price=?, stock=?, status=?, image_url=?, rating=?, review_count=? WHERE id=?`,
		p.Slug, p.CategoryID, p.Name, p.Description, p.Price.Cents(),
		nullInt64Ptr(p.ComparePrice), p.Stock, p.Status, p.ImageURL,
		p.Rating, p.ReviewCount, p.ID,
	)
	if err != nil {
		return fmt.Errorf("productRepo.Update: %w", err)
	}
	return nil
}

func (r *productRepo) Delete(ctx context.Context, productID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM products WHERE id=?`, productID)
	if err != nil {
		return fmt.Errorf("productRepo.Delete: %w", err)
	}
	return nil
}

func (r *productRepo) Search(ctx context.Context, query string, offset, limit int) ([]domain.Product, int, error) {
	like := "%" + query + "%"

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM products p JOIN categories c ON p.category_id=c.id
		 WHERE p.status='published' AND (p.name LIKE ? OR p.description LIKE ?)`,
		like, like,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("productRepo.Search count: %w", err)
	}

	listQuery := fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id
		WHERE p.status='published' AND (p.name LIKE ? OR p.description LIKE ?)
		ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, productCols)
	rows, err := r.db.QueryContext(ctx, listQuery, like, like, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("productRepo.Search query: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("productRepo.Search scan: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *productRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id WHERE p.id IN (%s)`,
		productCols, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("productRepo.ListByIDs: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("productRepo.ListByIDs scan: %w", err)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// ─── UserRepository ─────────────────────────────────────────────────────

type userRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) ports.UserRepository { return &userRepo{db} }

func scanUser(scanner interface{ Scan(...interface{}) error }) (domain.User, error) {
	var u domain.User
	var createdAt string
	if err := scanner.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &createdAt); err != nil {
		return u, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return u, nil
}

func (r *userRepo) ByID(ctx context.Context, userID string) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE id=?`, userID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.ByID: %w", err)
	}
	return &u, nil
}

func (r *userRepo) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at FROM users WHERE email=?`, email))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("userRepo.ByEmail: %w", err)
	}
	return &u, nil
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, created_at) VALUES (?,?,?,?,?,?)`,
		u.ID, u.Email, u.Name, u.PasswordHash, u.Role, u.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) List(ctx context.Context, offset, limit int) ([]domain.User, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("userRepo.List count: %w", err)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, name, password_hash, role, created_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("userRepo.List: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("userRepo.List scan: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// ─── CartRepository ─────────────────────────────────────────────────────

type cartRepo struct{ db *sql.DB }

func NewCartRepo(db *sql.DB) ports.CartRepository { return &cartRepo{db} }

func (r *cartRepo) Get(ctx context.Context, token string) (*domain.Cart, error) {
	cart := &domain.Cart{}
	var userID sql.NullString
	var discountID sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, discount_id, created_at, updated_at FROM carts WHERE id=?`, token,
	).Scan(&cart.ID, &userID, &discountID, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("cartRepo.Get: %w", err)
	}
	if userID.Valid {
		cart.UserID = &userID.String
	}
	cart.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	cart.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	// Load items (product_id, quantity).
	itemRows, err := r.db.QueryContext(ctx,
		`SELECT ci.product_id, ci.quantity FROM cart_items ci WHERE ci.cart_id=? ORDER BY ci.product_id`, token)
	if err != nil {
		return nil, fmt.Errorf("cartRepo.Get items: %w", err)
	}
	defer itemRows.Close()

	var productIDs []string
	itemMap := make(map[string]int)
	for itemRows.Next() {
		var pid string
		var qty int
		if err := itemRows.Scan(&pid, &qty); err != nil {
			return nil, fmt.Errorf("cartRepo.Get scan item: %w", err)
		}
		productIDs = append(productIDs, pid)
		itemMap[pid] = qty
	}
	itemRows.Close()

	// Load full product data.
	if len(productIDs) > 0 {
		placeholders := make([]string, len(productIDs))
		args := make([]interface{}, len(productIDs))
		for i, pid := range productIDs {
			placeholders[i] = "?"
			args[i] = pid
		}
		query := fmt.Sprintf(`SELECT %s FROM products p JOIN categories c ON p.category_id=c.id WHERE p.id IN (%s)`,
			productCols, strings.Join(placeholders, ","))
		pRows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("cartRepo.Get products: %w", err)
		}
		defer pRows.Close()

		prodMap := make(map[string]domain.Product)
		for pRows.Next() {
			p, err := scanProduct(pRows)
			if err != nil {
				return nil, fmt.Errorf("cartRepo.Get scan product: %w", err)
			}
			prodMap[p.ID] = p
		}
		if err := pRows.Err(); err != nil {
			return nil, fmt.Errorf("cartRepo.Get products iter: %w", err)
		}

		for _, pid := range productIDs {
			p := prodMap[pid]
			cart.Items = append(cart.Items, domain.CartItem{
				ProductID: pid,
				Product:   &p,
				Quantity:  itemMap[pid],
			})
		}
	}

	// Load discount.
	if discountID.Valid {
		d, err := r.loadDiscount(ctx, discountID.String)
		if err != nil {
			return nil, fmt.Errorf("cartRepo.Get discount: %w", err)
		}
		cart.Discount = d
	}

	return cart, nil
}

func (r *cartRepo) loadDiscount(ctx context.Context, discountID string) (*domain.Discount, error) {
	var d domain.Discount
	var active int
	var expiresAt sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, code, percent, active, expires_at FROM discounts WHERE id=?`, discountID,
	).Scan(&d.ID, &d.Code, &d.Percent, &active, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("loadDiscount: %w", err)
	}
	d.Active = active == 1
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		d.ExpiresAt = &t
	}
	return &d, nil
}

func (r *cartRepo) Save(ctx context.Context, cart *domain.Cart) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cartRepo.Save begin: %w", err)
	}
	defer tx.Rollback()

	var discountID interface{}
	if cart.Discount != nil {
		discountID = cart.Discount.ID
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO carts (id, user_id, discount_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET user_id=excluded.user_id, discount_id=excluded.discount_id, updated_at=excluded.updated_at`,
		cart.ID, cart.UserID, discountID,
		cart.CreatedAt.Format(time.RFC3339), cart.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("cartRepo.Save upsert cart: %w", err)
	}

	// Replace items.
	if _, err := tx.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id=?`, cart.ID); err != nil {
		return fmt.Errorf("cartRepo.Save delete items: %w", err)
	}
	for _, item := range cart.Items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO cart_items (cart_id, product_id, quantity) VALUES (?, ?, ?)`,
			cart.ID, item.ProductID, item.Quantity,
		); err != nil {
			return fmt.Errorf("cartRepo.Save insert item: %w", err)
		}
	}

	return tx.Commit()
}

func (r *cartRepo) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM carts WHERE id=?`, token)
	if err != nil {
		return fmt.Errorf("cartRepo.Delete: %w", err)
	}
	return nil
}

func (r *cartRepo) AddItem(ctx context.Context, token string, productID string, qty int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity) VALUES (?, ?, ?)
		 ON CONFLICT(cart_id, product_id) DO UPDATE SET quantity = quantity + excluded.quantity`,
		token, productID, qty,
	)
	if err != nil {
		return fmt.Errorf("cartRepo.AddItem: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.db.ExecContext(ctx, `UPDATE carts SET updated_at=? WHERE id=?`, now, token)
	return nil
}

func (r *cartRepo) UpdateQuantity(ctx context.Context, token string, productID string, qty int) error {
	if qty <= 0 {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id=? AND product_id=?`, token, productID); err != nil {
			return fmt.Errorf("cartRepo.UpdateQuantity delete: %w", err)
		}
	} else {
		if _, err := r.db.ExecContext(ctx,
			`UPDATE cart_items SET quantity=? WHERE cart_id=? AND product_id=?`,
			qty, token, productID,
		); err != nil {
			return fmt.Errorf("cartRepo.UpdateQuantity update: %w", err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.db.ExecContext(ctx, `UPDATE carts SET updated_at=? WHERE id=?`, now, token)
	return nil
}

func (r *cartRepo) RemoveItem(ctx context.Context, token string, productID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM cart_items WHERE cart_id=? AND product_id=?`, token, productID); err != nil {
		return fmt.Errorf("cartRepo.RemoveItem: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.db.ExecContext(ctx, `UPDATE carts SET updated_at=? WHERE id=?`, now, token)
	return nil
}

func (r *cartRepo) MergeGuestCart(ctx context.Context, guestToken string, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cartRepo.MergeGuestCart begin: %w", err)
	}
	defer tx.Rollback()

	// Get or create user cart.
	var userCartID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM carts WHERE user_id=?`, userID).Scan(&userCartID)
	if err == sql.ErrNoRows {
		userCartID = id.NewV4()
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO carts (id, user_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
			userCartID, userID, now, now,
		); err != nil {
			return fmt.Errorf("cartRepo.MergeGuestCart create: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("cartRepo.MergeGuestCart find: %w", err)
	}

	// Merge items from guest into user cart.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity)
		 SELECT ?, ci.product_id, ci.quantity FROM cart_items ci WHERE ci.cart_id = ?
		 ON CONFLICT(cart_id, product_id) DO UPDATE SET quantity = cart_items.quantity + excluded.quantity`,
		userCartID, guestToken,
	)
	if err != nil {
		return fmt.Errorf("cartRepo.MergeGuestCart merge: %w", err)
	}

	// Delete guest cart.
	if _, err := tx.ExecContext(ctx, `DELETE FROM carts WHERE id=?`, guestToken); err != nil {
		return fmt.Errorf("cartRepo.MergeGuestCart delete: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `UPDATE carts SET updated_at=? WHERE id=?`, now, userCartID); err != nil {
		return fmt.Errorf("cartRepo.MergeGuestCart touch: %w", err)
	}

	return tx.Commit()
}

// ─── OrderRepository ────────────────────────────────────────────────────

type orderRepo struct{ db *sql.DB }

func NewOrderRepo(db *sql.DB) ports.OrderRepository { return &orderRepo{db} }

func (r *orderRepo) Create(ctx context.Context, order *domain.Order, idempotencyKey string) error {
	// Check idempotency first.
	var existingOrderID string
	err := r.db.QueryRowContext(ctx, `SELECT order_id FROM idempotency WHERE key=?`, idempotencyKey).Scan(&existingOrderID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("orderRepo.Create idempotency check: %w", err)
	}
	if existingOrderID != "" {
		existing, err := r.ByID(ctx, existingOrderID)
		if err != nil {
			return err
		}
		*order = *existing
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("orderRepo.Create begin: %w", err)
	}
	defer tx.Rollback()

	// Deduct stock for each item — concurrency-safe conditional UPDATE.
	for _, item := range order.Items {
		result, err := tx.ExecContext(ctx,
			`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`,
			item.Quantity, item.ProductID, item.Quantity,
		)
		if err != nil {
			return fmt.Errorf("orderRepo.Create stock: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("orderRepo.Create rowsaffected: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("orderRepo.Create: insufficient stock for product %s", item.ProductID)
		}
	}

	// Insert order.
	_, err = tx.ExecContext(ctx,
		`INSERT INTO orders (id, ref, user_id, email, status, subtotal, shipping_cost, discount_amount, total, shipping_address, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		order.ID, order.Ref, order.UserID, order.Email, order.Status,
		order.Subtotal.Cents(), order.ShippingCost.Cents(), order.DiscountAmount.Cents(), order.Total.Cents(),
		order.ShippingAddress, order.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("orderRepo.Create insert order: %w", err)
	}

	// Insert order items.
	for _, item := range order.Items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO order_items (id, order_id, product_id, name, unit_price, quantity, image_url) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id.NewV4(), order.ID, item.ProductID, item.Name, item.UnitPrice.Cents(), item.Quantity, item.ImageURL,
		); err != nil {
			return fmt.Errorf("orderRepo.Create insert item: %w", err)
		}
	}

	// Record idempotency key.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO idempotency (key, order_id, created_at) VALUES (?, ?, ?)`,
		idempotencyKey, order.ID, time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("orderRepo.Create idempotency: %w", err)
	}

	return tx.Commit()
}

func (r *orderRepo) ByID(ctx context.Context, orderID string) (*domain.Order, error) {
	var o domain.Order
	var userID sql.NullString
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, ref, user_id, email, status, subtotal, shipping_cost, discount_amount, total, shipping_address, created_at
		 FROM orders WHERE id=?`, orderID,
	).Scan(&o.ID, &o.Ref, &userID, &o.Email, &o.Status,
		(*int64)(&o.Subtotal), (*int64)(&o.ShippingCost), (*int64)(&o.DiscountAmount), (*int64)(&o.Total),
		&o.ShippingAddress, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("orderRepo.ByID: %w", err)
	}
	if userID.Valid {
		o.UserID = &userID.String
	}
	o.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	items, err := r.loadOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (r *orderRepo) ByRef(ctx context.Context, ref string) (*domain.Order, error) {
	var o domain.Order
	var userID sql.NullString
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, ref, user_id, email, status, subtotal, shipping_cost, discount_amount, total, shipping_address, created_at
		 FROM orders WHERE ref=?`, ref,
	).Scan(&o.ID, &o.Ref, &userID, &o.Email, &o.Status,
		(*int64)(&o.Subtotal), (*int64)(&o.ShippingCost), (*int64)(&o.DiscountAmount), (*int64)(&o.Total),
		&o.ShippingAddress, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("orderRepo.ByRef: %w", err)
	}
	if userID.Valid {
		o.UserID = &userID.String
	}
	o.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	items, err := r.loadOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

func (r *orderRepo) ListByUser(ctx context.Context, userID string, offset, limit int) ([]domain.Order, int, error) {
	// ANTI-IDOR: WHERE user_id=?
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE user_id=?`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("orderRepo.ListByUser count: %w", err)
	}
	return r.listOrders(ctx,
		`SELECT id, ref, user_id, email, status, subtotal, shipping_cost, discount_amount, total, shipping_address, created_at
		 FROM orders WHERE user_id=? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset, total,
	)
}

func (r *orderRepo) List(ctx context.Context, status string, offset, limit int) ([]domain.Order, int, error) {
	if status != "" {
		var total int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE status=?`, status).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("orderRepo.List count: %w", err)
		}
		return r.listOrders(ctx,
			`SELECT id, ref, user_id, email, status, subtotal, shipping_cost, discount_amount, total, shipping_address, created_at
			 FROM orders WHERE status=? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			status, limit, offset, total,
		)
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("orderRepo.List count: %w", err)
	}
	return r.listOrders(ctx,
		`SELECT id, ref, user_id, email, status, subtotal, shipping_cost, discount_amount, total, shipping_address, created_at
		 FROM orders ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset, total,
	)
}

func (r *orderRepo) listOrders(ctx context.Context, query string, args ...interface{}) ([]domain.Order, int, error) {
	total := args[len(args)-1].(int)
	sqlArgs := args[:len(args)-1]

	rows, err := r.db.QueryContext(ctx, query, sqlArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("orderRepo.listOrders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		var userID sql.NullString
		var createdAt string
		if err := rows.Scan(&o.ID, &o.Ref, &userID, &o.Email, &o.Status,
			(*int64)(&o.Subtotal), (*int64)(&o.ShippingCost), (*int64)(&o.DiscountAmount), (*int64)(&o.Total),
			&o.ShippingAddress, &createdAt,
		); err != nil {
			return nil, 0, fmt.Errorf("orderRepo.listOrders scan: %w", err)
		}
		if userID.Valid {
			o.UserID = &userID.String
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

		items, err := r.loadOrderItems(ctx, o.ID)
		if err != nil {
			return nil, 0, err
		}
		o.Items = items
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func (r *orderRepo) loadOrderItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT product_id, name, unit_price, quantity, image_url FROM order_items WHERE order_id=?`, orderID)
	if err != nil {
		return nil, fmt.Errorf("loadOrderItems: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Name, (*int64)(&item.UnitPrice), &item.Quantity, &item.ImageURL); err != nil {
			return nil, fmt.Errorf("loadOrderItems scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *orderRepo) UpdateStatus(ctx context.Context, orderID string, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status=? WHERE id=?`, status, orderID)
	if err != nil {
		return fmt.Errorf("orderRepo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *orderRepo) UpsertIdempotency(ctx context.Context, key string, orderID string) (string, error) {
	if orderID == "" {
		var existing string
		err := r.db.QueryRowContext(ctx, `SELECT order_id FROM idempotency WHERE key=?`, key).Scan(&existing)
		if err == sql.ErrNoRows {
			return "", nil
		}
		if err != nil {
			return "", fmt.Errorf("orderRepo.UpsertIdempotency: %w", err)
		}
		return existing, nil
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO idempotency (key, order_id, created_at) VALUES (?, ?, ?)`,
		key, orderID, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("orderRepo.UpsertIdempotency insert: %w", err)
	}
	return orderID, nil
}

// ─── DiscountRepository ─────────────────────────────────────────────────

type discountRepo struct{ db *sql.DB }

func NewDiscountRepo(db *sql.DB) ports.DiscountRepository { return &discountRepo{db} }

func scanDiscount(scanner interface{ Scan(...interface{}) error }) (domain.Discount, error) {
	var d domain.Discount
	var active int
	var expiresAt sql.NullString
	if err := scanner.Scan(&d.ID, &d.Code, &d.Percent, &active, &expiresAt); err != nil {
		return d, err
	}
	d.Active = active == 1
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		d.ExpiresAt = &t
	}
	return d, nil
}

func (r *discountRepo) ByCode(ctx context.Context, code string) (*domain.Discount, error) {
	d, err := scanDiscount(r.db.QueryRowContext(ctx,
		`SELECT id, code, percent, active, expires_at FROM discounts WHERE code=? AND active=1`, code))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("discountRepo.ByCode: %w", err)
	}
	if d.ExpiresAt != nil && d.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	return &d, nil
}

func (r *discountRepo) List(ctx context.Context) ([]domain.Discount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, code, percent, active, expires_at FROM discounts ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("discountRepo.List: %w", err)
	}
	defer rows.Close()

	var discounts []domain.Discount
	for rows.Next() {
		d, err := scanDiscount(rows)
		if err != nil {
			return nil, fmt.Errorf("discountRepo.List scan: %w", err)
		}
		discounts = append(discounts, d)
	}
	return discounts, rows.Err()
}

func (r *discountRepo) Create(ctx context.Context, d *domain.Discount) error {
	var active int
	if d.Active {
		active = 1
	}
	var expiresAt interface{}
	if d.ExpiresAt != nil {
		expiresAt = d.ExpiresAt.Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO discounts (id, code, percent, active, expires_at) VALUES (?, ?, ?, ?, ?)`,
		d.ID, d.Code, d.Percent, active, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("discountRepo.Create: %w", err)
	}
	return nil
}

func (r *discountRepo) Delete(ctx context.Context, discountID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM discounts WHERE id=?`, discountID)
	if err != nil {
		return fmt.Errorf("discountRepo.Delete: %w", err)
	}
	return nil
}

// ─── CategoryRepository ─────────────────────────────────────────────────

type categoryRepo struct{ db *sql.DB }

func NewCategoryRepo(db *sql.DB) ports.CategoryRepository { return &categoryRepo{db} }

func (r *categoryRepo) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.slug, c.name, c.parent_id,
		 (SELECT COUNT(*) FROM products WHERE category_id=c.id AND status='published') AS product_count
		 FROM categories c ORDER BY c.name`)
	if err != nil {
		return nil, fmt.Errorf("categoryRepo.List: %w", err)
	}
	defer rows.Close()

	var all []domain.Category
	for rows.Next() {
		var c domain.Category
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &parentID, &c.ProductCount); err != nil {
			return nil, fmt.Errorf("categoryRepo.List scan: %w", err)
		}
		all = append(all, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return all, nil
}

func (r *categoryRepo) BySlug(ctx context.Context, slug string) (*domain.Category, error) {
	var c domain.Category
	var parentID sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, slug, name, parent_id,
		 (SELECT COUNT(*) FROM products WHERE category_id=categories.id AND status='published')
		 FROM categories WHERE slug=?`, slug,
	).Scan(&c.ID, &c.Slug, &c.Name, &parentID, &c.ProductCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("categoryRepo.BySlug: %w", err)
	}
	return &c, nil
}

func (r *categoryRepo) Create(ctx context.Context, c *domain.Category) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (id, slug, name) VALUES (?, ?, ?)`,
		c.ID, c.Slug, c.Name,
	)
	if err != nil {
		return fmt.Errorf("categoryRepo.Create: %w", err)
	}
	return nil
}

func (r *categoryRepo) Update(ctx context.Context, c *domain.Category) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE categories SET slug=?, name=? WHERE id=?`,
		c.Slug, c.Name, c.ID,
	)
	if err != nil {
		return fmt.Errorf("categoryRepo.Update: %w", err)
	}
	return nil
}

func (r *categoryRepo) Delete(ctx context.Context, categoryID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id=?`, categoryID)
	if err != nil {
		return fmt.Errorf("categoryRepo.Delete: %w", err)
	}
	return nil
}

// ─── SessionStore ───────────────────────────────────────────────────────

type sessionStore struct{ db *sql.DB }

func NewSessionStore(db *sql.DB) ports.SessionStore { return &sessionStore{db} }

func (s *sessionStore) Create(ctx context.Context, userID string) (string, error) {
	var role string
	if err := s.db.QueryRowContext(ctx, `SELECT role FROM users WHERE id=?`, userID).Scan(&role); err != nil {
		return "", fmt.Errorf("sessionStore.Create user: %w", err)
	}

	token := id.NewV4()
	hash := sha256Hex(token)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, role, created_at) VALUES (?, ?, ?, ?)`,
		hash, userID, role, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("sessionStore.Create insert: %w", err)
	}
	return token, nil
}

func (s *sessionStore) Get(ctx context.Context, token string) (string, string, error) {
	hash := sha256Hex(token)
	var userID, role string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, role FROM sessions WHERE token_hash=?`, hash,
	).Scan(&userID, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", fmt.Errorf("sessionStore.Get: %w", err)
	}
	return userID, role, nil
}

func (s *sessionStore) Delete(ctx context.Context, token string) error {
	hash := sha256Hex(token)
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, hash)
	if err != nil {
		return fmt.Errorf("sessionStore.Delete: %w", err)
	}
	return nil
}

func (s *sessionStore) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, userID)
	if err != nil {
		return fmt.Errorf("sessionStore.DeleteAllForUser: %w", err)
	}
	return nil
}
