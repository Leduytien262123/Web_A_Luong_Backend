package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID      `json:"id" gorm:"type:char(36);primaryKey"`
	Username          string         `json:"username" gorm:"unique;not null;size:50;index"`
	FullName          string         `json:"full_name" gorm:"not null;size:255"`
	Email             string         `json:"email" gorm:"unique;not null;size:255;index"`
	Password          string         `json:"-" gorm:"not null;size:255"`
	Phone             string         `json:"phone" gorm:"size:20;index"`
	DateOfBirth       *time.Time     `json:"date_of_birth"`
	Gender            string         `json:"gender" gorm:"size:10"`
	Avatar            datatypes.JSON  `json:"avatar" gorm:"type:json"`
	IsActive          bool           `json:"is_active" gorm:"default:true;index"`
	IsEmailVerified   bool           `json:"is_email_verified" gorm:"default:false"`
	EmailVerifiedAt   *time.Time     `json:"email_verified_at"`
	PasswordChangedAt *time.Time     `json:"password_changed_at"`
	LastLoginAt       *time.Time     `json:"last_login_at"`
	LoginAttempts     int            `json:"login_attempts" gorm:"default:0"`
	LockedUntil       *time.Time     `json:"locked_until"`
	Role              string         `json:"role" gorm:"not null;default:user;size:20;index"`
	
	// Thêm các trường thống kê đơn hàng
	TotalOrders       int            `json:"total_orders" gorm:"default:0"`         // Tổng số đơn hàng
	CompletedOrders   int            `json:"completed_orders" gorm:"default:0"`     // Số đơn hàng hoàn thành
	TotalSpent        float64        `json:"total_spent" gorm:"type:decimal(15,2);default:0"` // Tổng tiền đã chi tiêu
	LastOrderAt       *time.Time     `json:"last_order_at"`                         // Thời gian đơn hàng gần nhất
	
	CreatedAt         time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Quan hệ
	Addresses []Address `json:"addresses,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Orders    []Order   `json:"orders,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Reviews   []Review  `json:"reviews,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Cart      *Cart     `json:"cart,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	News      []News    `json:"news,omitempty" gorm:"foreignKey:CreatorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

type Avatar struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

// TableName chỉ định tên bảng cho model User
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook để tự động tạo UUID
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}

type UserInput struct {
	Username string  `json:"username" binding:"required,min=3,max=50"`
	Email    string  `json:"email" binding:"required,email,max=100"`
	Password string  `json:"password" binding:"required,min=6,max=100"`
	FullName string  `json:"full_name" binding:"max=100"`
	Avatar   []Avatar `json:"avatar"`
}

type CreateUserInput struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email,max=100"`
	Password string   `json:"password" binding:"required,min=6,max=100"`
	FullName string   `json:"full_name" binding:"max=100"`
	Avatar   []Avatar `json:"avatar"`
	Role     string   `json:"role" binding:"required,oneof=admin member"`
	Addresses []string `json:"addresses"` // Thêm trường này để nhận mảng địa chỉ từ FE
}

type UpdateUserRoleInput struct {
	Role string `json:"role" binding:"required,oneof=owner admin member user"`
}

type AssignPermissionInput struct {
	UserID     uuid.UUID `json:"user_id" binding:"required"`
	Permission string    `json:"permission" binding:"required,oneof=full write read none"`
	Resource   string    `json:"resource" binding:"required"`
}

type UpdateUserInput struct {
	FullName string   `json:"full_name" binding:"max=100"`
	Email    string   `json:"email" binding:"email,max=100"`
	Avatar   []Avatar `json:"avatar"`
}

type AdminUserUpdateInput struct {
    Role      string   `json:"role" binding:"required,oneof=owner admin user member"`
    CreatorID uuid.UUID `json:"creator_id" binding:"required"`
    FullName  string   `json:"full_name" binding:"max=100"`
    Phone     string   `json:"phone" binding:"max=20"`
    Email     string   `json:"email" binding:"required,email,max=100"`
    Addresses []string `json:"addresses"`
    Avatar    []Avatar `json:"avatar"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// SimpleAddressResponse cho địa chỉ đơn giản (deprecated - sử dụng AddressResponse thay thế)
type SimpleAddressResponse struct {
	Address string `json:"address"`
}

type UserResponse struct {
	ID                uuid.UUID           `json:"id"`
	Username          string              `json:"username"`
	Email             string              `json:"email"`
	Phone             string              `json:"phone"`
	FullName          string              `json:"full_name"`
	Addresses         []map[string]string `json:"addresses"`  // Trả về đúng dạng FE yêu cầu
	Avatar            []Avatar            `json:"avatar"`
	Role              string              `json:"role"`
	IsActive          bool                `json:"is_active"`
	TotalOrders       int                 `json:"total_orders"`
	CompletedOrders   int                 `json:"completed_orders"`
	TotalSpent        float64             `json:"total_spent"`
	LastOrderAt       *time.Time          `json:"last_order_at"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

func (u *User) ToResponse() UserResponse {
	response := UserResponse{
		ID:              u.ID,
		Username:        u.Username,
		Email:           u.Email,
		Phone:           u.Phone,
		FullName:        u.FullName,
		Avatar:          []Avatar{},
		Role:            u.Role,
		IsActive:        u.IsActive,
		TotalOrders:     u.TotalOrders,
		CompletedOrders: u.CompletedOrders,
		TotalSpent:      u.TotalSpent,
		LastOrderAt:     u.LastOrderAt,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}

	// Giải mã avatar (lưu trữ dưới dạng JSON array trong DB) thành []Avatar
	if len(u.Avatar) > 0 {
		var avs []Avatar
		if err := json.Unmarshal(u.Avatar, &avs); err == nil {
			response.Avatar = avs
		}
	}

	if len(u.Addresses) > 0 {
		var addresses []map[string]string
		for _, addr := range u.Addresses {
			addresses = append(addresses, map[string]string{"address": addr.AddressLine1})
		}
		response.Addresses = addresses
	}

	return response
}
