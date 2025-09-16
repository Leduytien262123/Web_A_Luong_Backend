package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepo() *CategoryRepo {
	return &CategoryRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một danh mục
func (r *CategoryRepo) Create(category *model.Category) error {
	return r.db.Create(category).Error
}

// GetByID lấy danh mục theo ID
func (r *CategoryRepo) GetByID(id uuid.UUID) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
}

// GetBySlug lấy danh mục theo slug
func (r *CategoryRepo) GetBySlug(slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
}

// GetAll lấy tất cả danh mục
func (r *CategoryRepo) GetAll() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Find(&categories).Error
	return categories, err
}

// GetAllWithPagination lấy tất cả danh mục có phân trang
func (r *CategoryRepo) GetAllWithPagination(page, limit int) ([]model.Category, int64, error) {
	var categories []model.Category
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Category{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy danh sách danh mục
	err := r.db.Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&categories).Error

	return categories, total, err
}

// GetAllWithProducts lấy tất cả danh mục kèm theo danh sách sản phẩm
func (r *CategoryRepo) GetAllWithProducts() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Preload("Products").Find(&categories).Error
	return categories, err
}

// GetAllWithProductsAndPagination lấy tất cả danh mục kèm theo danh sách sản phẩm có phân trang
func (r *CategoryRepo) GetAllWithProductsAndPagination(page, limit int) ([]model.Category, int64, error) {
	var categories []model.Category
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Category{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy danh sách danh mục kèm sản phẩm
	err := r.db.Preload("Products").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&categories).Error

	return categories, total, err
}

// Update cập nhật danh mục
func (r *CategoryRepo) Update(category *model.Category) error {
	return r.db.Save(category).Error
}

// Delete xóa mềm một danh mục
func (r *CategoryRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Category{}).Error
}

// CheckSlugExists kiểm tra slug đã tồn tại hay chưa (loại trừ danh mục có ID = excludeID)
func (r *CategoryRepo) CheckSlugExists(slug string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.Category{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}
