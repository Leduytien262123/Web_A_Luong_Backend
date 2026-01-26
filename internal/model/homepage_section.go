package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HomepageSection - Các mục hiển thị ở trang chủ
type HomepageSection struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Title       string         `json:"title" gorm:"not null;size:255;index"`
	Description string         `json:"description" gorm:"type:text"`
	TypeKey     string         `json:"type_key" gorm:"type:varchar(50);not null;uniqueIndex"` // TYPE01, TYPE02, etc.
	Metadata    datatypes.JSON `json:"metadata" gorm:"type:json"`                             // JSON array lưu dữ liệu items
	Position    int            `json:"position" gorm:"default:0;index"`                       // Vị trí hiển thị
	ShowHome    bool           `json:"show_home" gorm:"default:true;index"`                   // Hiển thị ở trang chủ
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (HomepageSection) TableName() string {
	return "homepage_sections"
}

func (h *HomepageSection) BeforeCreate(tx *gorm.DB) (err error) {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return
}

// HomepageSectionInput - Input cho tạo/cập nhật section
type HomepageSectionInput struct {
	Title       string          `json:"title" binding:"required,min=1,max=255"`
	Description string          `json:"description"`
	TypeKey     string          `json:"type_key" binding:"required,min=1,max=50"`
	Metadata    json.RawMessage `json:"metadata"`
	Position    *int            `json:"position"`
	ShowHome    *bool           `json:"show_home"`
}

// HomepageSectionResponse - Response khi trả về section
type HomepageSectionResponse struct {
	ID          uuid.UUID       `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	TypeKey     string          `json:"type_key"`
	Metadata    json.RawMessage `json:"metadata"`
	Position    int             `json:"position"`
	ShowHome    bool            `json:"show_home"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// HomepageSectionPublicResponse - Response công khai cho người dùng
type HomepageSectionPublicResponse struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	TypeKey     string          `json:"type_key"`
	Metadata    json.RawMessage `json:"metadata"`
}

// ToResponse chuyển đổi từ model sang response
func (h *HomepageSection) ToResponse() HomepageSectionResponse {
	return HomepageSectionResponse{
		ID:          h.ID,
		Title:       h.Title,
		Description: h.Description,
		TypeKey:     h.TypeKey,
		Metadata:    json.RawMessage(h.Metadata),
		Position:    h.Position,
		ShowHome:    h.ShowHome,
		CreatedAt:   h.CreatedAt,
		UpdatedAt:   h.UpdatedAt,
	}
}

// ToPublicResponse chuyển đổi từ model sang response công khai
func (h *HomepageSection) ToPublicResponse() HomepageSectionPublicResponse {
	return HomepageSectionPublicResponse{
		Title:       h.Title,
		Description: h.Description,
		TypeKey:     h.TypeKey,
		Metadata:    json.RawMessage(h.Metadata),
	}
}
