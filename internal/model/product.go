package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	Name        string         `json:"name" gorm:"not null;size:255;index"`
	Description string         `json:"description" gorm:"type:longtext"`
	Price       float64        `json:"price" gorm:"not null;type:decimal(10,2);index"`
	DiscountPrice float64      `json:"discount_price" gorm:"type:decimal(10,2);index"`
	SKU         string         `json:"sku" gorm:"unique;not null;size:100;index"`
	Stock       int            `json:"stock" gorm:"not null;default:0;index"`
	CategoryID  *uuid.UUID     `json:"category_id" gorm:"type:char(36);index"`
	BrandID     *uuid.UUID     `json:"brand_id" gorm:"type:char(36);index"`
	Material    string         `json:"material" gorm:"size:100"`
	Color       string         `json:"color" gorm:"size:50"`
	Size        string         `json:"size" gorm:"size:50"`
	Weight      float64        `json:"weight" gorm:"type:decimal(8,2)"`
	Dimensions  string         `json:"dimensions" gorm:"size:100"`
	Metadata    datatypes.JSON `json:"metadata" gorm:"type:json"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	IsFeatured  bool           `json:"is_featured" gorm:"default:false;index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Quan hệ
	Category      *Category       `json:"category,omitempty" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Brand         *Brand          `json:"brand,omitempty" gorm:"foreignKey:BrandID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	ProductImages []ProductImage  `json:"product_images,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Reviews       []Review        `json:"reviews,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName chỉ định tên bảng cho model Product
func (Product) TableName() string {
	return "products"
}

// BeforeCreate hook để tự động tạo UUID cho Product
func (p *Product) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}

type ProductInput struct {
	Name        string     `json:"name" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"max=1000"`
	Price       float64    `json:"price" binding:"required,gt=0"`
	DiscountPrice *float64   `json:"discount_price,omitempty" binding:"omitempty,gt=0,ltefield=Price"`
	SKU         string     `json:"sku" binding:"required,min=1,max=50"`
	Stock       int        `json:"stock" binding:"gte=0"`
	CategoryID  *uuid.UUID `json:"category_id"`
	BrandID     *uuid.UUID `json:"brand_id"`
	Material    string     `json:"material" binding:"max=100"`
	Color       string     `json:"color" binding:"max=50"`
	Size        string     `json:"size" binding:"max=50"`
	Weight      float64    `json:"weight" binding:"gte=0"`
	Dimensions  string     `json:"dimensions" binding:"max=100"`
	IsFeatured  bool       `json:"is_featured"`
	Metadata    *ProductMetadata `json:"metadata,omitempty"`
}

// CategoryShortResponse chỉ trả về id và name
type CategoryShortResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProductMetadata struct {
	MetaTitle       string    `json:"meta_title"`
	MetaDescription string    `json:"meta_description"`
	MetaImage       MetaImageProduct `json:"meta_image"`
	MetaKeywords    string    `json:"meta_keywords"`
}

type MetaImageProduct struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ProductResponse struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Price         float64                `json:"price"`
	DiscountPrice *float64               `json:"discount_price,omitempty"`
	SKU           string                 `json:"sku"`
	Stock         int                    `json:"stock"`
	CategoryID    *uuid.UUID             `json:"category_id"`
	Category      *CategoryShortResponse  `json:"category,omitempty"`
	BrandID       *uuid.UUID             `json:"brand_id"`
	Brand         *BrandResponse         `json:"brand,omitempty"`
	Material      string                 `json:"material"`
	Color         string                 `json:"color"`
	Size          string                 `json:"size"`
	Weight        float64                `json:"weight"`
	Dimensions    string                 `json:"dimensions"`
	IsActive      bool                   `json:"is_active"`
	IsFeatured    bool                   `json:"is_featured"`
	Metadata      datatypes.JSON         `json:"metadata" gorm:"type:json"`
	ProductImages []ProductImageResponse `json:"product_images,omitempty"`
	AverageRating float64                `json:"average_rating"`
	ReviewCount   int                    `json:"review_count"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ToResponse chuyển Product thành ProductResponse
func (p *Product) ToResponse() ProductResponse {
	response := ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		SKU:         p.SKU,
		Stock:       p.Stock,
		CategoryID:  p.CategoryID,
		BrandID:     p.BrandID,
		Material:    p.Material,
		Color:       p.Color,
		Size:        p.Size,
		Weight:      p.Weight,
		Dimensions:  p.Dimensions,
		IsActive:    p.IsActive,
		Metadata:    p.Metadata,
		IsFeatured:  p.IsFeatured,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	// Nếu có giá giảm (khác 0) thì gán pointer, ngược lại để nil
	if p.DiscountPrice > 0 {
		dp := p.DiscountPrice
		response.DiscountPrice = &dp
	}

	// Bao gồm thông tin danh mục nếu đã được nạp
	if p.Category != nil {
		response.Category = &CategoryShortResponse{
			ID:   p.Category.ID,
			Name: p.Category.Name,
		}
	}

	// Bao gồm thông tin thương hiệu nếu đã được nạp
	if p.Brand != nil {
		brandResponse := p.Brand.ToResponse()
		response.Brand = &brandResponse
	}

	// Bao gồm danh sách ảnh sản phẩm nếu đã được nạp
	if len(p.ProductImages) > 0 {
		for _, img := range p.ProductImages {
			response.ProductImages = append(response.ProductImages, img.ToResponse())
		}
	}

	// Tính điểm đánh giá trung bình và số lượng đánh giá
	if len(p.Reviews) > 0 {
		totalRating := 0.0
		activeReviews := 0
		for _, review := range p.Reviews {
			if review.IsActive {
				totalRating += float64(review.Rating)
				activeReviews++
			}
		}
		if activeReviews > 0 {
			response.AverageRating = totalRating / float64(activeReviews)
		}
		response.ReviewCount = activeReviews
	}

	return response
}
