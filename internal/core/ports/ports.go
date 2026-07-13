// Package ports defines interfaces that the core domain depends on.
// Implementations live in adapters/.
package ports

import (
	"context"
	"io"
	"time"

	"github.com/agents-vps/epure-shop/internal/core/domain"
)

// ─── Repositories ───

type ProductRepository interface {
	List(ctx context.Context, categorySlug string, sort string, offset, limit int) ([]domain.Product, int, error)
	BySlug(ctx context.Context, slug string) (*domain.Product, error)
	ByID(ctx context.Context, id string) (*domain.Product, error)
	Create(ctx context.Context, p *domain.Product) error
	Update(ctx context.Context, p *domain.Product) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, offset, limit int) ([]domain.Product, int, error)
	ListByIDs(ctx context.Context, ids []string) ([]domain.Product, error)
}

type UserRepository interface {
	ByID(ctx context.Context, id string) (*domain.User, error)
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, u *domain.User) error
	List(ctx context.Context, offset, limit int) ([]domain.User, int, error)
}

type CartRepository interface {
	Get(ctx context.Context, token string) (*domain.Cart, error)
	Save(ctx context.Context, cart *domain.Cart) error
	Delete(ctx context.Context, token string) error
	AddItem(ctx context.Context, token string, productID string, qty int) error
	UpdateQuantity(ctx context.Context, token string, productID string, qty int) error
	RemoveItem(ctx context.Context, token string, productID string) error
	MergeGuestCart(ctx context.Context, guestToken string, userID string) error
}

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order, idempotencyKey string) error
	ByID(ctx context.Context, id string) (*domain.Order, error)
	ByRef(ctx context.Context, ref string) (*domain.Order, error)
	ListByUser(ctx context.Context, userID string, offset, limit int) ([]domain.Order, int, error)
	List(ctx context.Context, status string, offset, limit int) ([]domain.Order, int, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpsertIdempotency(ctx context.Context, key string, orderID string) (string, error)
}

type DiscountRepository interface {
	ByCode(ctx context.Context, code string) (*domain.Discount, error)
	List(ctx context.Context) ([]domain.Discount, error)
	Create(ctx context.Context, d *domain.Discount) error
	Delete(ctx context.Context, id string) error
}

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	BySlug(ctx context.Context, slug string) (*domain.Category, error)
	Create(ctx context.Context, c *domain.Category) error
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, id string) error
}

type SessionStore interface {
	Create(ctx context.Context, userID string) (token string, err error)
	Get(ctx context.Context, token string) (userID string, role string, err error)
	Delete(ctx context.Context, token string) error
	DeleteAllForUser(ctx context.Context, userID string) error
}

// ─── Infrastructure ───

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewV4() string
	NewV7() string
	Ref() string
}

// ─── Renderer ───

type Renderer interface {
	Render(w io.Writer, page string, data any) error
	RenderPartial(w io.Writer, partial string, data any) error
	RenderStatus(w io.Writer, status int, page string, data any) error
}
