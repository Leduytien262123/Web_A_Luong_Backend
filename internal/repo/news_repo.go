package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NewsRepo struct {
	db *gorm.DB
}

func NewNewsRepo() *NewsRepo {
	return &NewsRepo{
		db: app.GetDB(),
	}
}

// Create tạo mới một bài viết tin tức
func (r *NewsRepo) Create(news *model.News) error {
	return r.db.Create(news).Error
}

// GetByID lấy bài viết tin tức theo ID
func (r *NewsRepo) GetByID(id uuid.UUID) (*model.News, error) {
	var news model.News
	err := r.db.Preload("Author").Where("id = ?", id).First(&news).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news not found")
		}
		return nil, err
	}
	return &news, nil
}

// GetBySlug lấy bài viết tin tức theo slug
func (r *NewsRepo) GetBySlug(slug string) (*model.News, error) {
	var news model.News
	err := r.db.Preload("Author").Where("slug = ?", slug).First(&news).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news not found")
		}
		return nil, err
	}
	return &news, nil
}

// GetAll lấy danh sách tin tức có phân trang
func (r *NewsRepo) GetAll(page, limit int, publishedOnly bool) ([]model.News, int64, error) {
	var news []model.News
	var total int64

	query := r.db.Model(&model.News{})
	if publishedOnly {
		query = query.Where("is_published = ?", true)
	}

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy tin tức kèm thông tin tác giả
	err := query.Preload("Author").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&news).Error

	return news, total, err
}

// Update cập nhật một bài viết tin tức
func (r *NewsRepo) Update(news *model.News) error {
	return r.db.Save(news).Error
}

// Delete xóa mềm một bài viết tin tức
func (r *NewsRepo) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&model.News{}).Error
}

// IncrementViewCount tăng số lượt xem
func (r *NewsRepo) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&model.News{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// CheckSlugExists kiểm tra slug đã tồn tại hay chưa
func (r *NewsRepo) CheckSlugExists(slug string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.News{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// GetPublishedNews lấy danh sách tin tức đã được publish
func (r *NewsRepo) GetPublishedNews(page, limit int) ([]model.News, int64, error) {
	return r.GetAll(page, limit, true)
}

// GetLatestNews lấy tin tức mới nhất
func (r *NewsRepo) GetLatestNews(limit int) ([]model.News, error) {
	var news []model.News
	err := r.db.Where("is_published = ?", true).
		Preload("Author").
		Order("created_at DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// GetPopularNews lấy tin tức phổ biến (theo view count)
func (r *NewsRepo) GetPopularNews(limit int) ([]model.News, error) {
	var news []model.News
	err := r.db.Where("is_published = ?", true).
		Preload("Author").
		Order("view_count DESC").
		Limit(limit).
		Find(&news).Error
	return news, err
}

// SearchNews tìm kiếm tin tức theo từ khóa
func (r *NewsRepo) SearchNews(keyword string, page, limit int) ([]model.News, int64, error) {
	var news []model.News
	var total int64

	query := r.db.Model(&model.News{}).
		Where("is_published = ? AND (title LIKE ? OR content LIKE ? OR tags LIKE ?)", 
			true, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính toán offset
	offset := (page - 1) * limit

	// Lấy kết quả tìm kiếm
	err := query.Preload("Author").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&news).Error

	return news, total, err
}