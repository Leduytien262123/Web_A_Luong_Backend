package repo

import (
	"backend/app"
	"backend/internal/helpers"
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
	err := r.db.Preload("Parent").Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
}

// GetByIDWithArticles lấy danh mục theo ID kèm bài viết
func (r *CategoryRepo) GetByIDWithArticles(id uuid.UUID) (*model.Category, error) {
	var category model.Category
	err := r.db.Preload("Parent").Preload("Articles").Where("id = ?", id).First(&category).Error
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
	err := r.db.Preload("Parent").Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
}

// GetBySlugWithArticles lấy danh mục theo slug kèm bài viết
func (r *CategoryRepo) GetBySlugWithArticles(slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.Preload("Parent").Preload("Articles").Where("slug = ?", slug).First(&category).Error
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
	err := r.db.Preload("Parent").Find(&categories).Error
	return categories, err
}

// GetActive lấy danh mục đang hoạt động
func (r *CategoryRepo) GetActive() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Preload("Parent").
		Where("is_active = ?", true).
		Order("display_order ASC, created_at DESC").
		Find(&categories).Error
	return categories, err
}

// GetActiveWithArticles lấy danh mục hoạt động kèm bài viết đã xuất bản
func (r *CategoryRepo) GetActiveWithArticles() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Preload("Parent").
		Preload("Articles", "status IN ? AND is_active = ?", []string{"post"}, true).
		Where("is_active = ?", true).
		Order("display_order ASC, created_at DESC").
		Find(&categories).Error
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
	err := r.db.Preload("Parent").Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&categories).Error

	return categories, total, err
}

// GetAllWithArticles lấy tất cả danh mục kèm theo danh sách bài viết
func (r *CategoryRepo) GetAllWithArticles() ([]model.Category, error) {
	var categories []model.Category
	err := r.db.Preload("Parent").Preload("Articles").Find(&categories).Error
	return categories, err
}

// GetAllWithArticlesAndPagination lấy tất cả danh mục kèm theo danh sách bài viết có phân trang
func (r *CategoryRepo) GetAllWithArticlesAndPagination(page, limit int) ([]model.Category, int64, error) {
	var categories []model.Category
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Category{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy danh sách danh mục kèm sản phẩm
	err := r.db.Preload("Parent").Preload("Articles").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&categories).Error

	return categories, total, err
}

// GetHomeCategoriesWithArticles lấy các danh mục có show_on_home = true kèm bài viết đã xuất bản
// limitPerCategory: số lượng bài viết lấy cho mỗi danh mục (0 = không giới hạn)
func (r *CategoryRepo) GetHomeCategoriesWithArticles(limitPerCategory int) ([]model.Category, error) {
	var categories []model.Category

	preloadArticles := func(db *gorm.DB) *gorm.DB {
		q := db.Where("status IN ? AND is_active = ?", []string{"post"}, true).
			Order("published_at DESC, created_at DESC")
		if limitPerCategory > 0 {
			q = q.Limit(limitPerCategory)
		}
		return q
	}

	err := r.db.Preload("Parent").
		Preload("Articles", preloadArticles).
		Where("show_on_home = ? AND is_active = ?", true, true).
		Order("display_order ASC, created_at DESC").
		Find(&categories).Error
	return categories, err
}

// GetActiveBySlugWithArticles lấy danh mục hoạt động theo slug kèm bài viết đã xuất bản
func (r *CategoryRepo) GetActiveBySlugWithArticles(slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.Preload("Parent").
		Preload("Articles", "status IN ?", []string{"post"}).
		Where("slug = ? AND is_active = ?", slug, true).
		First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return &category, nil
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

// GetByName lấy danh mục theo tên (search tương đối với text không dấu)
func (r *CategoryRepo) GetByName(name string, includeArticles bool) ([]model.Category, error) {
	// Giữ nguyên hàm cũ để tương thích ngược
	results, _, err := r.GetByNameWithPagination(name, includeArticles, 1, int(^uint(0)>>1))
	return results, err
}

// GetByNameWithPagination tìm kiếm danh mục theo tên và phân trang
func (r *CategoryRepo) GetByNameWithPagination(name string, includeArticles bool, page, limit int) ([]model.Category, int64, error) {
	var categories []model.Category

	// Nếu search term rỗng, trả về mảng rỗng
	if name == "" {
		return categories, 0, nil
	}

	// Lấy tất cả danh mục (vẫn cần vì IsSearchMatch hoạt động trên text không dấu)
	query := r.db
	if includeArticles {
		query = query.Preload("Articles")
	}

	err := query.Find(&categories).Error
	if err != nil {
		return nil, 0, err
	}

	// Lọc các danh mục phù hợp với search term
	var filteredCategories []model.Category
	for _, category := range categories {
		if helpers.IsSearchMatch(name, category.Name) {
			filteredCategories = append(filteredCategories, category)
		}
	}

	total := int64(len(filteredCategories))

	// Chuẩn hóa page/limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 1
	}

	// Áp dụng phân trang thủ công trên slice đã lọc
	start := (page - 1) * limit
	if start >= len(filteredCategories) {
		return []model.Category{}, total, nil
	}

	end := start + limit
	if end > len(filteredCategories) {
		end = len(filteredCategories)
	}

	return filteredCategories[start:end], total, nil
}
