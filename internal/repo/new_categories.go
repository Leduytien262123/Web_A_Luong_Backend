package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NewsCategoryRepo struct {
	db *gorm.DB
}

func NewNewsCategoryRepo() *NewsCategoryRepo {
	return &NewsCategoryRepo{
		db: app.GetDB(),
	}
}

// Create tạo danh mục tin tức mới
func (r *NewsCategoryRepo) Create(category *model.NewsCategory) error {
	return r.db.Create(category).Error
}

// GetAll lấy tất cả danh mục tin tức với phân trang
func (r *NewsCategoryRepo) GetAll(page, limit int, activeOnly bool) ([]model.NewsCategory, int64, error) {
	var categories []model.NewsCategory
	var total int64

	query := r.db.Model(&model.NewsCategory{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get with pagination
	offset := (page - 1) * limit
	err := query.Preload("Parent").Preload("Children").
		Order("sort_order ASC, created_at DESC").
		Offset(offset).Limit(limit).Find(&categories).Error

	return categories, total, err
}

// GetByID lấy danh mục tin tức theo ID
func (r *NewsCategoryRepo) GetByID(id uuid.UUID) (*model.NewsCategory, error) {
	var category model.NewsCategory
	err := r.db.Preload("Parent").Preload("Children").
		Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news category not found")
		}
		return nil, err
	}
	return &category, nil
}

// GetBySlug lấy danh mục tin tức theo slug
func (r *NewsCategoryRepo) GetBySlug(slug string) (*model.NewsCategory, error) {
	var category model.NewsCategory
	err := r.db.Preload("Parent").Preload("Children").
		Where("slug = ? AND is_active = ?", slug, true).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news category not found")
		}
		return nil, err
	}
	return &category, nil
}

// Update cập nhật danh mục tin tức
func (r *NewsCategoryRepo) Update(category *model.NewsCategory) error {
	return r.db.Save(category).Error
}

// Delete xóa mềm danh mục tin tức
func (r *NewsCategoryRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.NewsCategory{}).Error
}

// GetTreeStructure lấy cấu trúc cây danh mục
func (r *NewsCategoryRepo) GetTreeStructure() ([]model.NewsCategory, error) {
	var categories []model.NewsCategory
	err := r.db.Where("parent_id IS NULL AND is_active = ?", true).
		Preload("Children", "is_active = ?", true).
		Order("sort_order ASC").Find(&categories).Error
	return categories, err
}

// CheckSlugExists kiểm tra slug đã tồn tại chưa
func (r *NewsCategoryRepo) CheckSlugExists(slug string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.NewsCategory{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// GetActiveCategories lấy danh sách danh mục đang hoạt động
func (r *NewsCategoryRepo) GetActiveCategories() ([]model.NewsCategory, error) {
	var categories []model.NewsCategory
	err := r.db.Where("is_active = ?", true).
		Order("sort_order ASC, name ASC").Find(&categories).Error
	return categories, err
}