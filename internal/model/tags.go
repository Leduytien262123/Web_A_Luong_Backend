package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tag struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	Name        string         `json:"name" gorm:"not null;size:100;index"`
	Slug        string         `json:"slug" gorm:"unique;not null;size:100;index"`
	Color       string         `json:"color" gorm:"size:7;default:#007bff"` // Hex color
	Description string         `json:"description" gorm:"type:text"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	UsageCount  int            `json:"usage_count" gorm:"default:0;index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Quan hệ many-to-many
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

type TagInput struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Slug        string `json:"slug" binding:"required,min=1,max=100"`
	Color       string `json:"color" binding:"omitempty,len=7"`
	Description string `json:"description" binding:"max=500"`
	IsActive    bool   `json:"is_active"`
}

type TagResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	UsageCount  int       `json:"usage_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse chuyển Tag thành TagResponse
func (t *Tag) ToResponse() TagResponse {
	return TagResponse{
		ID:          t.ID,
		Name:        t.Name,
		Slug:        t.Slug,
		Color:       t.Color,
		Description: t.Description,
		IsActive:    t.IsActive,
		UsageCount:  t.UsageCount,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// ProductTag - Bảng trung gian cho Product và Tag
type ProductTag struct {
	ProductID uuid.UUID `gorm:"type:char(36);primaryKey"`
	TagID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// NewsTag - Bảng trung gian cho News và Tag
type NewsTag struct {
	NewsID    uuid.UUID `gorm:"type:char(36);primaryKey"`
	TagID     uuid.UUID `gorm:"type:char(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}