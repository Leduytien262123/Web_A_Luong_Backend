package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Review struct {
	ID        uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	ProductID uuid.UUID      `json:"product_id" gorm:"type:char(36);not null;index"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:char(36);not null;index"`
	Rating    int            `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Title     string         `json:"title" gorm:"size:200"`
	Comment   string         `json:"comment" gorm:"type:text"`
	IsActive  bool           `json:"is_active" gorm:"default:true;index"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Product *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User    *User    `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// BeforeCreate hook để tự động tạo UUID cho Review
func (r *Review) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}

type Coupon struct {
	ID               uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Code             string         `json:"code" gorm:"unique;not null;size:50;index"`
	Name             string         `json:"name" gorm:"not null;size:200"`
	Description      string         `json:"description" gorm:"size:500"`
	Type             string         `json:"type" gorm:"not null;size:20;check:type IN ('percentage', 'fixed')"`
	Value            float64        `json:"value" gorm:"not null;type:decimal(10,2)"`
	MinOrderAmount   float64        `json:"min_order_amount" gorm:"type:decimal(10,2);default:0"`
	MaxDiscountValue float64        `json:"max_discount_value" gorm:"type:decimal(10,2)"`
	UsageLimit       int            `json:"usage_limit" gorm:"default:0"`
	UsedCount        int            `json:"used_count" gorm:"default:0"`
	IsActive         bool           `json:"is_active" gorm:"default:true;index"`
	StartDate        time.Time      `json:"start_date" gorm:"not null"`
	EndDate          time.Time      `json:"end_date" gorm:"not null"`
	CreatedAt        time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate hook để tự động tạo UUID cho Coupon
func (c *Coupon) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

type Address struct {
	ID           uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	UserID       uuid.UUID      `json:"user_id" gorm:"type:char(36);not null;index"`
	Name         string         `json:"name" gorm:"not null;size:100"`
	Phone        string         `json:"phone" gorm:"not null;size:20"`
	AddressLine1 string         `json:"address_line1" gorm:"not null;size:200"`
	AddressLine2 string         `json:"address_line2" gorm:"size:200"`
	City         string         `json:"city" gorm:"not null;size:100"`
	State        string         `json:"state" gorm:"not null;size:100"`
	PostalCode   string         `json:"postal_code" gorm:"not null;size:20"`
	Country      string         `json:"country" gorm:"not null;size:100;default:Vietnam"`
	IsDefault    bool           `json:"is_default" gorm:"default:false;index"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// BeforeCreate hook để tự động tạo UUID cho Address
func (a *Address) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}

type Brand struct {
	ID          uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Name        string         `json:"name" gorm:"not null;size:100;index"`
	Slug        string         `json:"slug" gorm:"unique;not null;size:100;index"`
	Description string         `json:"description" gorm:"size:500"`
	LogoURL     string         `json:"logo_url" gorm:"size:500"`
	Website     string         `json:"website" gorm:"size:200"`
	IsActive    bool           `json:"is_active" gorm:"default:true;index"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Products []Product `json:"products,omitempty" gorm:"foreignKey:BrandID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

// BeforeCreate hook để tự động tạo UUID cho Brand
func (b *Brand) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return
}

type ProductImage struct {
	ID        uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	ProductID uuid.UUID      `json:"product_id" gorm:"type:char(36);not null;index"`
	ImageURL  string         `json:"image_url" gorm:"not null;size:500"`
	AltText   string         `json:"alt_text" gorm:"size:200"`
	IsPrimary bool           `json:"is_primary" gorm:"default:false;index"`
	SortOrder int            `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Product *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// BeforeCreate hook để tự động tạo UUID cho ProductImage
func (pi *ProductImage) BeforeCreate(tx *gorm.DB) (err error) {
	if pi.ID == uuid.Nil {
		pi.ID = uuid.New()
	}
	return
}

// TableName specifies the table names
func (Review) TableName() string { return "reviews" }
func (Coupon) TableName() string { return "coupons" }
func (Address) TableName() string { return "addresses" }
func (Brand) TableName() string { return "brands" }
func (ProductImage) TableName() string { return "product_images" }

// Input structs
type ReviewInput struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Rating    int       `json:"rating" binding:"required,min=1,max=5"`
	Title     string    `json:"title" binding:"max=200"`
	Comment   string    `json:"comment" binding:"max=1000"`
}

type CouponInput struct {
	Code             string    `json:"code" binding:"required,min=1,max=50"`
	Name             string    `json:"name" binding:"required,min=1,max=200"`
	Description      string    `json:"description" binding:"max=500"`
	Type             string    `json:"type" binding:"required,oneof=percentage fixed"`
	Value            float64   `json:"value" binding:"required,gt=0"`
	MinOrderAmount   float64   `json:"min_order_amount" binding:"gte=0"`
	MaxDiscountValue float64   `json:"max_discount_value" binding:"gte=0"`
	UsageLimit       int       `json:"usage_limit" binding:"gte=0"`
	StartDate        time.Time `json:"start_date" binding:"required"`
	EndDate          time.Time `json:"end_date" binding:"required"`
}

type AddressInput struct {
	Name         string `json:"name" binding:"required,min=1,max=100"`
	Phone        string `json:"phone" binding:"required,min=1,max=20"`
	AddressLine1 string `json:"address_line1" binding:"required,min=1,max=200"`
	AddressLine2 string `json:"address_line2" binding:"max=200"`
	City         string `json:"city" binding:"required,min=1,max=100"`
	State        string `json:"state" binding:"required,min=1,max=100"`
	PostalCode   string `json:"postal_code" binding:"required,min=1,max=20"`
	Country      string `json:"country" binding:"max=100"`
	IsDefault    bool   `json:"is_default"`
}

type BrandInput struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Slug        string `json:"slug" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	LogoURL     string `json:"logo_url" binding:"max=500"`
	Website     string `json:"website" binding:"max=200"`
}

