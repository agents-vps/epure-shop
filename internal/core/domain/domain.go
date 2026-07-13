// Package domain contains pure business entities with zero dependencies.
package domain

import "time"

// Money represents an amount in cents (integer).
type Money int64

func (m Money) Cents() int64  { return int64(m) }
func (m Money) Euros() float64 { return float64(m) / 100.0 }

type Product struct {
	ID               string
	Slug             string
	CategoryID       string
	CategoryName     string
	Name             string
	Description      string
	Price            Money
	ComparePrice     *Money // nil if no promo
	Stock            int
	Status           string // "draft", "published"
	ImageURL         string
	Rating           float64
	ReviewCount      int
	CreatedAt        time.Time
}

type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Role         string // "customer", "admin"
	CreatedAt    time.Time
}

type Cart struct {
	ID        string // token
	UserID    *string
	Items     []CartItem
	Discount  *Discount
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c Cart) Subtotal() Money {
	var total int64
	for _, item := range c.Items {
		if item.Product != nil && item.Product.Status == "published" {
			total += item.Product.Price.Cents() * int64(item.Quantity)
		}
	}
	return Money(total)
}

func (c Cart) DiscountAmount() Money {
	if c.Discount == nil {
		return 0
	}
	sub := c.Subtotal().Cents()
	discount := sub * int64(c.Discount.Percent) / 100
	return Money(discount)
}

func (c Cart) Total() Money {
	total := c.Subtotal().Cents() - c.DiscountAmount().Cents()
	if total < 0 {
		total = 0
	}
	return Money(total)
}

func (c Cart) ItemCount() int {
	n := 0
	for _, item := range c.Items {
		n += item.Quantity
	}
	return n
}

type CartItem struct {
	ProductID string
	Product   *Product // populated on load
	Quantity  int
}

type Order struct {
	ID              string
	Ref             string // human-readable reference (UUID v7-style)
	UserID          *string
	Email           string
	Status          string
	Items           []OrderItem
	Subtotal        Money
	ShippingCost    Money
	DiscountAmount  Money
	Total           Money
	ShippingAddress string // JSON string
	CreatedAt       time.Time
}

type OrderItem struct {
	ProductID     string
	Name          string
	UnitPrice     Money
	Quantity      int
	ImageURL      string
}

type Discount struct {
	ID       string
	Code     string
	Percent  int
	Active   bool
	ExpiresAt *time.Time
}

type Category struct {
	ID           string
	Slug         string
	Name         string
	ProductCount int
	Children     []Category
}
