package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiscountRepo struct {
	db *gorm.DB
}

func NewDiscountRepo(db *gorm.DB) *DiscountRepo {
	if db == nil {
		db = app.GetDB()
	}
	return &DiscountRepo{db: db}
}

// Create tạo mới mã giảm giá
func (r *DiscountRepo) Create(discount *model.Discount) error {
	return r.db.Create(discount).Error
}

// GetByID lấy mã giảm giá theo ID
func (r *DiscountRepo) GetByID(id uuid.UUID) (*model.Discount, error) {
	var discount model.Discount
	err := r.db.Where("id = ?", id).First(&discount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("discount not found")
		}
		return nil, err
	}
	return &discount, nil
}

// GetByCode lấy mã giảm giá theo code
func (r *DiscountRepo) GetByCode(code string) (*model.Discount, error) {
	var discount model.Discount
	err := r.db.Where("code = ?", code).First(&discount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("discount not found")
		}
		return nil, err
	}
	return &discount, nil
}

// GetAll lấy danh sách mã giảm giá có phân trang
func (r *DiscountRepo) GetAll(page, limit int, activeOnly bool) ([]model.Discount, int64, error) {
	var discounts []model.Discount
	var total int64

	query := r.db.Model(&model.Discount{})
	if activeOnly {
		now := time.Now()
		query = query.Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now)
	}

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy danh sách mã giảm giá
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&discounts).Error

	return discounts, total, err
}

// Update cập nhật mã giảm giá
func (r *DiscountRepo) Update(discount *model.Discount) error {
	return r.db.Save(discount).Error
}

// Delete xóa mềm mã giảm giá
func (r *DiscountRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Discount{}).Error
}

// CheckCodeExists kiểm tra mã code đã tồn tại hay chưa
func (r *DiscountRepo) CheckCodeExists(code string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.Discount{}).Where("code = ?", code)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// GetActiveDiscounts lấy danh sách mã giảm giá đang hoạt động
func (r *DiscountRepo) GetActiveDiscounts() ([]model.Discount, error) {
	var discounts []model.Discount
	now := time.Now()
	err := r.db.Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Order("created_at DESC").
		Find(&discounts).Error
	return discounts, err
}

// GetValidDiscountForOrder lấy mã giảm giá hợp lệ cho đơn hàng
func (r *DiscountRepo) GetValidDiscountForOrder(code string, orderAmount float64) (*model.Discount, error) {
	discount, err := r.GetByCode(code)
	if err != nil {
		return nil, err
	}

	if !discount.CanApply(orderAmount) {
		return nil, errors.New("discount cannot be applied to this order")
	}

	return discount, nil
}

// IncrementUsedCount tăng số lần sử dụng của mã giảm giá
func (r *DiscountRepo) IncrementUsedCount(id uuid.UUID) error {
	return r.db.Model(&model.Discount{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + ?", 1)).Error
}

// GetExpiredDiscounts lấy danh sách mã giảm giá đã hết hạn
func (r *DiscountRepo) GetExpiredDiscounts() ([]model.Discount, error) {
	var discounts []model.Discount
	now := time.Now()
	err := r.db.Where("end_date < ?", now).
		Order("end_date DESC").
		Find(&discounts).Error
	return discounts, err
}

// GetDiscountStatistics lấy thống kê về mã giảm giá
func (r *DiscountRepo) GetDiscountStatistics() (map[string]interface{}, error) {
	var stats struct {
		TotalDiscounts   int64 `json:"total_discounts"`
		ActiveDiscounts  int64 `json:"active_discounts"`
		ExpiredDiscounts int64 `json:"expired_discounts"`
		UsedDiscounts    int64 `json:"used_discounts"`
	}

	now := time.Now()

	// Tổng số mã giảm giá
	r.db.Model(&model.Discount{}).Count(&stats.TotalDiscounts)

	// Mã giảm giá đang hoạt động
	r.db.Model(&model.Discount{}).
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Count(&stats.ActiveDiscounts)

	// Mã giảm giá đã hết hạn
	r.db.Model(&model.Discount{}).
		Where("end_date < ?", now).
		Count(&stats.ExpiredDiscounts)

	// Mã giảm giá đã được sử dụng
	r.db.Model(&model.Discount{}).
		Where("used_count > ?", 0).
		Count(&stats.UsedDiscounts)

	result := map[string]interface{}{
		"total_discounts":   stats.TotalDiscounts,
		"active_discounts":  stats.ActiveDiscounts,
		"expired_discounts": stats.ExpiredDiscounts,
		"used_discounts":    stats.UsedDiscounts,
	}

	return result, nil
}

// ToggleStatus đảo trạng thái hoạt động của mã giảm giá
func (r *DiscountRepo) ToggleStatus(id uuid.UUID) error {
	return r.db.Model(&model.Discount{}).
		Where("id = ?", id).
		Update("is_active", gorm.Expr("NOT is_active")).Error
}