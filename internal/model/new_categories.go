package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NewsCategory struct {
	ID          uuid.UUID      		`json:"id" gorm:"type:char(36);primaryKey"`
	Name        string         		`json:"name" gorm:"not null;size:255;index"`
	Slug        string         		`json:"slug" gorm:"unique;not null;size:255;index"`
	Description string         		`json:"description" gorm:"type:text"`
	ImageURL    string         		`json:"image_url" gorm:"size:500"`
	ParentID    *uuid.UUID     		`json:"parent_id" gorm:"type:char(36);index"`
	SortOrder   int            		`json:"sort_order" gorm:"default:0;index"`
	IsActive    bool           		`json:"is_active" gorm:"default:true;index"`
	Metadata    datatypes.JSON 		`json:"metadata" gorm:"type:json"`
	Content     datatypes.JSON 		`json:"content" gorm:"type:json"`
	
	CreatedAt   time.Time      		`json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      		`json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt 		`json:"-" gorm:"index"`

	// Quan hệ
	News     []News          `json:"news,omitempty" gorm:"many2many:news_category_associations;"`
}

// BeforeCreate hook để tự động tạo UUID
func (nc *NewsCategory) BeforeCreate(tx *gorm.DB) (err error) {
	if nc.ID == uuid.Nil {
		nc.ID = uuid.New()
	}
	return
}

// TableName chỉ định tên bảng
func (NewsCategory) TableName() string {
	return "news_categories"
}

type NewsCategoryMetadata struct {
	MetaTitle       string                   `json:"meta_title"`
	MetaDescription string                   `json:"meta_description"`
	MetaImage       []MetaImageNewsCategory  `json:"meta_image"`
	MetaKeywords    string                   `json:"meta_keywords"`
}

type MetaImageNewsCategory struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type NewsCategoryContent struct {
	CoverPhoto  []CoverPhotoNewsCategory `json:"cover_photo"`
	Images      []ImageNewsCategory      `json:"images"`
	Description string                   `json:"description"`
	Content     string                   `json:"content"`
}

type CoverPhotoNewsCategory struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ImageNewsCategory struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type NewsCategoryInput struct {
	Name         string                  `json:"name" binding:"required,min=1,max=200"`
	Slug         string                  `json:"slug" binding:"required,min=1,max=200"`
	DisplayOrder int                     `json:"display_order"`
	IsActive     bool                    `json:"is_active"`
	Metadata     *NewsCategoryMetadata   `json:"metadata"`
	Content      *NewsCategoryContent    `json:"content"`
}

type NewsCategoryResponse struct {
	ID          uuid.UUID               `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	DisplayOrder int                     `json:"display_order"`
	IsActive    bool                    `json:"is_active"`
	Metadata    *NewsCategoryMetadata   `json:"metadata"`
	Content     *NewsCategoryContent    `json:"content"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	NewsCount   int                     `json:"news_count,omitempty"`
}

// ToResponse chuyển NewsCategory thành NewsCategoryResponse
func (nc *NewsCategory) ToResponse() NewsCategoryResponse {
	response := NewsCategoryResponse{
		ID:          nc.ID,
		Name:        nc.Name,
		Slug:        nc.Slug,
		DisplayOrder: nc.SortOrder,
		IsActive:    nc.IsActive,
		CreatedAt:   nc.CreatedAt,
		UpdatedAt:   nc.UpdatedAt,
	}

	// Parse metadata từ JSON sang struct
	if len(nc.Metadata) > 0 {
		var metadata NewsCategoryMetadata
		if err := json.Unmarshal(nc.Metadata, &metadata); err == nil {
			response.Metadata = &metadata
		}
	}

	// Parse content từ JSON sang struct
	if len(nc.Content) > 0 {
		var content NewsCategoryContent
		if err := json.Unmarshal(nc.Content, &content); err == nil {
			response.Content = &content
		}
	}

	// Count news if loaded
	if len(nc.News) > 0 {
		response.NewsCount = len(nc.News)
	}

	return response
}