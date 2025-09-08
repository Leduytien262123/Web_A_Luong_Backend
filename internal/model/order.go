package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID               uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	UserID           *uuid.UUID     `json:"user_id" gorm:"type:char(36);index"` // Nullable for guest orders
	OrderNumber      string         `json:"order_number" gorm:"unique;not null;size:100;index"`
	Status           string         `json:"status" gorm:"not null;size:50;default:pending;index"`
	PaymentStatus    string         `json:"payment_status" gorm:"not null;size:50;default:pending;index"`
	PaymentMethod    string         `json:"payment_method" gorm:"size:50"`
	TotalAmount      float64        `json:"total_amount" gorm:"not null;type:decimal(10,2)"`
	DiscountAmount   float64        `json:"discount_amount" gorm:"type:decimal(10,2);default:0"`
	ShippingAmount   float64        `json:"shipping_amount" gorm:"type:decimal(10,2);default:0"`
	FinalAmount      float64        `json:"final_amount" gorm:"not null;type:decimal(10,2)"`
	CouponCode       string         `json:"coupon_code" gorm:"size:50"`
	ShippingAddress  string         `json:"shipping_address" gorm:"type:text;not null"`
	BillingAddress   string         `json:"billing_address" gorm:"type:text"`
	CustomerName     string         `json:"customer_name" gorm:"not null;size:255"`
	CustomerPhone    string         `json:"customer_phone" gorm:"not null;size:20;index"` // Add index for lookup
	CustomerEmail    string         `json:"customer_email" gorm:"not null;size:255;index"` // Add index for lookup
	Notes            string         `json:"notes" gorm:"type:text"`
	IsGuestOrder     bool           `json:"is_guest_order" gorm:"default:false;index"` // New field to identify guest orders
	ShippedAt        *time.Time     `json:"shipped_at"`
	DeliveredAt      *time.Time     `json:"delivered_at"`
	CancelledAt      *time.Time     `json:"cancelled_at"`
	CreatedAt        time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User       *User        `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	OrderItems []OrderItem  `json:"order_items,omitempty" gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// BeforeCreate hook để tự động tạo UUID cho Order
func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return
}

type OrderItem struct {
	ID        uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	OrderID   uuid.UUID      `json:"order_id" gorm:"type:char(36);not null;index"`
	ProductID uuid.UUID      `json:"product_id" gorm:"type:char(36);not null;index"`
	Quantity  int            `json:"quantity" gorm:"not null;default:1"`
	Price     float64        `json:"price" gorm:"not null;type:decimal(10,2)"`
	Total     float64        `json:"total" gorm:"not null;type:decimal(10,2)"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Order   *Order   `json:"order,omitempty" gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Product *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

// BeforeCreate hook để tự động tạo UUID cho OrderItem
func (oi *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return
}

// TableName specifies the table name for Order model
func (Order) TableName() string {
	return "orders"
}

// TableName specifies the table name for OrderItem model
func (OrderItem) TableName() string {
	return "order_items"
}

type OrderInput struct {
	UserID          *uuid.UUID          `json:"user_id"` // Optional for guest orders
	PaymentMethod   string              `json:"payment_method" binding:"required,oneof=cod bank_transfer momo zalopay"`
	CouponCode      string              `json:"coupon_code"`
	ShippingAddress string              `json:"shipping_address" binding:"required"`
	BillingAddress  string              `json:"billing_address"`
	CustomerName    string              `json:"customer_name" binding:"required"`
	CustomerPhone   string              `json:"customer_phone" binding:"required"`
	CustomerEmail   string              `json:"customer_email" binding:"required,email"`
	Notes           string              `json:"notes"`
	Items           []OrderItemInput    `json:"items" binding:"required,min=1"`
}

type OrderItemInput struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type GuestOrderLookupInput struct {
	EmailOrPhone string `json:"email_or_phone" binding:"required" example:"user@example.com or 0123456789"`
}

type OrderResponse struct {
	ID               uuid.UUID           `json:"id"`
	UserID           *uuid.UUID          `json:"user_id"`
	OrderNumber      string              `json:"order_number"`
	Status           string              `json:"status"`
	PaymentStatus    string              `json:"payment_status"`
	PaymentMethod    string              `json:"payment_method"`
	TotalAmount      float64             `json:"total_amount"`
	DiscountAmount   float64             `json:"discount_amount"`
	ShippingAmount   float64             `json:"shipping_amount"`
	FinalAmount      float64             `json:"final_amount"`
	CouponCode       string              `json:"coupon_code"`
	ShippingAddress  string              `json:"shipping_address"`
	BillingAddress   string              `json:"billing_address"`
	CustomerName     string              `json:"customer_name"`
	CustomerPhone    string              `json:"customer_phone"`
	CustomerEmail    string              `json:"customer_email"`
	Notes            string              `json:"notes"`
	IsGuestOrder     bool                `json:"is_guest_order"`
	ShippedAt        *time.Time          `json:"shipped_at"`
	DeliveredAt      *time.Time          `json:"delivered_at"`
	OrderItems       []OrderItemResponse `json:"order_items,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type OrderItemResponse struct {
	ID        uuid.UUID        `json:"id"`
	OrderID   uuid.UUID        `json:"order_id"`
	ProductID uuid.UUID        `json:"product_id"`
	Product   *ProductResponse `json:"product,omitempty"`
	Quantity  int              `json:"quantity"`
	Price     float64          `json:"price"`
	Total     float64          `json:"total"`
}

// ToResponse converts Order to OrderResponse
func (o *Order) ToResponse() OrderResponse {
	response := OrderResponse{
		ID:               o.ID,
		UserID:           o.UserID,
		OrderNumber:      o.OrderNumber,
		Status:           o.Status,
		PaymentStatus:    o.PaymentStatus,
		PaymentMethod:    o.PaymentMethod,
		TotalAmount:      o.TotalAmount,
		DiscountAmount:   o.DiscountAmount,
		ShippingAmount:   o.ShippingAmount,
		FinalAmount:      o.FinalAmount,
		CouponCode:       o.CouponCode,
		ShippingAddress:  o.ShippingAddress,
		BillingAddress:   o.BillingAddress,
		CustomerName:     o.CustomerName,
		CustomerPhone:    o.CustomerPhone,
		CustomerEmail:    o.CustomerEmail,
		Notes:            o.Notes,
		IsGuestOrder:     o.IsGuestOrder,
		ShippedAt:        o.ShippedAt,
		DeliveredAt:      o.DeliveredAt,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}

	// Include order items if loaded
	if len(o.OrderItems) > 0 {
		for _, item := range o.OrderItems {
			itemResponse := OrderItemResponse{
				ID:        item.ID,
				OrderID:   item.OrderID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     item.Price,
				Total:     item.Total,
			}
			if item.Product != nil {
				productResponse := item.Product.ToResponse()
				itemResponse.Product = &productResponse
			}
			response.OrderItems = append(response.OrderItems, itemResponse)
		}
	}

	return response
}