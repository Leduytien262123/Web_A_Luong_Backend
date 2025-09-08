package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepo struct {
	db *gorm.DB
}

func NewProductRepo() *ProductRepo {
	return &ProductRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một sản phẩm
func (r *ProductRepo) Create(product *model.Product) error {
	return r.db.Create(product).Error
}

// GetByID lấy sản phẩm theo ID kèm danh mục
func (r *ProductRepo) GetByID(id uuid.UUID) (*model.Product, error) {
	var product model.Product
	err := r.db.Preload("Category").Where("id = ?", id).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &product, nil
}

// GetBySKU lấy sản phẩm theo SKU
func (r *ProductRepo) GetBySKU(sku string) (*model.Product, error) {
	var product model.Product
	err := r.db.Preload("Category").Where("sku = ?", sku).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &product, nil
}

// GetAll lấy tất cả sản phẩm đang hoạt động (có phân trang)
func (r *ProductRepo) GetAll(page, limit int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Product{}).Where("is_active = ?", true).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy sản phẩm kèm danh mục
	err := r.db.Preload("Category").
		Where("is_active = ?", true).
		Offset(offset).
		Limit(limit).
		Find(&products).Error

	return products, total, err
}

// GetByCategoryID lấy sản phẩm theo ID danh mục
func (r *ProductRepo) GetByCategoryID(categoryID uuid.UUID, page, limit int) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	// Đếm tổng số bản ghi
	query := r.db.Model(&model.Product{}).Where("category_id = ? AND is_active = ?", categoryID, true)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy sản phẩm kèm danh mục
	err := r.db.Preload("Category").
		Where("category_id = ? AND is_active = ?", categoryID, true).
		Offset(offset).
		Limit(limit).
		Find(&products).Error

	return products, total, err
}

// Update cập nhật sản phẩm
func (r *ProductRepo) Update(product *model.Product) error {
	return r.db.Save(product).Error
}

// Delete xóa mềm sản phẩm
func (r *ProductRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.Product{}).Error
}

// CheckSKUExists kiểm tra SKU đã tồn tại (loại trừ sản phẩm khác khi truyền excludeID)
func (r *ProductRepo) CheckSKUExists(sku string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.Product{}).Where("sku = ?", sku)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// UpdateStock cập nhật tồn kho sản phẩm
func (r *ProductRepo) UpdateStock(id uuid.UUID, stock int) error {
	return r.db.Model(&model.Product{}).Where("id = ?", id).Update("stock", stock).Error
}
