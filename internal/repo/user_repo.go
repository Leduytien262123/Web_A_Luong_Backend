package repo

import (
	"backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetUserByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateUser(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) DeleteUser(id uuid.UUID) error {
	return r.db.Delete(&model.User{}, "id = ?", id).Error
}

func (r *UserRepository) GetAllUsers() ([]model.User, error) {
	var users []model.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *UserRepository) GetUsersByRolesWithPagination(roles []string, name string, phone string, email string, page, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	db := r.db.Model(&model.User{})
	
	if len(roles) > 0 {
		db = db.Where("role IN ?", roles)
	}
	if name != "" {
		db = db.Where("full_name LIKE ?", "%"+name+"%")
	}
	if phone != "" {
		db = db.Where("phone LIKE ?", "%"+phone+"%")
	}
	if email != "" {
		db = db.Where("email LIKE ?", "%"+email+"%")
	}
	
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * limit
	err := db.Order("CASE WHEN role = 'super_admin' THEN 0 WHEN role = 'admin' THEN 1 ELSE 2 END, created_at DESC").
		Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

func (r *UserRepository) IsUsernameExists(username string) bool {
	var count int64
	r.db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	return count > 0
}

func (r *UserRepository) IsEmailExists(email string) bool {
	var count int64
	r.db.Model(&model.User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

// CheckSuperAdminExists kiểm tra tài khoản super admin đã tồn tại hay chưa
func (r *UserRepository) CheckSuperAdminExists() (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("role = ?", "super_admin").Count(&count).Error
	return count > 0, err
}

// GetUsersByRole lấy người dùng theo vai trò, có phân trang
func (r *UserRepository) GetUsersByRole(role string, page, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if err := r.db.Model(&model.User{}).Where("role = ?", role).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	err := r.db.Where("role = ?", role).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error

	return users, total, err
}

// UpdateUserRole cập nhật vai trò người dùng
func (r *UserRepository) UpdateUserRole(userID uuid.UUID, newRole string) error {
	return r.db.Model(&model.User{}).Where("id = ?", userID).Update("role", newRole).Error
}

// GetUserStats lấy thống kê người dùng theo vai trò
func (r *UserRepository) GetUserStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Đếm người dùng theo từng vai trò
	var roleCounts []struct {
		Role  string `json:"role"`
		Count int64  `json:"count"`
	}
	if err := r.db.Model(&model.User{}).
		Select("role, COUNT(*) as count").
		Group("role").
		Find(&roleCounts).Error; err != nil {
		return nil, err
	}
	stats["users_by_role"] = roleCounts

	// Tổng số người dùng đang hoạt động
	var activeUsers int64
	if err := r.db.Model(&model.User{}).Where("is_active = ?", true).Count(&activeUsers).Error; err != nil {
		return nil, err
	}
	stats["active_users"] = activeUsers

	// Tổng số người dùng không hoạt động
	var inactiveUsers int64
	if err := r.db.Model(&model.User{}).Where("is_active = ?", false).Count(&inactiveUsers).Error; err != nil {
		return nil, err
	}
	stats["inactive_users"] = inactiveUsers

	// Tổng số admin
	var adminCount int64
	if err := r.db.Model(&model.User{}).Where("role = ?", "admin").Count(&adminCount).Error; err != nil {
		return nil, err
	}
	stats["total_admins"] = adminCount

	// Tổng số super admin
	var superAdminCount int64
	if err := r.db.Model(&model.User{}).Where("role = ?", "super_admin").Count(&superAdminCount).Error; err != nil {
		return nil, err
	}
	stats["total_super_admins"] = superAdminCount

	return stats, nil
}

// CheckUserCanManage kiểm tra người dùng có thể quản lý user mục tiêu không
func (r *UserRepository) CheckUserCanManage(managerID, targetID uuid.UUID) (bool, error) {
	var manager, target model.User
	
	if err := r.db.First(&manager, "id = ?", managerID).Error; err != nil {
		return false, err
	}
	
	if err := r.db.First(&target, "id = ?", targetID).Error; err != nil {
		return false, err
	}

	// Super admin có thể quản lý tất cả trừ super admin khác
	// Admin không thể quản lý ai cả (chỉ tạo bài viết)
	return manager.Role == "super_admin" && target.Role != "super_admin", nil
}
