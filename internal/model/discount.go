package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Discount struct {
	ID                       uuid.UUID                   `gorm:"type:char(36);primaryKey" json:"id"`
	DiscountCode             string                      `json:"discount_code" gorm:"unique;not null;size:50;index"`
	Name                     string                      `json:"name" gorm:"not null;size:200"`
	Description              string                      `json:"description" gorm:"size:500"`
	Type                     string                      `json:"type" gorm:"not null;size:20"` // Tạm thời để string
	ValueVoucher             float64                     `json:"value_voucher" gorm:"not null;type:decimal(10,2)"`
	MinOrderAmount           float64                     `json:"min_order_amount" gorm:"type:decimal(10,2);default:0"`
	Condition                float64                     `json:"condition" gorm:"type:decimal(10,2);default:0;comment:Điều kiện áp dụng - số tiền tối thiểu của đơn hàng"`
	MaximumDiscount          float64                     `json:"maximum_discount" gorm:"type:decimal(10,2);default:0;comment:Giới hạn tối đa số tiền giảm cho mã giảm %"`
	Quantity                 int                         `json:"quantity" gorm:"not null;default:0;comment:Tổng số lượng mã"`
	UsageLimit               int                         `json:"usage_limit" gorm:"default:0;comment:0=không giới hạn,1=giới hạn theo usage_count"`
	UsageCount               int                         `json:"usage_count" gorm:"default:0;comment:Số lần đã sử dụng"`
	UsedCount                int                         `json:"used_count" gorm:"default:0;comment:Đếm số lần sử dụng thực tế"`
	IsActive                 bool                        `json:"is_active" gorm:"default:true;index"`
	StartDate                time.Time                   `json:"start_at" gorm:"not null"`
	EndDate                  time.Time                   `json:"end_at" gorm:"not null"`
	CreatedAt                time.Time                   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                time.Time                   `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt                gorm.DeletedAt              `json:"-" gorm:"index"`

	// Relationships
	AppliedProducts          []DiscountProduct           `json:"applied_products,omitempty" gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	AppliedCategoryProducts  []DiscountCategory          `json:"applied_category_products,omitempty" gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserUsages               []UserDiscountUsage         `json:"user_usages,omitempty" gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// DiscountProduct - Bảng liên kết mã giảm giá với sản phẩm cụ thể
type DiscountProduct struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	DiscountID uuid.UUID `gorm:"type:char(36);not null;index" json:"discount_id"`
	ProductID  uuid.UUID `gorm:"type:char(36);not null;index" json:"product_id"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Relationships
	Discount *Discount `json:"discount,omitempty" gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Product  *Product  `json:"product,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// DiscountCategory - Bảng liên kết mã giảm giá với danh mục sản phẩm
type DiscountCategory struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	DiscountID uuid.UUID `gorm:"type:char(36);not null;index" json:"discount_id"`
	CategoryID uuid.UUID `gorm:"type:char(36);not null;index" json:"category_id"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Relationships
	Discount *Discount `json:"discount,omitempty" gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Category *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// UserDiscountUsage - Bảng theo dõi việc sử dụng mã giảm giá của user
type UserDiscountUsage struct {
	ID         uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:char(36);not null;index" json:"user_id"`
	DiscountID uuid.UUID `gorm:"type:char(36);not null;index" json:"discount_id"`
	OrderID    uuid.UUID `gorm:"type:char(36);not null;index" json:"order_id"`
	UsedAt     time.Time `json:"used_at" gorm:"autoCreateTime"`

	// Relationships
	User     *User     `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Discount *Discount `json:"discount,omitempty" gorm:"foreignKey:DiscountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Order    *Order    `json:"order,omitempty" gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// BeforeCreate hooks
func (d *Discount) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

func (dp *DiscountProduct) BeforeCreate(tx *gorm.DB) (err error) {
	if dp.ID == uuid.Nil {
		dp.ID = uuid.New()
	}
	return
}

func (dc *DiscountCategory) BeforeCreate(tx *gorm.DB) (err error) {
	if dc.ID == uuid.Nil {
		dc.ID = uuid.New()
	}
	return
}

func (udu *UserDiscountUsage) BeforeCreate(tx *gorm.DB) (err error) {
	if udu.ID == uuid.Nil {
		udu.ID = uuid.New()
	}
	return
}

// Table names
func (Discount) TableName() string {
	return "discounts"
}

func (DiscountProduct) TableName() string {
	return "discount_products"
}

func (DiscountCategory) TableName() string {
	return "discount_categories"
}

func (UserDiscountUsage) TableName() string {
	return "user_discount_usages"
}

// IsValid kiểm tra mã giảm giá có hợp lệ không
func (d *Discount) IsValid() bool {
	now := time.Now()
	return d.IsActive && 
		now.After(d.StartDate) && 
		now.Before(d.EndDate) && 
		(d.UsageLimit == 0 || d.UsedCount < d.UsageCount)
}

// CanApply kiểm tra có thể áp dụng mã giảm giá cho đơn hàng không
func (d *Discount) CanApply(orderAmount float64) bool {
	return d.IsValid() && orderAmount >= d.Condition
}

// CanUserUse kiểm tra user có thể sử dụng mã giảm giá không (theo usage_limit)
func (d *Discount) CanUserUse(userID uuid.UUID, db *gorm.DB) bool {
	if d.UsageLimit == 0 {
		return true // Không giới hạn
	}

	// Đếm số lần user đã sử dụng mã này
	var count int64
	db.Model(&UserDiscountUsage{}).
		Where("user_id = ? AND discount_id = ?", userID, d.ID).
		Count(&count)

	return int(count) < d.UsageCount
}

// CalculateDiscount tính toán số tiền giảm giá
func (d *Discount) CalculateDiscount(orderAmount float64) float64 {
	if !d.CanApply(orderAmount) {
		return 0
	}

	var discountAmount float64
	if d.Type == "percentage" { // percentage
		discountAmount = orderAmount * (d.ValueVoucher / 100)
		// Áp dụng giới hạn tối đa nếu có
		if d.MaximumDiscount > 0 && discountAmount > d.MaximumDiscount {
			discountAmount = d.MaximumDiscount
		}
	} else if d.Type == "fixed" { // fixed
		discountAmount = d.ValueVoucher
		// Không được giảm quá số tiền đơn hàng
		if discountAmount > orderAmount {
			discountAmount = orderAmount
		}
	}

	return discountAmount
}

// CheckProductApplicable kiểm tra sản phẩm có được áp dụng mã giảm giá không
func (d *Discount) CheckProductApplicable(productID uuid.UUID, db *gorm.DB) bool {
	// Nếu không có sản phẩm cụ thể và không có danh mục cụ thể thì áp dụng cho tất cả
	var productCount, categoryCount int64
	
	db.Model(&DiscountProduct{}).Where("discount_id = ?", d.ID).Count(&productCount)
	db.Model(&DiscountCategory{}).Where("discount_id = ?", d.ID).Count(&categoryCount)
	
	if productCount == 0 && categoryCount == 0 {
		return true // Áp dụng cho tất cả sản phẩm
	}

	// Kiểm tra sản phẩm cụ thể
	if productCount > 0 {
		var count int64
		db.Model(&DiscountProduct{}).
			Where("discount_id = ? AND product_id = ?", d.ID, productID).
			Count(&count)
		if count > 0 {
			return true
		}
	}

	// Kiểm tra theo danh mục
	if categoryCount > 0 {
		var product Product
		if err := db.Where("id = ?", productID).First(&product).Error; err == nil && product.CategoryID != nil {
			var count int64
			db.Model(&DiscountCategory{}).
				Where("discount_id = ? AND category_id = ?", d.ID, *product.CategoryID).
				Count(&count)
			if count > 0 {
				return true
			}
		}
	}

	return false
}

