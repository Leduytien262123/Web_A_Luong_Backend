package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Category struct {
	ID           uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Name         string         `json:"name" gorm:"not null;size:255;index"`
	Description  string         `json:"description" gorm:"type:text"`
	Slug         string         `json:"slug" gorm:"unique;not null;size:255;index"`
	IsActive     bool           `json:"is_active" gorm:"default:true;index"`
	DisplayOrder int            `json:"display_order" gorm:"default:0;index"`
	ShowOnMenu   bool           `json:"show_menu" gorm:"default:false"`
	ShowOnHome   bool           `json:"show_home" gorm:"default:false"`
	ShowOnFooter bool           `json:"show_footer" gorm:"default:false"`
	PositionMenu int            `json:"position_menu" gorm:"default:0;index"`
	PositionFooter int          `json:"position_footer" gorm:"default:0;index"`
	PositionHome int            `json:"position_home" gorm:"default:0;index"`
	Metadata     datatypes.JSON `json:"metadata" gorm:"type:json"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	ParentID *uuid.UUID `json:"parent_id,omitempty" gorm:"type:char(36);index"`
	Parent   *Category  `json:"parent_category,omitempty" gorm:"foreignKey:ParentID"`

	// Quan hệ (constraints handled manually in database.go)
	Articles []Article `json:"articles,omitempty" gorm:"foreignKey:CategoryID"`
}

type CategoryMetadata struct {
	MetaTitle       string              `json:"meta_title"`
	MetaDescription string              `json:"meta_description"`
	MetaImage       []MetaImageCategory `json:"meta_image"` // Đổi thành array
	MetaKeywords    string              `json:"meta_keywords"`
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
	Name         string            `json:"name" binding:"required,min=1,max=100"`
	Description  string            `json:"description" binding:"max=500"`
	Slug         string            `json:"slug" binding:"required,min=1,max=100"`
	DisplayOrder *int              `json:"display_order,omitempty"`
	IsActive     *bool             `json:"is_active,omitempty"`
	ShowOnMenu   *bool             `json:"show_menu,omitempty"`
	ShowOnHome   *bool             `json:"show_home,omitempty"`
	ShowOnFooter *bool             `json:"show_footer,omitempty"`
	PositionMenu *int              `json:"position_menu,omitempty"`
	PositionFooter *int            `json:"position_footer,omitempty"`
	PositionHome *int              `json:"position_home,omitempty"`
	Metadata     *CategoryMetadata `json:"metadata,omitempty"`
	ParentCategory *string         `json:"parent_category,omitempty"`
}

type CategoryResponse struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Slug         string            `json:"slug"`
	DisplayOrder int               `json:"display_order"`
	IsActive     bool              `json:"is_active"`
	ShowOnMenu   bool              `json:"show_menu"`
	ShowOnHome   bool              `json:"show_home"`
	ShowOnFooter bool              `json:"show_footer"`
	PositionMenu int               `json:"position_menu"`
	PositionFooter int             `json:"position_footer"`
	PositionHome int               `json:"position_home"`
	Metadata     *CategoryMetadata `json:"metadata"` // Đổi từ datatypes.JSON sang *CategoryMetadata
	Articles     []ArticleSummary  `json:"articles,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ParentCategory *struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"parent_category,omitempty"`
	Children     []CategoryResponse `json:"children,omitempty"`
}

// ArticleSummary là dữ liệu bài viết trả về gọn nhẹ khi đính kèm trong danh mục
type ArticleSummary struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Metadata    datatypes.JSON `json:"metadata,omitempty"`
	Content     datatypes.JSON `json:"content,omitempty"`
}

// ToResponse chuyển Category thành CategoryResponse
func (c *Category) ToResponse() CategoryResponse {
	response := CategoryResponse{
		ID:           c.ID,
		Name:         c.Name,
		Description:  c.Description,
		Slug:         c.Slug,
		DisplayOrder: c.DisplayOrder,
		IsActive:     c.IsActive,
		ShowOnMenu:   c.ShowOnMenu,
		ShowOnHome:   c.ShowOnHome,
		ShowOnFooter: c.ShowOnFooter,
		PositionMenu: c.PositionMenu,
		PositionFooter: c.PositionFooter,
		PositionHome: c.PositionHome,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}

	// Parse metadata từ JSON sang struct
	if len(c.Metadata) > 0 {
		var metadata CategoryMetadata
		if err := json.Unmarshal(c.Metadata, &metadata); err == nil {
			response.Metadata = &metadata
		}
	}

	if len(c.Articles) > 0 {
		articles := make([]ArticleSummary, 0, len(c.Articles))
		for _, article := range c.Articles {
			articles = append(articles, ArticleSummary{
				ID:          article.ID,
				Title:       article.Title,
				Slug:        article.Slug,
				Description: article.Description,
				Status:      article.Status,
				PublishedAt: article.PublishedAt,
				CreatedAt:   article.CreatedAt,
				UpdatedAt:   article.UpdatedAt,
				Metadata:    article.Metadata,
				Content:     article.Content,
			})
		}
		response.Articles = articles
	}

	// Parent category info
	if c.Parent != nil {
		response.ParentCategory = &struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		}{
			ID:   c.Parent.ID,
			Name: c.Parent.Name,
		}
	} else if c.ParentID != nil && *c.ParentID != uuid.Nil {
		response.ParentCategory = &struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		}{
			ID:   *c.ParentID,
			Name: "",
		}
	}

	return response
}

// BuildCategoryTree xây dựng cấu trúc cây từ danh sách categories
func BuildCategoryTree(categories []Category) []CategoryResponse {
	// Tạo map để tra cứu nhanh
	categoryMap := make(map[uuid.UUID]*CategoryResponse)
	var rootCategories []CategoryResponse

	// Chuyển đổi tất cả categories sang response và lưu vào map
	for _, cat := range categories {
		response := cat.ToResponse()
		response.Children = []CategoryResponse{}
		categoryMap[cat.ID] = &response
	}

	// Xây dựng cây phân cấp
	for _, cat := range categories {
		if cat.ParentID != nil && *cat.ParentID != uuid.Nil {
			// Có parent - thêm vào children của parent
			if parent, exists := categoryMap[*cat.ParentID]; exists {
				if parent.Children == nil {
					parent.Children = []CategoryResponse{}
				}
				parent.Children = append(parent.Children, *categoryMap[cat.ID])
			}
		} else {
			// Không có parent - là root category
			rootCategories = append(rootCategories, *categoryMap[cat.ID])
		}
	}

	return rootCategories
}
