package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type News struct {
	ID           uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Title        string         `json:"title" gorm:"not null;size:255;index"`
	Slug         string         `json:"slug" gorm:"unique;not null;size:255;index"`
	Summary      string         `json:"summary" gorm:"type:text"`
	ImageURL     string         `json:"image_url" gorm:"size:500"`
	CreatorID    uuid.UUID      `json:"creator_id" gorm:"type:char(36);not null;index"`
	CategoryID   *uuid.UUID     `json:"category_id" gorm:"type:char(36);index"`
	IsPublished  bool           `json:"is_published" gorm:"default:false;index"`
	IsFeatured   bool           `json:"is_featured" gorm:"default:false;index"` // Tin nổi bật
	PublishedAt  *time.Time     `json:"published_at"`
	ViewCount    int            `json:"view_count" gorm:"default:0;index"`
	LikeCount    int            `json:"like_count" gorm:"default:0;index"`
	CommentCount int            `json:"comment_count" gorm:"default:0;index"`
	Metadata     datatypes.JSON `json:"metadata" gorm:"type:json"`
	Content      datatypes.JSON `json:"content" gorm:"type:json"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Quan hệ
	Creator    *User         `json:"creator,omitempty" gorm:"foreignKey:CreatorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Category   *NewsCategory `json:"category,omitempty" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Tags       []Tag         `json:"tags,omitempty" gorm:"many2many:news_tags;"`
	Categories []NewsCategory `json:"categories,omitempty" gorm:"many2many:news_category_associations;"`
}

// BeforeCreate hook để tự động tạo UUID cho News
func (n *News) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return
}

// TableName chỉ định tên bảng cho model News
func (News) TableName() string {
	return "news"
}

type NewsInput struct {
	Title       string                 `json:"title" binding:"required,min=1,max=200"`
	Slug        string                 `json:"slug" binding:"required,min=1,max=200"`
	CategoryID  *uuid.UUID             `json:"category_id"`
	TagID       *uuid.UUID             `json:"tag_id"`
	Status      string                 `json:"status" binding:"required,oneof=post draft"`
	PublishedAt *time.Time             `json:"published_at"`
	Description string                 `json:"description" binding:"max=500"`
	Metadata    *NewsMetadata          `json:"metadata"`
	Content     *NewsContent           `json:"content"`
}

type NewsMetadata struct {
	MetaTitle       string           `json:"meta_title"`
	MetaKeywords    string           `json:"meta_keywords"`
	MetaDescription string           `json:"meta_description"`
	MetaImage       []MetaImageNews  `json:"meta_image"` // Đổi thành array
}

type MetaImageNews struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type NewsContent struct {
	CoverPhoto  []CoverPhotoNews `json:"cover_photo"`  
	Images      []ImageNews      `json:"images"`       
	Description string           `json:"description"`
	Content     string           `json:"content"` // Thêm trường content
}

type CoverPhotoNews struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ImageNews struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type CreatorResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type NewsCategorySimpleResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type NewsResponse struct {
	ID           uuid.UUID                   `json:"id"`
	Title        string                      `json:"title"`
	Slug         string                      `json:"slug"`
	CategoryID   *uuid.UUID                  `json:"category_id"`
	TagID        *uuid.UUID                  `json:"tag_id"`
	Status       string                      `json:"status"`
	PublishedAt  *time.Time                  `json:"published_at"`
	Description  string                      `json:"description"`
	Metadata     *NewsMetadata               `json:"metadata"`
	Content      *NewsContent                `json:"content"`
	Creator      *CreatorResponse            `json:"creator,omitempty"`
	Category     *NewsCategorySimpleResponse `json:"category,omitempty"`
	Tags         []TagResponse               `json:"tags,omitempty"`
	ViewCount    int                         `json:"view_count"`
	LikeCount    int                         `json:"like_count"`
	CommentCount int                         `json:"comment_count"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
}

// ToResponse chuyển News thành NewsResponse
func (n *News) ToResponse() NewsResponse {
	// Convert status from IsPublished boolean to string
	status := "draft"
	if n.IsPublished {
		status = "post"
	}

	// Parse metadata JSON
	var metadata *NewsMetadata
	if len(n.Metadata) > 0 {
		var meta NewsMetadata
		if err := json.Unmarshal(n.Metadata, &meta); err == nil {
			metadata = &meta
		}
	}

	// Parse content JSON
	var content *NewsContent
	if len(n.Content) > 0 {
		var cont NewsContent
		if err := json.Unmarshal(n.Content, &cont); err == nil {
			content = &cont
		}
	}

	// Get tag_id from first tag if exists
	var tagID *uuid.UUID
	if len(n.Tags) > 0 {
		tagID = &n.Tags[0].ID
	}

	response := NewsResponse{
		ID:           n.ID,
		Title:        n.Title,
		Slug:         n.Slug,
		CategoryID:   n.CategoryID,
		TagID:        tagID,
		Status:       status,
		PublishedAt:  n.PublishedAt,
		Description:  n.Summary,
		Metadata:     metadata,
		Content:      content,
		ViewCount:    n.ViewCount,
		LikeCount:    n.LikeCount,
		CommentCount: n.CommentCount,
		CreatedAt:    n.CreatedAt,
		UpdatedAt:    n.UpdatedAt,
	}

	// Bao gồm thông tin creator đơn giản nếu đã được nạp
	if n.Creator != nil {
		response.Creator = &CreatorResponse{
			ID:   n.Creator.ID,
			Name: n.Creator.FullName,
		}
	}

	// Include main category đơn giản if loaded
	if n.Category != nil {
		response.Category = &NewsCategorySimpleResponse{
			ID:   n.Category.ID,
			Name: n.Category.Name,
		}
	}

	// Include tags if loaded
	if len(n.Tags) > 0 {
		for _, tag := range n.Tags {
			response.Tags = append(response.Tags, tag.ToResponse())
		}
	}

	return response
}

// NewsCategory association table
type NewsCategoryAssociation struct {
	NewsID     uuid.UUID `gorm:"type:char(36);not null"`
	CategoryID uuid.UUID `gorm:"type:char(36);not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// TableName specifies the table name
func (NewsCategoryAssociation) TableName() string {
	return "news_category_associations"
}