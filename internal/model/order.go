package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID               uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	CreatorID        *uuid.UUID     `json:"creator_id" gorm:"type:char(36);index"`
	CreatorName      string         `json:"creator_name" gorm:"size:255;index"`
	OrderType        string         `json:"order_type" gorm:"size:50;index"`
	UserID           *uuid.UUID     `json:"user_id" gorm:"type:char(36);index"`
	OrderNumber      string         `json:"order_number" gorm:"unique;not null;size:100;index"`
	Status           string         `json:"status" gorm:"not null;size:50;default:pending;index"`
	PaymentStatus    string         `json:"payment_status" gorm:"not null;size:50;default:pending;index"`
	PaymentMethod    string         `json:"payment_method" gorm:"size:50"`
	TotalAmount      float64        `json:"total_amount" gorm:"not null;type:decimal(10,2)"`
	DiscountAmount   float64        `json:"discount_amount" gorm:"type:decimal(10,2);default:0"`
	ShippingAmount   float64        `json:"shipping_fee" gorm:"type:decimal(10,2);default:0"`
	FinalAmount      float64        `json:"final_amount" gorm:"not null;type:decimal(10,2)"`
	DiscountCode     string         `json:"discount_code" gorm:"size:50"`
	Address          string         `json:"address" gorm:"column:shipping_address;type:text;not null"`
	CustomerName     string         `json:"customer_name" gorm:"not null;size:255"`
	CustomerPhone    string         `json:"customer_phone" gorm:"not null;size:20;index"`
	CustomerEmail    string         `json:"customer_email" gorm:"not null;size:255;index"`
	Notes            string         `json:"notes" gorm:"type:text"`
	IsGuestOrder     bool           `json:"is_guest_order" gorm:"default:false;index"`
	ShippedAt        *time.Time     `json:"shipped_at"`
	DeliveredAt      *time.Time     `json:"delivered_at"`
	CancelledAt      *time.Time     `json:"cancelled_at"`
	CreatedAt        time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`

	User       *User        `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Creator    *User        `json:"creator_user,omitempty" gorm:"foreignKey:CreatorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	OrderItems []OrderItem  `json:"order_items,omitempty" gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

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

	Order     *Order   `json:"order,omitempty" gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Product   *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:RESTRICT"`
	Creator   *User    `json:"creator,omitempty" gorm:"foreignKey:CreatorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatorID *uuid.UUID `json:"creator_id" gorm:"type:char(36);index"`
}

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return
}

func (Order) TableName() string {
	return "orders"
}

func (OrderItem) TableName() string {
	return "order_items"
}

type OrderInput struct {
	UserID          *uuid.UUID       `json:"user_id"`
	PaymentMethod   string           `json:"payment_method" binding:"required,oneof=cod bank_transfer momo zalopay"`
	DiscountCode    string           `json:"discount_code"`
	Address         string           `json:"address" binding:"required"`
	CustomerName    string           `json:"customer_name" binding:"required"`
	CustomerPhone   string           `json:"customer_phone" binding:"required"`
	CustomerEmail   string           `json:"customer_email" binding:"required,email"`
	Notes           string           `json:"notes"`
	Items           []OrderItemInput `json:"items" binding:"required,min=1"`
}

type OrderItemInput struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type GuestOrderLookupInput struct {
	EmailOrPhone string `json:"email_or_phone" binding:"required" example:"user@example.com or 0123456789"`
}

type AdminOrderInput struct {
	CreatorID      uuid.UUID   `json:"creator_id" binding:"required"`
	CreatorName    string      `json:"creator_name" binding:"required"`
	CustomerName   string      `json:"customer_name" binding:"required"`
	Phone          string      `json:"phone" binding:"required"`
	Email          string      `json:"email" binding:"required,email"`
	Address        string      `json:"address" binding:"required"`
	Note           string      `json:"note"`
	DiscountCode   *string     `json:"discount_code"`
	ShippingFee    float64     `json:"shipping_fee"`
	Status         string      `json:"status" binding:"required,oneof=new pending confirmed processing shipped delivered cancelled"`
	PaymentMethod  string      `json:"payment_method" binding:"required,oneof=unpaid cod bank_transfer momo zalopay"`
	OrderType      string      `json:"order_type" binding:"required,oneof=retail wholesale online"`
	ProductIDs     []uuid.UUID `json:"product_ids" binding:"required,min=1"`
}

type AdminOrderUpdateInput struct {
	CustomerName   *string  `json:"customer_name"`
	Phone          *string  `json:"phone"`
	Email          *string  `json:"email"`
	Address        *string  `json:"address"`
	Note           *string  `json:"note"`
	DiscountCode   *string  `json:"discount_code"`
	ShippingFee    *float64 `json:"shipping_fee"`
	Status         *string  `json:"status" binding:"omitempty,oneof=new pending confirmed processing shipped delivered cancelled"`
	PaymentMethod  *string  `json:"payment_method" binding:"omitempty,oneof=unpaid cod bank_transfer momo zalopay"`
	OrderType      *string  `json:"order_type" binding:"omitempty,oneof=retail wholesale online"`
}

type OrderResponse struct {
	ID               uuid.UUID           `json:"id"`
	UserID           *uuid.UUID          `json:"user_id"`
	CreatorID        *uuid.UUID          `json:"creator_id,omitempty"`
	CreatorName      string              `json:"creator_name,omitempty"`
	OrderType        string              `json:"order_type,omitempty"`
	OrderNumber      string              `json:"order_number"`
	Status           string              `json:"status"`
	PaymentStatus    string              `json:"payment_status"`
	PaymentMethod    string              `json:"payment_method"`
	TotalAmount      float64             `json:"total_amount"`
	DiscountAmount   float64             `json:"discount_amount"`
	ShippingAmount   float64             `json:"shipping_fee"`
	FinalAmount      float64             `json:"final_amount"`
	DiscountCode     string              `json:"discount_code"`
	Address          string              `json:"address"`
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

type OrderDetailResponse struct {
	ID            uuid.UUID          `json:"id"`
	UserID        *uuid.UUID         `json:"user_id"`
	OrderNumber   string             `json:"order_number"`
	Status        string             `json:"status"`
	PaymentStatus string             `json:"payment_status"`
	PaymentMethod string             `json:"payment_method"`
	OrderType     string             `json:"order_type"`
	TotalAmount   float64            `json:"total_amount"`
	DiscountAmount float64           `json:"discount_amount"`
	ShippingAmount float64           `json:"shipping_fee"`
	FinalAmount   float64            `json:"final_amount"`
	IsGuestOrder  bool               `json:"is_guest_order"`
	ShippedAt     *time.Time         `json:"shipped_at"`
	DeliveredAt   *time.Time         `json:"delivered_at"`
	CancelledAt   *time.Time         `json:"cancelled_at"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Customer      CustomerInfo       `json:"customer"`
	Products      []ProductDetailInfo `json:"products"` // ghi chú: thay đổi thành danh sách ProductDetailInfo (product_id, name, quantity, price)
	Items         []ItemInfo         `json:"items"`
	Creator       *CreatorInfo       `json:"creator,omitempty"`
	InfoOrder     OrderInfo          `json:"info_order"`
	DiscountCode  string             `json:"discount_code"`
}