type ProductImageInput struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	ImageURL  string    `json:"image_url" binding:"required,max=500"`
	AltText   string    `json:"alt_text" binding:"max=200"`
	IsPrimary bool      `json:"is_primary"`
	SortOrder int       `json:"sort_order"`
}

// Response structs
type ReviewResponse struct {
	ID        uuid.UUID     `json:"id"`
	ProductID uuid.UUID     `json:"product_id"`
	UserID    uuid.UUID     `json:"user_id"`
	User      *UserResponse `json:"user,omitempty"`
	Rating    int           `json:"rating"`
	Title     string        `json:"title"`
	Comment   string        `json:"comment"`
	IsActive  bool          `json:"is_active"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type CouponResponse struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Type             string    `json:"type"`
	Value            float64   `json:"value"`
	MinOrderAmount   float64   `json:"min_order_amount"`
	MaxDiscountValue float64   `json:"max_discount_value"`
	UsageLimit       int       `json:"usage_limit"`
	UsedCount        int       `json:"used_count"`
	IsActive         bool      `json:"is_active"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AddressResponse struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	AddressLine1 string    `json:"address_line1"`
	AddressLine2 string    `json:"address_line2"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	PostalCode   string    `json:"postal_code"`
	Country      string    `json:"country"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BrandResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	LogoURL     string    `json:"logo_url"`
	Website     string    `json:"website"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProductImageResponse struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	ImageURL  string    `json:"image_url"`
	AltText   string    `json:"alt_text"`
	IsPrimary bool      `json:"is_primary"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Dashboard Statistics Models

// DashboardOverviewResponse - Tổng quan dashboard
type DashboardOverviewResponse struct {
	TotalRevenue       float64 `json:"total_revenue"`
	RevenueGrowth      float64 `json:"revenue_growth"`       // % tăng/giảm so với kỳ trước
	TotalOrders        int     `json:"total_orders"`
	OrdersGrowth       float64 `json:"orders_growth"`        // % tăng/giảm so với kỳ trước
	TotalProducts      int     `json:"total_products_sold"`  // Tổng số sản phẩm đã bán
	ProductsGrowth     float64 `json:"products_growth"`      // % tăng/giảm so với kỳ trước
	TotalCustomers     int     `json:"total_customers"`
	CustomersGrowth    float64 `json:"customers_growth"`     // % tăng/giảm so với kỳ trước
	PendingOrders      int     `json:"pending_orders"`       // Đơn hàng đang chờ xử lý
	LowStockProducts   int     `json:"low_stock_products"`   // Số sản phẩm sắp hết hàng
	AverageOrderValue  float64 `json:"average_order_value"`  // Giá trị đơn hàng trung bình
}

// DashboardFullOverview - API 1: Tổng hợp tất cả dữ liệu overview (thay thế 5 API)
type DashboardFullOverview struct {
	Summary           DashboardOverviewResponse  `json:"summary"`
	RevenueChart      []RevenueByTimeResponse    `json:"revenue_chart"`
	OrderStatusChart  []OrderStatusStatistics    `json:"order_status_chart"`
	TopProducts       []ProductStatistics        `json:"top_products"`
	TopCategories     []CategoryStatistics       `json:"top_categories"`
	RecentActivities  []RecentActivity           `json:"recent_activities"`
}

// DashboardAnalytics - API 2: Phân tích chi tiết (thay thế 4 API)
type DashboardAnalytics struct {
	CategoryStats        []CategoryStatistics        `json:"category_stats"`
	ProductStats         []ProductStatistics         `json:"product_stats"`
	PaymentMethodStats   []PaymentMethodStatistics   `json:"payment_method_stats"`
	OrderTypeStats       []OrderTypeStatistics       `json:"order_type_stats"`
	TopCustomers         []CustomerStatistics        `json:"top_customers"`
	RevenueByTime        []RevenueByTimeResponse     `json:"revenue_by_time"`
}

// DashboardAlerts - API 3: Cảnh báo và hoạt động (thay thế 2 API)
type DashboardAlerts struct {
	LowStockProducts  []LowStockProduct      `json:"low_stock_products"`
	PendingOrders     int                    `json:"pending_orders"`
	NewCustomers      []CustomerStatistics   `json:"new_customers"`
	RecentActivities  []RecentActivity       `json:"recent_activities"`
	CriticalAlerts    int                    `json:"critical_alerts"`
	WarningAlerts     int                    `json:"warning_alerts"`
}

// RevenueByTimeResponse - Doanh thu theo thời gian
type RevenueByTimeResponse struct {
	Period string  `json:"period"` // YYYY-MM-DD hoặc YYYY-MM hoặc YYYY
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
	Products int    `json:"products_sold"` // Số sản phẩm đã bán
}

// CategoryStatistics - Thống kê theo danh mục
type CategoryStatistics struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	ProductsSold int     `json:"products_sold"`    // Tổng số sản phẩm đã bán
	Revenue      float64 `json:"revenue"`          // Doanh thu
	Orders       int     `json:"orders"`           // Số đơn hàng
	Percentage   float64 `json:"percentage"`       // % so với tổng doanh thu
}

