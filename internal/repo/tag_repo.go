package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TagRepo struct {
	db *gorm.DB
}

func NewTagRepo() *TagRepo {
	return &TagRepo{
		db: app.GetDB(),
	}
}

// Create tạo tag mới
func (r *TagRepo) Create(tag *model.Tag) error {
	return r.db.Create(tag).Error
}

// GetAll lấy tất cả tags với phân trang
func (r *TagRepo) GetAll(page, limit int, activeOnly bool) ([]model.Tag, int64, error) {
	var tags []model.Tag
	var total int64

	query := r.db.Model(&model.Tag{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get with pagination
	offset := (page - 1) * limit
	err := query.Order("usage_count DESC, name ASC").
		Offset(offset).Limit(limit).Find(&tags).Error

	return tags, total, err
}

// GetByID lấy tag theo ID
func (r *TagRepo) GetByID(id uuid.UUID) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// GetBySlug lấy tag theo slug
func (r *TagRepo) GetBySlug(slug string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("slug = ? AND is_active = ?", slug, true).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// GetByIDs lấy nhiều tags theo danh sách IDs
func (r *TagRepo) GetByIDs(ids []uuid.UUID) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("id IN ? AND is_active = ?", ids, true).Find(&tags).Error
	return tags, err
}

// Update cập nhật tag
func (r *TagRepo) Update(tag *model.Tag) error {
	return r.db.Save(tag).Error
}

// Delete xóa mềm tag
func (r *TagRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Tag{}).Error
}

// CheckSlugExists kiểm tra slug đã tồn tại chưa
func (r *TagRepo) CheckSlugExists(slug string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.Tag{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// GetPopularTags lấy tags phổ biến nhất
func (r *TagRepo) GetPopularTags(limit int) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("is_active = ? AND usage_count > 0", true).
		Order("usage_count DESC").Limit(limit).Find(&tags).Error
	return tags, err
}

// IncrementUsageCount tăng số lần sử dụng tag
func (r *TagRepo) IncrementUsageCount(id uuid.UUID) error {
	return r.db.Model(&model.Tag{}).Where("id = ?", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error
}

// DecrementUsageCount giảm số lần sử dụng tag
func (r *TagRepo) DecrementUsageCount(id uuid.UUID) error {
	return r.db.Model(&model.Tag{}).Where("id = ? AND usage_count > 0", id).
		UpdateColumn("usage_count", gorm.Expr("usage_count - 1")).Error
}

// SearchTags tìm kiếm tags theo tên
func (r *TagRepo) SearchTags(keyword string, limit int) ([]model.Tag, error) {
	var tags []model.Tag
	err := r.db.Where("is_active = ? AND name ILIKE ?", true, "%"+keyword+"%").
		Order("usage_count DESC, name ASC").Limit(limit).Find(&tags).Error
	return tags, err
}