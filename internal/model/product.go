package model

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Name        string         `json:"name" gorm:"not null;size:255;index"`
	Slug        string         `json:"slug" gorm:"unique;not null;size:255;index"`
	Description string         `json:"description" gorm:"type:text"`
	Price       float64        `json:"price" gorm:"type:decimal(10,2);not null;index"`
	SalePrice   *float64       `json:"sale_price" gorm:"type:decimal(10,2);index"`
	SKU         string         `json:"sku" gorm:"unique;not null;size:100;index"`
	Weight      *float64       `json:"weight" gorm:"type:decimal(8,3)"` // kg
	Dimensions  string         `json:"dimensions" gorm:"size:100"`      // e.g., "10x20x30 cm"
	Stock       int            `json:"stock" gorm:"not null;default:0;index"`
	MinStock    int            `json:"min_stock" gorm:"default:5"`
	MaxStock    int            `json:"max_stock" gorm:"default:1000"`
	Status      string         `json:"status" gorm:"not null;default:active;size:20;index"`
	IsDigital   bool           `json:"is_digital" gorm:"default:false"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	IsFeatured  bool           `json:"is_featured" gorm:"default:false;index"`
	MetaTitle   string         `json:"meta_title" gorm:"size:255"`
	MetaDesc    string         `json:"meta_description" gorm:"size:500"`
	Views       int            `json:"views" gorm:"default:0;index"`
	Sales       int            `json:"sales" gorm:"default:0;index"`
	Rating      float64        `json:"rating" gorm:"type:decimal(3,2);default:0;index"`
	ReviewCount int            `json:"review_count" gorm:"default:0"`
	Metadata    datatypes.JSON `json:"metadata" gorm:"type:json"` // Thêm field Metadata
	Content     datatypes.JSON `json:"content" gorm:"type:json"`  // Thêm field Content
	
	// Foreign Keys
	CategoryID *uuid.UUID `json:"category_id" gorm:"type:char(36);index"`
	BrandID    *uuid.UUID `json:"brand_id" gorm:"type:char(36);index"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

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

// BeforeCreate hook để tự động tạo UUID và slug cho Product
func (p *Product) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	
	// Tự động tạo slug nếu chưa có
	if p.Slug == "" && p.Name != "" {
		p.Slug = generateUniqueSlug(tx, p.Name, p.ID.String())
	}
	
	return
}

// generateUniqueSlug tạo slug unique từ name
func generateUniqueSlug(tx *gorm.DB, name, id string) string {
	// Chuyển name thành slug cơ bản
	baseSlug := strings.ToLower(name)
	baseSlug = strings.ReplaceAll(baseSlug, " ", "-")
	baseSlug = strings.ReplaceAll(baseSlug, "&", "and")
	baseSlug = strings.ReplaceAll(baseSlug, ".", "")
	baseSlug = strings.ReplaceAll(baseSlug, "/", "-")
	baseSlug = strings.ReplaceAll(baseSlug, "_", "-")
	
	// Remove multiple consecutive dashes
	for strings.Contains(baseSlug, "--") {
		baseSlug = strings.ReplaceAll(baseSlug, "--", "-")
	}
	
	// Trim dashes from start and end
	baseSlug = strings.Trim(baseSlug, "-")
	
	// Nếu slug rỗng hoặc quá ngắn, dùng fallback
	if len(baseSlug) < 2 {
		baseSlug = "product"
	}
	
	// Kiểm tra xem slug đã tồn tại chưa
	var count int64
	tx.Model(&Product{}).Where("slug = ?", baseSlug).Count(&count)
	
	// Nếu đã tồn tại, thêm suffix unique
	if count > 0 {
		baseSlug = baseSlug + "-" + id[:8]
	}
	
	return baseSlug
}

type ProductMetadata struct {
	MetaTitle       string             `json:"meta_title"`
	MetaDescription string             `json:"meta_description"`
	MetaImage       []MetaImageProduct `json:"meta_image"`
	MetaKeywords    string             `json:"meta_keywords"`
}

type MetaImageProduct struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ProductContent struct {
	CoverPhoto  []CoverPhotoProduct `json:"cover_photo"`
	Images      []ImageProduct      `json:"images"`
	Description string              `json:"description"`
	Content     string              `json:"content"`
}

type CoverPhotoProduct struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ImageProduct struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ProductInput struct {
	Name          string           `json:"name" binding:"required,min=1,max=200"`
	Description   string           `json:"description" binding:"max=1000"`
	Price         float64          `json:"price" binding:"required,gt=0"`
	DiscountPrice *float64         `json:"discount_price,omitempty" binding:"omitempty,gt=0,ltefield=Price"`
	SKU           string           `json:"sku" binding:"required,min=1,max=50"`
	Stock         int              `json:"stock" binding:"gte=0"`
	CategoryID    *uuid.UUID       `json:"category_id"`
	BrandID       *uuid.UUID       `json:"brand_id"`
	Material      string           `json:"material" binding:"max=100"`
	Color         string           `json:"color" binding:"max=50"`
	Size          string           `json:"size" binding:"max=50"`
	Weight        float64          `json:"weight" binding:"gte=0"`
	Dimensions    string           `json:"dimensions" binding:"max=100"`
	IsFeatured    bool             `json:"is_featured"`
	IsActive      bool             `json:"is_active"`
	Metadata      *ProductMetadata `json:"metadata,omitempty"`
	Content       *ProductContent  `json:"content,omitempty"` // Thêm Content
}

// CategoryShortResponse chỉ trả về id và name
type CategoryShortResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
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
	Metadata      *ProductMetadata       `json:"metadata"`
	Content       *ProductContent        `json:"content"` // Thêm Content
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
		IsFeatured:  p.IsFeatured,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	// Handle Weight pointer
	if p.Weight != nil {
		response.Weight = *p.Weight
	}

	// Nếu có giá giảm (khác 0) thì gán pointer, ngược lại để nil
	if p.SalePrice != nil && *p.SalePrice > 0 {
		response.DiscountPrice = p.SalePrice
	}

	// Parse metadata từ JSON sang struct
	if len(p.Metadata) > 0 {
		var metadata ProductMetadata
		if err := json.Unmarshal(p.Metadata, &metadata); err == nil {
			response.Metadata = &metadata
		}
	}

	// Parse content từ JSON sang struct
	if len(p.Content) > 0 {
		var content ProductContent
		if err := json.Unmarshal(p.Content, &content); err == nil {
			response.Content = &content
		}
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
	} else {
		response.AverageRating = p.Rating
		response.ReviewCount = p.ReviewCount
	}

	return response
}
