package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ArticleRepo struct {
	db *gorm.DB
}

const (
	articleStatusDraft = "draft"
	articleStatusPost  = "post"
)

var publishedStatuses = []string{articleStatusPost}

func NewArticleRepo() *ArticleRepo {
	return &ArticleRepo{
		db: app.GetDB(),
	}
}

// Create tạo bài viết mới
func (r *ArticleRepo) Create(article *model.Article) error {
	return r.db.Create(article).Error
}

// GetByID lấy bài viết theo ID
func (r *ArticleRepo) GetByID(id uuid.UUID) (*model.Article, error) {
	var article model.Article
	err := r.db.Preload("Author").Preload("Category").
		Where("id = ?", id).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, err
	}
	return &article, nil
}

// GetBySlug lấy bài viết theo slug
func (r *ArticleRepo) GetBySlug(slug string) (*model.Article, error) {
	var article model.Article
	err := r.db.Preload("Author").Preload("Category").
		Where("slug = ?", slug).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, err
	}
	return &article, nil
}

// GetAll lấy tất cả bài viết với phân trang
func (r *ArticleRepo) GetAll(page, limit int, published *bool) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.Article{})
	if published != nil {
		if *published {
			query = query.Where("status IN ?", publishedStatuses)
		} else {
			query = query.Where("status = ?", articleStatusDraft)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Author").Preload("Category").
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// GetPublished lấy bài viết đã xuất bản với phân trang
func (r *ArticleRepo) GetPublished(page, limit int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.Article{}).
		Where("status IN ? AND is_active = ?", publishedStatuses, true)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Author").Preload("Category").
		Order("published_at DESC").
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// GetByCategoryID lấy bài viết theo danh mục
func (r *ArticleRepo) GetByCategoryID(categoryID uuid.UUID, page, limit int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.Article{}).Where("category_id = ?", categoryID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Author").Preload("Category").
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// GetFeatured lấy bài viết nổi bật
func (r *ArticleRepo) GetFeatured(limit int) ([]model.Article, error) {
	var articles []model.Article
	err := r.db.Preload("Author").Preload("Category").
		Where("status IN ? AND is_active = ? AND is_hot = ?", publishedStatuses, true, true).
		Order("published_at DESC").
		Order("view_count DESC").
		Order("created_at DESC").
		Limit(limit).Find(&articles).Error
	if err != nil {
		return nil, err
	}
	return articles, nil
}

// GetPublishedBySlug lấy bài viết public theo slug
func (r *ArticleRepo) GetPublishedBySlug(slug string) (*model.Article, error) {
	var article model.Article
	err := r.db.Preload("Author").Preload("Category").
		Where("slug = ? AND status IN ? AND is_active = ?", slug, publishedStatuses, true).
		First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, err
	}
	return &article, nil
}

// GetPublishedByCategorySlug lấy bài viết public theo slug danh mục
func (r *ArticleRepo) GetPublishedByCategorySlug(slug string, page, limit int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.Article{}).
		Joins("JOIN categories ON categories.id = articles.category_id").
		Where("categories.slug = ? AND categories.is_active = ?", slug, true).
		Where("articles.status IN ? AND articles.is_active = ?", publishedStatuses, true)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Author").Preload("Category").
		Order("articles.published_at DESC").
		Order("articles.created_at DESC").
		Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// GetPublishedByTagID lấy bài viết public theo tag ID
func (r *ArticleRepo) GetPublishedByTagID(tagID uuid.UUID, page, limit int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * limit

	jsonContainsValue := fmt.Sprintf("\"%s\"", tagID.String())
	query := r.db.Model(&model.Article{}).
		Where("JSON_CONTAINS(tag_id, ?, '$')", jsonContainsValue).
		Where("status IN ? AND is_active = ?", publishedStatuses, true)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Author").Preload("Category").
		Order("published_at DESC").
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// Search tìm kiếm bài viết
func (r *ArticleRepo) Search(keyword string, page, limit int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.Article{}).
		Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Author").Preload("Category").
		Order("created_at DESC").
		Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// Update cập nhật bài viết
func (r *ArticleRepo) Update(article *model.Article) error {
	return r.db.Save(article).Error
}

// Delete xóa bài viết
func (r *ArticleRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Article{}, "id = ?", id).Error
}

// IncrementViewCount tăng lượt xem
func (r *ArticleRepo) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// CheckSlugExists kiểm tra slug đã tồn tại chưa
func (r *ArticleRepo) CheckSlugExists(slug string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.Article{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}