type CustomerInfo struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type ItemInfo struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Total     float64   `json:"total"`
}

type CreatorInfo struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type OrderInfo struct {
	Address string `json:"address"`
	Note    string `json:"note"`
}

type ProductDetailInfo struct {
	ProductID uuid.UUID `json:"product_id"` // ghi chú: id của sản phẩm
	Name      string    `json:"name"`       // ghi chú: tên sản phẩm hiển thị trong chi tiết
	Quantity  int       `json:"quantity"`   // ghi chú: số lượng sản phẩm trong đơn
	Price     float64   `json:"price"`      // ghi chú: giá đơn vị cho sản phẩm trong đơn
}

func (o *Order) ToResponse() OrderResponse {
	response := OrderResponse{
		ID:               o.ID,
		UserID:           o.UserID,
		CreatorID:        o.CreatorID,
		CreatorName:      o.CreatorName,
		OrderType:        o.OrderType,
		OrderNumber:      o.OrderNumber,
		Status:           o.Status,
		PaymentStatus:    o.PaymentStatus,
		PaymentMethod:    o.PaymentMethod,
		TotalAmount:      o.TotalAmount,
		DiscountAmount:   o.DiscountAmount,
		ShippingAmount:   o.ShippingAmount,
		FinalAmount:      o.FinalAmount,
		DiscountCode:     o.DiscountCode,
		Address:          o.Address,
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

func (o *Order) ToDetailResponse() OrderDetailResponse {
	response := OrderDetailResponse{
		ID:             o.ID,
		UserID:         o.UserID,
		OrderNumber:    o.OrderNumber,
		Status:         o.Status,
		PaymentStatus:  o.PaymentStatus,
		PaymentMethod:  o.PaymentMethod,
		OrderType:      o.OrderType,
		TotalAmount:    o.TotalAmount,
		DiscountAmount: o.DiscountAmount,
		ShippingAmount: o.ShippingAmount,
		FinalAmount:    o.FinalAmount,
		IsGuestOrder:   o.IsGuestOrder,
		ShippedAt:      o.ShippedAt,
		DeliveredAt:    o.DeliveredAt,
		CancelledAt:    o.CancelledAt,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		DiscountCode:   o.DiscountCode,
		Customer: CustomerInfo{
			Name:  o.CustomerName,
			Email: o.CustomerEmail,
			Phone: o.CustomerPhone,
		},
		InfoOrder: OrderInfo{
			Address: o.Address,
			Note:    o.Notes,
		},
	}

	if o.User != nil {
		response.Customer.Id = o.User.ID.String() // ghi chú: nếu user liên kết tồn tại, sử dụng user.ID
	} else if o.UserID != nil {
		response.Customer.Id = o.UserID.String() // ghi chú: fallback về UserID đã lưu
	}

	if len(o.OrderItems) > 0 {
		for _, item := range o.OrderItems {
			// thêm mục items đơn giản (giữ cấu trúc Items hiện có)
			response.Items = append(response.Items, ItemInfo{
				ProductID: item.ProductID, // ghi chú: id sản phẩm
				Quantity:  item.Quantity,  // ghi chú: số lượng trong đơn
				Price:     item.Price,     // ghi chú: giá mỗi đơn vị
				Total:     item.Total,     // ghi chú: tổng cho dòng này
			})

			// tạo entry chi tiết sản phẩm theo định dạng yêu cầu
			prod := ProductDetailInfo{
				ProductID: item.ProductID,        // ghi chú: gán id sản phẩm
				Name:      "",                   // ghi chú: mặc định rỗng, sẽ gán nếu product được preload
				Quantity:  item.Quantity,         // ghi chú: số lượng
				Price:     item.Price,            // ghi chú: giá đơn vị
			}

			if item.Product != nil {
				prod.Name = item.Product.Name // ghi chú: dùng tên sản phẩm nếu đã được tải kèm
			}

			response.Products = append(response.Products, prod) // ghi chú: thêm vào mảng products
		}
	}

	if o.Creator != nil {
		response.Creator = &CreatorInfo{
			ID:       o.Creator.ID,
			FullName: o.Creator.FullName,
		}
	} else if o.CreatorID != nil {
		response.Creator = &CreatorInfo{
			ID:       *o.CreatorID,
			FullName: o.CreatorName,
		}
	}

	return response
}