// ProductStatistics - Thống kê sản phẩm
type ProductStatistics struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	SKU          string  `json:"sku"`
	QuantitySold int     `json:"quantity_sold"`
	Revenue      float64 `json:"revenue"`
	Stock        int     `json:"stock"`
	Rating       float64 `json:"rating"`
	ReviewCount  int     `json:"review_count"`
}

// OrderStatusStatistics - Thống kê trạng thái đơn hàng
type OrderStatusStatistics struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Percentage float64 `json:"percentage"`
}

// PaymentMethodStatistics - Thống kê phương thức thanh toán
type PaymentMethodStatistics struct {
	Method string  `json:"method"`
	Count  int     `json:"count"`
	Revenue float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

// OrderTypeStatistics - Thống kê loại đơn hàng
type OrderTypeStatistics struct {
	Type   string  `json:"type"`
	Count  int     `json:"count"`
	Revenue float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

// CustomerStatistics - Thống kê khách hàng
type CustomerStatistics struct {
	UserID          string  `json:"user_id"`
	FullName        string  `json:"full_name"`
	Email           string  `json:"email"`
	Phone           string  `json:"phone"`
	TotalOrders     int     `json:"total_orders"`
	TotalSpent      float64 `json:"total_spent"`
	LastOrderDate   string  `json:"last_order_date"`
	AverageOrderValue float64 `json:"average_order_value"`
}

// LowStockProduct - Sản phẩm sắp hết hàng
type LowStockProduct struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	SKU         string `json:"sku"`
	CurrentStock int   `json:"current_stock"`
	MinStock    int    `json:"min_stock"`
	Status      string `json:"status"` // "critical", "low", "normal"
}

// RecentActivity - Hoạt động gần đây
type RecentActivity struct {
	Type        string `json:"type"` // "order", "review", "user"
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
	UserName    string `json:"user_name"`
	Amount      float64 `json:"amount,omitempty"`
}

// ToResponse methods
func (r *Review) ToResponse() ReviewResponse {
	response := ReviewResponse{
		ID:        r.ID,
		ProductID: r.ProductID,
		UserID:    r.UserID,
		Rating:    r.Rating,
		Title:     r.Title,
		Comment:   r.Comment,
		IsActive:  r.IsActive,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	if r.User != nil {
		userResponse := r.User.ToResponse()
		response.User = &userResponse
	}
	return response
}

func (c *Coupon) ToResponse() CouponResponse {
	return CouponResponse{
		ID:               c.ID,
		Code:             c.Code,
		Name:             c.Name,
		Description:      c.Description,
		Type:             c.Type,
		Value:            c.Value,
		MinOrderAmount:   c.MinOrderAmount,
		MaxDiscountValue: c.MaxDiscountValue,
		UsageLimit:       c.UsageLimit,
		UsedCount:        c.UsedCount,
		IsActive:         c.IsActive,
		StartDate:        c.StartDate,
		EndDate:          c.EndDate,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func (a *Address) ToResponse() AddressResponse {
	return AddressResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		Name:         a.Name,
		Phone:        a.Phone,
		AddressLine1: a.AddressLine1,
		AddressLine2: a.AddressLine2,
		City:         a.City,
		State:        a.State,
		PostalCode:   a.PostalCode,
		Country:      a.Country,
		IsDefault:    a.IsDefault,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

func (b *Brand) ToResponse() BrandResponse {
	return BrandResponse{
		ID:          b.ID,
		Name:        b.Name,
		Slug:        b.Slug,
		Description: b.Description,
		LogoURL:     b.LogoURL,
		Website:     b.Website,
		IsActive:    b.IsActive,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func (pi *ProductImage) ToResponse() ProductImageResponse {
	return ProductImageResponse{
		ID:        pi.ID,
		ProductID: pi.ProductID,
		ImageURL:  pi.ImageURL,
		AltText:   pi.AltText,
		IsPrimary: pi.IsPrimary,
		SortOrder: pi.SortOrder,
		CreatedAt: pi.CreatedAt,
		UpdatedAt: pi.UpdatedAt,
	}
}