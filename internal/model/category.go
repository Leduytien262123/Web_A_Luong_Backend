package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	Name        string         `json:"name" gorm:"not null;size:255;index"`
	Description string         `json:"description" gorm:"type:text"`
	Slug        string         `json:"slug" gorm:"unique;not null;size:255;index"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	ShowOnMenu  bool           `json:"show_menu" gorm:"default:false"`
	ShowOnHome  bool           `json:"show_home" gorm:"default:false"`
	ShowOnFooter bool         `json:"show_footer" gorm:"default:false"`
	Metadata    datatypes.JSON `json:"metadata" gorm:"type:json"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Quan hệ
	Products []Product `json:"products,omitempty" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

type CategoryMetadata struct {
	MetaTitle       string    `json:"meta_title"`
	MetaDescription string    `json:"meta_description"`
	MetaImage       MetaImageCategory `json:"meta_image"`
	MetaKeywords    string    `json:"meta_keywords"`
}

type MetaImageCategory struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

// TableName chỉ định tên bảng cho model Category
func (Category) TableName() string {
	return "categories"
}

// BeforeCreate hook để tự động tạo UUID cho Category
func (c *Category) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

type CategoryInput struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	Slug        string `json:"slug" binding:"required,min=1,max=100"`
	IsActive    *bool  `json:"is_active,omitempty"`
	ShowOnMenu  *bool  `json:"show_menu,omitempty"`
	ShowOnHome  *bool  `json:"show_home,omitempty"`
	ShowOnFooter *bool  `json:"show_footer,omitempty"`
	Metadata    *CategoryMetadata `json:"metadata,omitempty"`
}

type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Slug        string    `json:"slug"`
	IsActive    bool      `json:"is_active"`
	ShowOnMenu  bool      `json:"show_menu"`
	ShowOnHome  bool      `json:"show_home"`
	ShowOnFooter bool     `json:"show_footer"`
	Metadata    datatypes.JSON `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse chuyển Category thành CategoryResponse
func (c *Category) ToResponse() CategoryResponse {
	return CategoryResponse{
		ID:          c.ID,   
		Name:        c.Name,
		Description: c.Description,
		Slug:        c.Slug,
		IsActive:    c.IsActive,
		ShowOnMenu:  c.ShowOnMenu,
		ShowOnHome:  c.ShowOnHome,
		ShowOnFooter: c.ShowOnFooter,
		Metadata:     c.Metadata,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
