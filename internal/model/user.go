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
	Avatar            datatypes.JSON `json:"avatar" gorm:"type:json"`
	IsActive          bool           `json:"is_active" gorm:"default:true;index"`
	IsEmailVerified   bool           `json:"is_email_verified" gorm:"default:false"`
	EmailVerifiedAt   *time.Time     `json:"email_verified_at"`
	PasswordChangedAt *time.Time     `json:"password_changed_at"`
	LastLoginAt       *time.Time     `json:"last_login_at"`
	LoginAttempts     int            `json:"login_attempts" gorm:"default:0"`
	LockedUntil       *time.Time     `json:"locked_until"`
	Role              string         `json:"role" gorm:"not null;default:admin;size:20;index"` // super_admin hoặc admin
	
	CreatedAt         time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Quan hệ (constraints handled manually in database.go)
	Articles []Article `json:"articles,omitempty" gorm:"foreignKey:AuthorID"`
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
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email,max=100"`
	Password string   `json:"password" binding:"required,min=6,max=100"`
	FullName string   `json:"full_name" binding:"max=100"`
	Avatar   []Avatar `json:"avatar"`
}

type CreateUserInput struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Email    string   `json:"email" binding:"required,email,max=100"`
	Password string   `json:"password" binding:"required,min=6,max=100"`
	FullName string   `json:"full_name" binding:"max=100"`
	Avatar   []Avatar `json:"avatar"`
	Role     string   `json:"role" binding:"required,oneof=super_admin admin"`
}

type UpdateUserRoleInput struct {
	Role string `json:"role" binding:"required,oneof=super_admin admin"`
}

type UpdateUserInput struct {
	FullName string   `json:"full_name" binding:"max=100"`
	Email    string   `json:"email" binding:"email,max=100"`
	Phone    string   `json:"phone" binding:"max=20"`
	Avatar   []Avatar `json:"avatar"`
}

type AdminUserUpdateInput struct {
	Role     string   `json:"role" binding:"required,oneof=super_admin admin"`
	FullName string   `json:"full_name" binding:"max=100"`
	Phone    string   `json:"phone" binding:"max=20"`
	Email    string   `json:"email" binding:"required,email,max=100"`
	Avatar   []Avatar `json:"avatar"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	FullName        string    `json:"full_name"`
	Avatar          []Avatar  `json:"avatar"`
	Role            string    `json:"role"`
	IsActive        bool      `json:"is_active"`
	ArticleCount    int       `json:"article_count"` // Số bài viết đã tạo
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (u *User) ToResponse() UserResponse {
	response := UserResponse{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		Phone:        u.Phone,
		FullName:     u.FullName,
		Avatar:       []Avatar{},
		Role:         u.Role,
		IsActive:     u.IsActive,
		ArticleCount: len(u.Articles),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}

	// Giải mã avatar
	if len(u.Avatar) > 0 {
		var avs []Avatar
		if err := json.Unmarshal(u.Avatar, &avs); err == nil {
			response.Avatar = avs
		}
	}

	return response
}
