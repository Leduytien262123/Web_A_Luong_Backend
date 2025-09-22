package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID      `json:"id" gorm:"type:char(36);primary_key"`
	Username          string         `json:"username" gorm:"unique;not null;size:50;index"`
	FullName          string         `json:"full_name" gorm:"not null;size:255"`
	Email             string         `json:"email" gorm:"unique;not null;size:255;index"`
	Password          string         `json:"-" gorm:"not null;size:255"`
	Phone             string         `json:"phone" gorm:"size:20;index"`
	DateOfBirth       *time.Time     `json:"date_of_birth"`
	Gender            string         `json:"gender" gorm:"size:10"`
	Avatar            string         `json:"avatar" gorm:"size:500"`
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

	// Relationships
	Addresses []Address `json:"addresses,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Reviews   []Review  `json:"reviews,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Orders    []Order   `json:"orders,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Cart      *Cart     `json:"cart,omitempty" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	News      []News    `json:"news,omitempty" gorm:"foreignKey:AuthorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
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
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	FullName string `json:"full_name" binding:"max=100"`
	Avatar   string `json:"avatar" binding:"max=500"`
}

type CreateUserInput struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=100"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	FullName string `json:"full_name" binding:"max=100"`
	Avatar   string `json:"avatar" binding:"max=500"`
	Role     string `json:"role" binding:"required,oneof=admin member"`
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
	FullName string `json:"full_name" binding:"max=100"`
	Email    string `json:"email" binding:"email,max=100"`
	Avatar   string `json:"avatar" binding:"max=500"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Avatar    string    `json:"avatar"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FullName:  u.FullName,
		Avatar:    u.Avatar,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
