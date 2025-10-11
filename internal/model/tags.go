package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Tag struct {
	ID           uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Name         string         `json:"name" gorm:"not null;size:200;index"`
	Slug         string         `json:"slug" gorm:"unique;not null;size:200;index"`
	Description  string         `json:"description" gorm:"type:text"`
	DisplayOrder int            `json:"display_order" gorm:"default:0;index"`
	IsActive     bool           `json:"is_active" gorm:"default:true;index"`
	UsageCount   int            `json:"usage_count" gorm:"default:0;index"`
	Metadata     datatypes.JSON `json:"metadata" gorm:"type:json"`
	Content      datatypes.JSON `json:"content" gorm:"type:json"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Products []Product `json:"products,omitempty" gorm:"many2many:product_tags;"`
	News     []News    `json:"news,omitempty" gorm:"many2many:news_tags;"`
}

// BeforeCreate hook để tự động tạo UUID
func (t *Tag) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

// TableName chỉ định tên bảng
func (Tag) TableName() string {
	return "tags"
}

// Metadata and Content structs
type TagMetadata struct {
	MetaTitle       string           `json:"meta_title"`
	MetaDescription string           `json:"meta_description"`
	MetaImage       []MetaImageTag   `json:"meta_image"`
	MetaKeywords    string           `json:"meta_keywords"`
}

type MetaImageTag struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type TagContent struct {
	CoverPhoto  []CoverPhotoTag `json:"cover_photo"`
	Images      []ImageTag      `json:"images"`
	Description string          `json:"description"`
	Content     string          `json:"content"`
}

type CoverPhotoTag struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ImageTag struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

// Input structs
type TagInput struct {
	Name         string       `json:"name" binding:"required,min=1,max=200"`
	Slug         string       `json:"slug" binding:"required,min=1,max=200"`
	Color        string       `json:"color" binding:"omitempty,len=7"`
	Description  string       `json:"description" binding:"max=1000"`
	DisplayOrder *int         `json:"display_order,omitempty"`
	IsActive     *bool        `json:"is_active,omitempty"`
	Metadata     *TagMetadata `json:"metadata,omitempty"`
	Content      *TagContent  `json:"content,omitempty"`
}

type TagUpdateInput struct {
	Name         string       `json:"name" binding:"required,min=1,max=200"`
	Slug         string       `json:"slug" binding:"required,min=1,max=200"`
	Color        string       `json:"color" binding:"omitempty,len=7"`
	Description  string       `json:"description" binding:"max=1000"`
	DisplayOrder *int         `json:"display_order,omitempty"`
	IsActive     *bool        `json:"is_active,omitempty"`
	Metadata     *TagMetadata `json:"metadata,omitempty"`
	Content      *TagContent  `json:"content,omitempty"`
}

// Response structs
type TagResponse struct {
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Slug         string       `json:"slug"`
	Description  string       `json:"description"`
	DisplayOrder int          `json:"display_order"`
	IsActive     bool         `json:"is_active"`
	UsageCount   int          `json:"usage_count"`
	Metadata     *TagMetadata `json:"metadata"`
	Content      *TagContent  `json:"content"`
	NewsCount    int          `json:"news_count,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type TagShortResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// ToResponse chuyển Tag thành TagResponse
func (t *Tag) ToResponse() TagResponse {
	resp := TagResponse{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		Description:  t.Description,
		DisplayOrder: t.DisplayOrder,
		IsActive:     t.IsActive,
		UsageCount:   t.UsageCount,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}

	// Parse metadata từ JSON sang struct
	if len(t.Metadata) > 0 {
		var metadata TagMetadata
		if err := json.Unmarshal(t.Metadata, &metadata); err == nil {
			resp.Metadata = &metadata
		}
	}

	// Parse content từ JSON sang struct
	if len(t.Content) > 0 {
		var content TagContent
		if err := json.Unmarshal(t.Content, &content); err == nil {
			resp.Content = &content
		}
	}

	if len(t.News) > 0 {
		resp.NewsCount = len(t.News)
	}

	return resp
}

// ToShortResponse chuyển Tag thành TagShortResponse
func (t *Tag) ToShortResponse() TagShortResponse {
	return TagShortResponse{
		ID:   t.ID,
		Name: t.Name,
		Slug: t.Slug,
	}
}

// ProductTag - Bảng trung gian cho Product và Tag
type ProductTag struct {
	ProductID uuid.UUID `gorm:"type:char(36);primaryKey"`
	TagID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (ProductTag) TableName() string { return "product_tags" }

// NewsTag - Bảng trung gian cho News và Tag
type NewsTag struct {
	NewsID    uuid.UUID `gorm:"type:char(36);primaryKey"`
	TagID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (NewsTag) TableName() string { return "news_tags" }