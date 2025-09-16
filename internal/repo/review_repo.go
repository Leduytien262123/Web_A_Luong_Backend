package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReviewRepo struct {
	db *gorm.DB
}

func NewReviewRepo() *ReviewRepo {
	return &ReviewRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một đánh giá
func (r *ReviewRepo) Create(review *model.Review) error {
	return r.db.Create(review).Error
}

// GetByID lấy đánh giá theo ID
func (r *ReviewRepo) GetByID(id uuid.UUID) (*model.Review, error) {
	var review model.Review
	err := r.db.Preload("User").Preload("Product").Where("id = ?", id).First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("review not found")
		}
		return nil, err
	}
	return &review, nil
}

// GetByProductID lấy danh sách đánh giá theo ID sản phẩm
func (r *ReviewRepo) GetByProductID(productID uuid.UUID, page, limit int) ([]model.Review, int64, error) {
	var reviews []model.Review
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Review{}).Where("product_id = ? AND is_active = ?", productID, true).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy đánh giá kèm thông tin người dùng
	err := r.db.Preload("User").
		Where("product_id = ? AND is_active = ?", productID, true).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	return reviews, total, err
}

// GetByUserID lấy danh sách đánh giá theo ID người dùng
func (r *ReviewRepo) GetByUserID(userID uuid.UUID, page, limit int) ([]model.Review, int64, error) {
	var reviews []model.Review
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Review{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy đánh giá kèm thông tin sản phẩm
	err := r.db.Preload("Product").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&reviews).Error

	return reviews, total, err
}

// CheckUserReviewExists kiểm tra người dùng đã đánh giá sản phẩm này chưa
func (r *ReviewRepo) CheckUserReviewExists(userID, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.Review{}).Where("user_id = ? AND product_id = ?", userID, productID).Count(&count).Error
	return count > 0, err
}

// Update cập nhật một đánh giá
func (r *ReviewRepo) Update(review *model.Review) error {
	return r.db.Save(review).Error
}

// Delete xóa mềm một đánh giá
func (r *ReviewRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Review{}).Error
}

// ToggleStatus đảo trạng thái kích hoạt của đánh giá
func (r *ReviewRepo) ToggleStatus(id uuid.UUID) error {
	return r.db.Model(&model.Review{}).Where("id = ?", id).Update("is_active", gorm.Expr("NOT is_active")).Error
}

// GetProductRatingStats lấy thống kê rating của sản phẩm
func (r *ReviewRepo) GetProductRatingStats(productID uuid.UUID) (map[string]interface{}, error) {
	var stats struct {
		AverageRating float64 `json:"average_rating"`
		TotalReviews  int64   `json:"total_reviews"`
		Rating1Count  int64   `json:"rating_1_count"`
		Rating2Count  int64   `json:"rating_2_count"`
		Rating3Count  int64   `json:"rating_3_count"`
		Rating4Count  int64   `json:"rating_4_count"`
		Rating5Count  int64   `json:"rating_5_count"`
	}

	// Lấy rating trung bình và tổng số đánh giá
	err := r.db.Model(&model.Review{}).
		Select("AVG(rating) as average_rating, COUNT(*) as total_reviews").
		Where("product_id = ? AND is_active = ?", productID, true).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	// Lấy số lượng đánh giá theo từng mức rating
	for i := 1; i <= 5; i++ {
		var count int64
		r.db.Model(&model.Review{}).
			Where("product_id = ? AND is_active = ? AND rating = ?", productID, true, i).
			Count(&count)
		
		switch i {
		case 1:
			stats.Rating1Count = count
		case 2:
			stats.Rating2Count = count
		case 3:
			stats.Rating3Count = count
		case 4:
			stats.Rating4Count = count
		case 5:
			stats.Rating5Count = count
		}
	}

	result := map[string]interface{}{
		"average_rating":  stats.AverageRating,
		"total_reviews":   stats.TotalReviews,
		"rating_1_count":  stats.Rating1Count,
		"rating_2_count":  stats.Rating2Count,
		"rating_3_count":  stats.Rating3Count,
		"rating_4_count":  stats.Rating4Count,
		"rating_5_count":  stats.Rating5Count,
	}

	return result, nil
}

// GetLatestReviews lấy các đánh giá mới nhất
func (r *ReviewRepo) GetLatestReviews(limit int) ([]model.Review, error) {
	var reviews []model.Review
	err := r.db.Preload("User").Preload("Product").
		Where("is_active = ?", true).
		Order("created_at DESC").
		Limit(limit).
		Find(&reviews).Error
	return reviews, err
}

// GetHighRatingReviews lấy các đánh giá có rating cao (4-5 sao)
func (r *ReviewRepo) GetHighRatingReviews(productID uuid.UUID, limit int) ([]model.Review, error) {
	var reviews []model.Review
	err := r.db.Preload("User").
		Where("product_id = ? AND is_active = ? AND rating >= ?", productID, true, 4).
		Order("rating DESC, created_at DESC").
		Limit(limit).
		Find(&reviews).Error
	return reviews, err
}