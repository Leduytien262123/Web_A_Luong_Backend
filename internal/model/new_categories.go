package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NewsCategory struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	Name        string         `json:"name" gorm:"not null;size:200;index"`
	Slug        string         `json:"slug" gorm:"unique;not null;size:200;index"`
	Description string         `json:"description" gorm:"type:text"`
	ImageURL    string         `json:"image_url" gorm:"size:500"`
	ParentID    *uuid.UUID     `json:"parent_id" gorm:"type:char(36);index"`
	SortOrder   int            `json:"sort_order" gorm:"default:0;index"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	MetaTitle   string         `json:"meta_title" gorm:"size:255"`
	MetaDesc    string         `json:"meta_desc" gorm:"size:500"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Quan hệ
	Parent   *NewsCategory   `json:"parent,omitempty" gorm:"foreignKey:ParentID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Children []NewsCategory  `json:"children,omitempty" gorm:"foreignKey:ParentID"`
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

type NewsCategoryInput struct {
	Name        string     `json:"name" binding:"required,min=1,max=200"`
	Slug        string     `json:"slug" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"max=1000"`
	ImageURL    string     `json:"image_url" binding:"max=500"`
	ParentID    *uuid.UUID `json:"parent_id"`
	SortOrder   int        `json:"sort_order"`
	IsActive    bool       `json:"is_active"`
	MetaTitle   string     `json:"meta_title" binding:"max=255"`
	MetaDesc    string     `json:"meta_desc" binding:"max=500"`
}

type NewsCategoryResponse struct {
	ID          uuid.UUID               `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	ImageURL    string                  `json:"image_url"`
	ParentID    *uuid.UUID              `json:"parent_id"`
	SortOrder   int                     `json:"sort_order"`
	IsActive    bool                    `json:"is_active"`
	MetaTitle   string                  `json:"meta_title"`
	MetaDesc    string                  `json:"meta_desc"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	Parent      *NewsCategoryResponse   `json:"parent,omitempty"`
	Children    []NewsCategoryResponse  `json:"children,omitempty"`
	NewsCount   int                     `json:"news_count,omitempty"`
}

// ToResponse chuyển NewsCategory thành NewsCategoryResponse
func (nc *NewsCategory) ToResponse() NewsCategoryResponse {
	response := NewsCategoryResponse{
		ID:          nc.ID,
		Name:        nc.Name,
		Slug:        nc.Slug,
		Description: nc.Description,
		ImageURL:    nc.ImageURL,
		ParentID:    nc.ParentID,
		SortOrder:   nc.SortOrder,
		IsActive:    nc.IsActive,
		MetaTitle:   nc.MetaTitle,
		MetaDesc:    nc.MetaDesc,
		CreatedAt:   nc.CreatedAt,
		UpdatedAt:   nc.UpdatedAt,
	}

	// Include parent if loaded
	if nc.Parent != nil {
		parent := nc.Parent.ToResponse()
		response.Parent = &parent
	}

	// Include children if loaded
	if len(nc.Children) > 0 {
		for _, child := range nc.Children {
			response.Children = append(response.Children, child.ToResponse())
		}
	}

	// Count news if loaded
	if len(nc.News) > 0 {
		response.NewsCount = len(nc.News)
	}

	return response
}