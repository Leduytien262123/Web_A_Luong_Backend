package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string         `json:"username" gorm:"unique;not null;size:50;index"`
	Email     string         `json:"email" gorm:"unique;not null;size:100;index"`
	Password  string         `json:"-" gorm:"not null;size:255"`
	FullName  string         `json:"full_name" gorm:"size:100"`
	Avatar    string         `json:"avatar" gorm:"size:500"`
	Role      string         `json:"role" gorm:"default:user;size:20"`
	IsActive  bool           `json:"is_active" gorm:"default:true;index"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName chỉ định tên bảng cho model User
func (User) TableName() string {
	return "users"
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
	UserID     uint   `json:"user_id" binding:"required"`
	Permission string `json:"permission" binding:"required,oneof=full write read none"`
	Resource   string `json:"resource" binding:"required"`
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
	ID        uint      `json:"id"`
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
