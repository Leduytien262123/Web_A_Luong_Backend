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

// Create tạo mới mã giảm giá với các sản phẩm và danh mục áp dụng
func (r *DiscountRepo) Create(discount *model.Discount) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Tạo discount
		if err := tx.Create(discount).Error; err != nil {
			return err
		}
		return nil
	})
}

// CreateWithAssociations tạo mới mã giảm giá kèm các sản phẩm và danh mục áp dụng
func (r *DiscountRepo) CreateWithAssociations(discount *model.Discount, productIDs []uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Tạo discount
		if err := tx.Create(discount).Error; err != nil {
			return err
		}

		// Thêm sản phẩm áp dụng
		for _, productID := range productIDs {
			discountProduct := &model.DiscountProduct{
				DiscountID: discount.ID,
				ProductID:  productID,
			}
			if err := tx.Create(discountProduct).Error; err != nil {
				return err
			}
		}

		// Thêm danh mục áp dụng
		for _, categoryID := range categoryIDs {
			discountCategory := &model.DiscountCategory{
				DiscountID: discount.ID,
				CategoryID: categoryID,
			}
			if err := tx.Create(discountCategory).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetByID lấy mã giảm giá theo ID kèm thông tin liên kết
func (r *DiscountRepo) GetByID(id uuid.UUID) (*model.Discount, error) {
	var discount model.Discount
	err := r.db.Preload("AppliedProducts").
		Preload("AppliedProducts.Product").
		Preload("AppliedCategoryProducts").
		Preload("AppliedCategoryProducts.Category").
		Where("id = ?", id).First(&discount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("discount not found")
		}
		return nil, err
	}
	return &discount, nil
}

// GetByCode lấy mã giảm giá theo code kèm thông tin liên kết
func (r *DiscountRepo) GetByCode(code string) (*model.Discount, error) {
	var discount model.Discount
	err := r.db.Preload("AppliedProducts").
		Preload("AppliedProducts.Product").
		Preload("AppliedCategoryProducts").
		Preload("AppliedCategoryProducts.Category").
		Where("discount_code = ?", code).First(&discount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("discount not found")
		}
		return nil, err
	}
	return &discount, nil
}

// GetAll lấy danh sách mã giảm giá có phân trang và bộ lọc
func (r *DiscountRepo) GetAll(page, limit int, search, discountType, status string) ([]model.Discount, int64, error) {
	return r.GetAllWithCondition(page, limit, search, discountType, status, nil)
}

