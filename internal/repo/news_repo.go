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

// Create tạo mới một bài viết tin tức với tags và categories
func (r *NewsRepo) Create(news *model.News) error {
	return r.db.Create(news).Error
}

// CreateWithAssociations tạo tin tức với tags và categories
func (r *NewsRepo) CreateWithAssociations(news *model.News, tagIDs []uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create news
		if err := tx.Create(news).Error; err != nil {
			return err
		}

		// Associate tags
		if len(tagIDs) > 0 {
			var tags []model.Tag
			if err := tx.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
				return err
			}
			if err := tx.Model(news).Association("Tags").Append(tags); err != nil {
				return err
			}
			// Increment usage count for tags
			for _, tagID := range tagIDs {
				tx.Model(&model.Tag{}).Where("id = ?", tagID).
					UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
			}
		}

		// Associate categories
		if len(categoryIDs) > 0 {
			var categories []model.NewsCategory
			if err := tx.Where("id IN ?", categoryIDs).Find(&categories).Error; err != nil {
				return err
			}
			if err := tx.Model(news).Association("Categories").Append(categories); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetByID lấy bài viết tin tức theo ID với đầy đủ quan hệ
func (r *NewsRepo) GetByID(id uuid.UUID) (*model.News, error) {
	var news model.News
	err := r.db.Preload("Creator").Preload("Category").Preload("Categories").Preload("Tags").
		Where("id = ?", id).First(&news).Error
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
	err := r.db.Preload("Creator").Preload("Category").Preload("Categories").Preload("Tags").
		Where("slug = ? AND is_published = ?", slug, true).First(&news).Error
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

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get with pagination and relations
	offset := (page - 1) * limit
	err := query.Preload("Creator").Preload("Category").Preload("Tags").
		Order("created_at DESC").
		Offset(offset).Limit(limit).Find(&news).Error

	return news, total, err
}

// GetByCategory lấy tin tức theo danh mục
func (r *NewsRepo) GetByCategory(categoryID uuid.UUID, page, limit int) ([]model.News, int64, error) {
	var news []model.News
	var total int64

	// Query through main category or association table
	query := r.db.Model(&model.News{}).
		Where("is_published = ? AND (category_id = ? OR EXISTS (SELECT 1 FROM news_category_associations nca WHERE nca.news_id = news.id AND nca.category_id = ?))", 
			true, categoryID, categoryID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get results
	offset := (page - 1) * limit
	err := query.Preload("Creator").Preload("Category").Preload("Tags").
		Order("created_at DESC").
		Offset(offset).Limit(limit).Find(&news).Error

	return news, total, err
}

// GetByTag lấy tin tức theo tag
func (r *NewsRepo) GetByTag(tagID uuid.UUID, page, limit int) ([]model.News, int64, error) {
	var news []model.News
	var total int64

	query := r.db.Model(&model.News{}).
		Joins("JOIN news_tags nt ON nt.news_id = news.id").
		Where("news.is_published = ? AND nt.tag_id = ?", true, tagID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get results
	offset := (page - 1) * limit
	err := query.Preload("Creator").Preload("Category").Preload("Tags").
		Order("news.created_at DESC").
		Offset(offset).Limit(limit).Find(&news).Error

	return news, total, err
}

// Update cập nhật một bài viết tin tức
func (r *NewsRepo) Update(news *model.News) error {
	return r.db.Save(news).Error
}

// UpdateWithAssociations cập nhật tin tức với tags và categories
func (r *NewsRepo) UpdateWithAssociations(news *model.News, tagIDs []uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get old tags to decrement usage count
		var oldTags []model.Tag
		tx.Model(news).Association("Tags").Find(&oldTags)

		// Update news
		if err := tx.Save(news).Error; err != nil {
			return err
		}

		// Replace tags
		if tagIDs != nil {
			var newTags []model.Tag
			if len(tagIDs) > 0 {
				if err := tx.Where("id IN ?", tagIDs).Find(&newTags).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(news).Association("Tags").Replace(newTags); err != nil {
				return err
			}

			// Update usage counts
			for _, oldTag := range oldTags {
				tx.Model(&model.Tag{}).Where("id = ? AND usage_count > 0", oldTag.ID).
					UpdateColumn("usage_count", gorm.Expr("usage_count - 1"))
			}
			for _, newTag := range newTags {
				tx.Model(&model.Tag{}).Where("id = ?", newTag.ID).
					UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
			}
		}

		// Replace categories
		if categoryIDs != nil {
			var newCategories []model.NewsCategory
			if len(categoryIDs) > 0 {
				if err := tx.Where("id IN ?", categoryIDs).Find(&newCategories).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(news).Association("Categories").Replace(newCategories); err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete xóa mềm một bài viết tin tức
func (r *NewsRepo) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get news with tags to decrement usage count
		var news model.News
		if err := tx.Preload("Tags").Where("id = ?", id).First(&news).Error; err != nil {
			return err
		}

		// Decrement tag usage counts
		for _, tag := range news.Tags {
			tx.Model(&model.Tag{}).Where("id = ? AND usage_count > 0", tag.ID).
				UpdateColumn("usage_count", gorm.Expr("usage_count - 1"))
		}

		// Delete news (soft delete)
		return tx.Where("id = ?", id).Delete(&model.News{}).Error
	})
}

// IncrementViewCount tăng số lượt xem
func (r *NewsRepo) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&model.News{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
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

// GetFeaturedNews lấy tin tức nổi bật
func (r *NewsRepo) GetFeaturedNews(limit int) ([]model.News, error) {
	var news []model.News
	err := r.db.Where("is_published = ? AND is_featured = ?", true, true).
		Preload("Creator").Preload("Category").Preload("Tags").
		Order("created_at DESC").Limit(limit).Find(&news).Error
	return news, err
}

// GetLatestNews lấy tin tức mới nhất
func (r *NewsRepo) GetLatestNews(limit int) ([]model.News, error) {
	var news []model.News
	err := r.db.Where("is_published = ?", true).
		Preload("Creator").Preload("Category").Preload("Tags").
		Order("created_at DESC").Limit(limit).Find(&news).Error
	return news, err
}

// GetPopularNews lấy tin tức phổ biến (theo view count)
func (r *NewsRepo) GetPopularNews(limit int) ([]model.News, error) {
	var news []model.News
	err := r.db.Where("is_published = ?", true).
		Preload("Creator").Preload("Category").Preload("Tags").
		Order("view_count DESC").Limit(limit).Find(&news).Error
	return news, err
}

// SearchNews tìm kiếm tin tức theo từ khóa
func (r *NewsRepo) SearchNews(keyword string, page, limit int) ([]model.News, int64, error) {
	var news []model.News
	var total int64

	query := r.db.Model(&model.News{}).
		Where("is_published = ? AND (title ILIKE ? OR summary ILIKE ? OR content ILIKE ?)", 
			true, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get results
	offset := (page - 1) * limit
	err := query.Preload("Creator").Preload("Category").Preload("Tags").
		Order("created_at DESC").
		Offset(offset).Limit(limit).Find(&news).Error

	return news, total, err
}