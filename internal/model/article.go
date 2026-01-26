package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Article - Bài viết
type Article struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Title       string         `json:"title" gorm:"not null;size:500;index"`
	Description string         `json:"description" gorm:"type:text"`
	Slug        string         `json:"slug" gorm:"unique;not null;size:500;index"`
	CategoryID  *uuid.UUID     `json:"category_id" gorm:"type:char(36);index"`
	TagIDs      datatypes.JSON `json:"tag_ids" gorm:"type:json;column:tag_id"` // Mảng UUID của tags
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	IsHot       bool           `json:"is_hot" gorm:"default:false;index"`
	Status      string         `json:"status" gorm:"type:varchar(20);default:'draft';index"` // draft, post
	PublishedAt *time.Time     `json:"published_at"`
	Metadata    datatypes.JSON `json:"metadata" gorm:"type:json"`
	Content     datatypes.JSON `json:"content" gorm:"type:json"`
	AuthorID    uuid.UUID      `json:"author_id" gorm:"type:char(36);not null;index"` // Admin tạo bài
	ViewCount   int            `json:"view_count" gorm:"default:0;index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Quan hệ (constraints handled manually in database.go)
	Author   *User     `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Category *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
}

func (Article) TableName() string {
	return "articles"
}

func (a *Article) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

type ArticleInput struct {
	Title       string          `json:"title" binding:"required,min=1,max=500"`
	Description string          `json:"description"`
	Slug        string          `json:"slug" binding:"required,min=1,max=500"`
	CategoryID  *uuid.UUID      `json:"category_id"`
	TagIDs      []uuid.UUID     `json:"tag_ids"` // Đổi tên từ tag_id thành tag_ids cho nhất quán
	IsActive    *bool           `json:"is_active"`
	IsHot       *bool           `json:"is_hot"`
	Status      *string         `json:"status"`
	PublishedAt *time.Time      `json:"published_at"`
	Metadata    json.RawMessage `json:"metadata"`
	Content     json.RawMessage `json:"content"`
}

type AuthorResponse struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

type CategorySimpleResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type ArticleResponse struct {
	ID          uuid.UUID               `json:"id"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Slug        string                  `json:"slug"`
	CategoryID  *uuid.UUID              `json:"category_id"`
	TagIDs      []uuid.UUID             `json:"tag_ids"` // Đổi tên từ tag_id thành tag_ids
	TagNames    []string                `json:"tag_names,omitempty"`
	IsActive    bool                    `json:"is_active"`
	IsHot       bool                    `json:"is_hot"`
	Status      string                  `json:"status"`
	PublishedAt *time.Time              `json:"published_at"`
	Metadata    json.RawMessage         `json:"metadata,omitempty"`
	Content     json.RawMessage         `json:"content,omitempty"`
	AuthorID    uuid.UUID               `json:"author_id"`
	ViewCount   int                     `json:"view_count"`
	Author      *AuthorResponse         `json:"author,omitempty"`
	Category    *CategorySimpleResponse `json:"category,omitempty"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// GetTagIDs trả về danh sách UUID của tags từ JSON
func (a *Article) GetTagIDs() []uuid.UUID {
	if len(a.TagIDs) == 0 {
		return []uuid.UUID{}
	}

	var ids []uuid.UUID
	if err := json.Unmarshal(a.TagIDs, &ids); err != nil {
		return []uuid.UUID{}
	}

	return ids
}

// SetTagIDs thiết lập danh sách tag IDs từ slice UUID
func (a *Article) SetTagIDs(tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		a.TagIDs = datatypes.JSON([]byte("[]"))
		return nil
	}

	bytes, err := json.Marshal(tagIDs)
	if err != nil {
		return err
	}

	a.TagIDs = datatypes.JSON(bytes)
	return nil
}

func (a *Article) ToResponse() ArticleResponse {
	response := ArticleResponse{
		ID:          a.ID,
		Title:       a.Title,
		Description: a.Description,
		Slug:        a.Slug,
		CategoryID:  a.CategoryID,
		TagIDs:      a.GetTagIDs(), // Sử dụng method mới
		IsActive:    a.IsActive,
		IsHot:       a.IsHot,
		Status:      a.Status,
		PublishedAt: a.PublishedAt,
		AuthorID:    a.AuthorID,
		ViewCount:   a.ViewCount,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}

	// Preserve metadata/content as-is
	if len(a.Metadata) > 0 {
		response.Metadata = json.RawMessage(a.Metadata)
	}

	if len(a.Content) > 0 {
		response.Content = json.RawMessage(a.Content)
	}

	// Include author info
	if a.Author != nil {
		response.Author = &AuthorResponse{
			ID:       a.Author.ID,
			FullName: a.Author.FullName,
			Email:    a.Author.Email,
		}
	}

	// Include category info
	if a.Category != nil {
		response.Category = &CategorySimpleResponse{
			ID:   a.Category.ID,
			Name: a.Category.Name,
			Slug: a.Category.Slug,
		}
	}
	return response
}
