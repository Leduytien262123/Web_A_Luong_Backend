package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Discount struct {
	ID               uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Code             string         `json:"code" gorm:"unique;not null;size:50;index"`
	Name             string         `json:"name" gorm:"not null;size:200"`
	Description      string         `json:"description" gorm:"size:500"`
	Type             string         `json:"type" gorm:"not null;size:20;check:type IN ('percentage', 'fixed')"`
	Value            float64        `json:"value" gorm:"not null;type:decimal(10,2)"`
	MinOrderAmount   float64        `json:"min_order_amount" gorm:"type:decimal(10,2);default:0"`
	MaxDiscountValue float64        `json:"max_discount_value" gorm:"type:decimal(10,2)"`
	UsageLimit       int            `json:"usage_limit" gorm:"default:0"`
	UsedCount        int            `json:"used_count" gorm:"default:0"`
	IsActive         bool           `json:"is_active" gorm:"default:true;index"`
	StartDate        time.Time      `json:"start_date" gorm:"not null"`
	EndDate          time.Time      `json:"end_date" gorm:"not null"`
	CreatedAt        time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate hook để tự động tạo UUID cho Discount
func (d *Discount) BeforeCreate(tx *gorm.DB) (err error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return
}

// IsValid kiểm tra mã giảm giá có hợp lệ không
func (d *Discount) IsValid() bool {
	now := time.Now()
	return d.IsActive && 
		   now.After(d.StartDate) && 
		   now.Before(d.EndDate) && 
		   (d.UsageLimit == 0 || d.UsedCount < d.UsageLimit)
}

// CanApply kiểm tra có thể áp dụng mã giảm giá cho đơn hàng không
func (d *Discount) CanApply(orderAmount float64) bool {
	return d.IsValid() && orderAmount >= d.MinOrderAmount
}

// CalculateDiscount tính toán số tiền giảm giá
func (d *Discount) CalculateDiscount(orderAmount float64) float64 {
	if !d.CanApply(orderAmount) {
		return 0
	}

	var discountAmount float64
	if d.Type == "percentage" {
		discountAmount = orderAmount * (d.Value / 100)
		// Áp dụng giới hạn tối đa nếu có
		if d.MaxDiscountValue > 0 && discountAmount > d.MaxDiscountValue {
			discountAmount = d.MaxDiscountValue
		}
	} else if d.Type == "fixed" {
		discountAmount = d.Value
		// Không được giảm quá số tiền đơn hàng
		if discountAmount > orderAmount {
			discountAmount = orderAmount
		}
	}

	return discountAmount
}