// GetAllWithCondition lấy danh sách mã giảm giá có phân trang và bộ lọc, bao gồm filter theo condition
func (r *DiscountRepo) GetAllWithCondition(page, limit int, search, discountType, status string, conditionFilter *float64) ([]model.Discount, int64, error) {
	var discounts []model.Discount
	var total int64
	now := time.Now()

	query := r.db.Model(&model.Discount{})

	// Filter by search term (tìm trong discount_code, name, description)
	if search != "" {
		query = query.Where("discount_code LIKE ? OR name LIKE ? OR description LIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Filter by type
	if discountType != "" {
		query = query.Where("type = ?", discountType)
	}

	// Filter by condition (điều kiện áp dụng)
	if conditionFilter != nil {
		query = query.Where("condition >= ?", *conditionFilter)
	}

	// Filter by status
	if status != "" {
		switch status {
		case "upcoming":
			query = query.Where("is_active = ? AND start_date > ?", true, now)
		case "active":
			query = query.Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now)
		case "stopped":
			query = query.Where("is_active = ?", false)
		case "ended":
			query = query.Where("end_date < ?", now)
		}
	}

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy danh sách mã giảm giá kèm thông tin liên kết
	err := query.Preload("AppliedProducts").
		Preload("AppliedProducts.Product").
		Preload("AppliedCategoryProducts").
		Preload("AppliedCategoryProducts.Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&discounts).Error

	return discounts, total, err
}

// Update cập nhật mã giảm giá
func (r *DiscountRepo) Update(discount *model.Discount) error {
	return r.db.Save(discount).Error
}

// UpdateWithAssociations cập nhật mã giảm giá kèm cập nhật sản phẩm và danh mục áp dụng
func (r *DiscountRepo) UpdateWithAssociations(discount *model.Discount, productIDs []uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Cập nhật discount
		if err := tx.Save(discount).Error; err != nil {
			return err
		}

		// Xóa các liên kết cũ
		if err := tx.Where("discount_id = ?", discount.ID).Delete(&model.DiscountProduct{}).Error; err != nil {
			return err
		}
		if err := tx.Where("discount_id = ?", discount.ID).Delete(&model.DiscountCategory{}).Error; err != nil {
			return err
		}

		// Thêm sản phẩm áp dụng mới
		for _, productID := range productIDs {
			discountProduct := &model.DiscountProduct{
				DiscountID: discount.ID,
				ProductID:  productID,
			}
			if err := tx.Create(discountProduct).Error; err != nil {
				return err
			}
		}

		// Thêm danh mục áp dụng mới
		for _, categoryID := range categoryIDs {
			discountCategory := &model.DiscountCategory{
				DiscountID: discount.ID,
				CategoryID: categoryID,
			}
			if err := tx.Create(discountCategory).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete xóa mềm mã giảm giá
func (r *DiscountRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Discount{}).Error
}

// CheckCodeExists kiểm tra mã code đã tồn tại hay chưa
func (r *DiscountRepo) CheckCodeExists(code string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.Discount{}).Where("discount_code = ?", code)
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
	err := r.db.Preload("AppliedProducts").
		Preload("AppliedProducts.Product").
		Preload("AppliedCategoryProducts").
		Preload("AppliedCategoryProducts.Category").
		Where("is_active = ? AND start_date <= ? AND end_date >= ?", true, now, now).
		Order("created_at DESC").
		Find(&discounts).Error
	return discounts, err
}

// GetValidDiscountForOrder lấy mã giảm giá hợp lệ cho đơn hàng
func (r *DiscountRepo) GetValidDiscountForOrder(code string, orderAmount float64, userID uuid.UUID) (*model.Discount, error) {
	discount, err := r.GetByCode(code)
	if (err != nil) {
		return nil, err
	}

	if !discount.CanApply(orderAmount) {
		return nil, errors.New("discount cannot be applied to this order")
	}

	// Kiểm tra user có thể sử dụng mã này không
	if !discount.CanUserUse(userID, r.db) {
		return nil, errors.New("user has exceeded usage limit for this discount")
	}

	return discount, nil
}

// IncrementUsedCount tăng số lần sử dụng của mã giảm giá
func (r *DiscountRepo) IncrementUsedCount(id uuid.UUID) error {
	return r.db.Model(&model.Discount{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + ?", 1)).Error
}

// RecordUserUsage ghi lại việc sử dụng mã giảm giá của user
func (r *DiscountRepo) RecordUserUsage(userID, discountID, orderID uuid.UUID) error {
	usage := &model.UserDiscountUsage{
		UserID:     userID,
		DiscountID: discountID,
		OrderID:    orderID,
	}
	return r.db.Create(usage).Error
}

// GetUserUsageCount lấy số lần user đã sử dụng mã giảm giá
func (r *DiscountRepo) GetUserUsageCount(userID, discountID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&model.UserDiscountUsage{}).
		Where("user_id = ? AND discount_id = ?", userID, discountID).
		Count(&count).Error
	return count, err
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

// CheckProductApplicable kiểm tra sản phẩm có được áp dụng mã giảm giá không
func (r *DiscountRepo) CheckProductApplicable(discountID, productID uuid.UUID) (bool, error) {
	discount, err := r.GetByID(discountID)
	if err != nil {
		return false, err
	}
	return discount.CheckProductApplicable(productID, r.db), nil
}

// GetDiscountsByProduct lấy danh sách mã giảm giá áp dụng cho sản phẩm
func (r *DiscountRepo) GetDiscountsByProduct(productID uuid.UUID) ([]model.Discount, error) {
	var discounts []model.Discount
	now := time.Now()

	// Lấy mã giảm giá áp dụng trực tiếp cho sản phẩm
	err := r.db.Distinct().
		Joins("LEFT JOIN discount_products dp ON discounts.id = dp.discount_id").
		Joins("LEFT JOIN discount_categories dc ON discounts.id = dc.discount_id").
		Joins("LEFT JOIN products p ON p.id = ? AND (dp.product_id = p.id OR dc.category_id = p.category_id)", productID).
		Where("discounts.is_active = ? AND discounts.start_date <= ? AND discounts.end_date >= ?", true, now, now).
		Where("dp.product_id IS NOT NULL OR dc.category_id IS NOT NULL OR (SELECT COUNT(*) FROM discount_products dp2 WHERE dp2.discount_id = discounts.id) = 0 AND (SELECT COUNT(*) FROM discount_categories dc2 WHERE dc2.discount_id = discounts.id) = 0").
		Find(&discounts).Error

	return discounts, err
}

// GetDiscountsByCategory lấy danh sách mã giảm giá áp dụng cho danh mục
func (r *DiscountRepo) GetDiscountsByCategory(categoryID uuid.UUID) ([]model.Discount, error) {
	var discounts []model.Discount
	now := time.Now()

	err := r.db.Joins("JOIN discount_categories dc ON discounts.id = dc.discount_id").
		Where("dc.category_id = ? AND discounts.is_active = ? AND discounts.start_date <= ? AND discounts.end_date >= ?", 
			categoryID, true, now, now).
		Find(&discounts).Error

	return discounts, err
}