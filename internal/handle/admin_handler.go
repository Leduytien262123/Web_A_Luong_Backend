package handle

import (
	"backend/internal/consts"
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AdminHandler struct {
	userRepo *repo.UserRepository
}

func NewAdminHandler(userRepo *repo.UserRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
	}
}

func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	roleQuery := c.Query("role")
	name := c.Query("name")
	phone := c.Query("phone")
	email := c.Query("email")

	var roles []string
	if roleQuery != "" {
		roles = strings.Split(roleQuery, ",")
	}
	users, total, err := h.userRepo.GetUsersByRolesWithPagination(roles, name, phone, email, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	var response []model.UserResponse
	for _, user := range users {
		response = append(response, user.ToResponse())
	}

	result := map[string]interface{}{
		"users": response,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, result)
}

func (h *AdminHandler) GetUserByID(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		helpers.BadRequestResponse(c, "ID người dùng không hợp lệ")
		return
	}

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.ErrorResponse(c, http.StatusNotFound, consts.MSG_USER_NOT_FOUND, nil)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, user.ToResponse())
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		helpers.BadRequestResponse(c, "ID người dùng không hợp lệ")
		return
	}

	var input struct {
		Role string `json:"role" binding:"required,oneof=super_admin admin"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.BadRequestResponse(c, consts.MSG_VALIDATION_ERROR)
		return
	}

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.ErrorResponse(c, http.StatusNotFound, consts.MSG_USER_NOT_FOUND, nil)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	user.Role = input.Role
	if err := h.userRepo.UpdateUser(user); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, user.ToResponse())
}

func (h *AdminHandler) ToggleUserStatus(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		helpers.BadRequestResponse(c, "ID người dùng không hợp lệ")
		return
	}

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.ErrorResponse(c, http.StatusNotFound, consts.MSG_USER_NOT_FOUND, nil)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	user.IsActive = !user.IsActive
	if err := h.userRepo.UpdateUser(user); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, user.ToResponse())
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		helpers.BadRequestResponse(c, "ID người dùng không hợp lệ")
		return
	}

	_, err = h.userRepo.GetUserByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.ErrorResponse(c, http.StatusNotFound, consts.MSG_USER_NOT_FOUND, nil)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	if err := h.userRepo.DeleteUser(userID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	helpers.SuccessResponse(c, "Xóa người dùng thành công", nil)
}

// CreateUser tạo tài khoản admin mới (chỉ Super Admin)
func (h *AdminHandler) CreateUser(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ super_admin mới có thể tạo người dùng
	if currentUserRole != "super_admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không đủ quyền", nil)
		return
	}

	var input model.CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Không cho phép tạo super_admin khác
	if input.Role == "super_admin" {
		exists, err := h.userRepo.CheckSuperAdminExists()
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		}
		if exists {
			helpers.ErrorResponse(c, http.StatusConflict, "Chỉ được phép có một tài khoản Super Admin", nil)
			return
		}
	}

	// Kiểm tra username đã tồn tại
	if h.userRepo.IsUsernameExists(input.Username) {
		helpers.ErrorResponse(c, http.StatusConflict, "Tên người dùng đã tồn tại", nil)
		return
	}

	// Kiểm tra email đã tồn tại
	if h.userRepo.IsEmailExists(input.Email) {
		helpers.ErrorResponse(c, http.StatusConflict, "Email đã tồn tại", nil)
		return
	}

	// Mã hóa mật khẩu
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể mã hóa mật khẩu", err)
		return
	}

	// Tạo người dùng
	var avatarJSON datatypes.JSON
	if len(input.Avatar) > 0 {
		if b, err := json.Marshal(input.Avatar); err == nil {
			avatarJSON = datatypes.JSON(b)
		} else {
			avatarJSON = datatypes.JSON([]byte(`[]`))
		}
	} else {
		avatarJSON = datatypes.JSON([]byte(`[]`))
	}

	user := model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
		FullName: input.FullName,
		Avatar:   avatarJSON,
		Role:     input.Role,
		IsActive: true,
	}

	if err := h.userRepo.CreateUser(&user); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo người dùng", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "Tạo người dùng thành công",
		Data:    user.ToResponse(),
	})
}

// GetUsersByRole lấy danh sách người dùng theo vai trò
func (h *AdminHandler) GetUsersByRole(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ super_admin mới có quyền
	if currentUserRole != "super_admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không đủ quyền", nil)
		return
	}

	role := c.Param("role")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	users, total, err := h.userRepo.GetUsersByRole(role, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách người dùng", err)
		return
	}

	var response []model.UserResponse
	for _, user := range users {
		response = append(response, user.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách người dùng thành công",
		Data: map[string]interface{}{
			"users":       response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"has_next":    page < int(totalPages),
			"has_prev":    page > 1,
		},
	})
}

// AssignUserRole gán vai trò người dùng (chỉ Super Admin)
func (h *AdminHandler) AssignUserRole(c *gin.Context) {
	currentUserID, exists := c.Get("userID")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ super_admin mới có quyền
	if currentUserRole != "super_admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không đủ quyền", nil)
		return
	}

	targetUserIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID người dùng không hợp lệ", err)
		return
	}

	var input model.UpdateUserRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra có thể quản lý user đích không
	canManage, err := h.userRepo.CheckUserCanManage(currentUserID.(uuid.UUID), targetUserID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	if !canManage {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không thể quản lý người dùng này", nil)
		return
	}

	// Không cho phép tạo thêm super_admin
	if input.Role == "super_admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không thể gán vai trò Super Admin", nil)
		return
	}

	// Cập nhật vai trò
	if err := h.userRepo.UpdateUserRole(targetUserID, input.Role); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật vai trò người dùng", err)
		return
	}

	// Lấy thông tin đã cập nhật
	updatedUser, err := h.userRepo.GetUserByID(targetUserID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy thông tin người dùng đã cập nhật", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật vai trò người dùng thành công",
		Data:    updatedUser.ToResponse(),
	})
}

// GetUserStats lấy thống kê người dùng (chỉ Super Admin)
func (h *AdminHandler) GetUserStats(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ super_admin mới có quyền
	if currentUserRole != "super_admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không đủ quyền", nil)
		return
	}

	stats, err := h.userRepo.GetUserStats()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy thống kê người dùng", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thống kê người dùng thành công",
		Data:    stats,
	})
}

// UpdateUser cập nhật thông tin user (chỉ Super Admin)
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID người dùng không hợp lệ", err)
		return
	}

	var input model.AdminUserUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.BadRequestResponse(c, consts.MSG_VALIDATION_ERROR)
		return
	}

	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.ErrorResponse(c, http.StatusNotFound, consts.MSG_USER_NOT_FOUND, nil)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	// Chỉ super_admin mới được sửa thông tin
	currentUserRole, exists := c.Get("user_role")
	if !exists || currentUserRole != "super_admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Chỉ Super Admin mới được sửa thông tin người dùng", nil)
		return
	}

	// Cập nhật các trường
	if input.FullName != "" {
		user.FullName = input.FullName
	}
	if input.Email != "" && input.Email != user.Email {
		if h.userRepo.IsEmailExists(input.Email) {
			helpers.ErrorResponse(c, http.StatusBadRequest, consts.MSG_EMAIL_EXISTS, nil)
			return
		}
		user.Email = input.Email
	}
	if len(input.Avatar) > 0 {
		if b, err := json.Marshal(input.Avatar); err == nil {
			user.Avatar = datatypes.JSON(b)
		} else {
			user.Avatar = datatypes.JSON([]byte(`[]`))
		}
	}
	if input.Phone != "" {
		user.Phone = input.Phone
	}
	
	// Không cho phép thay đổi role của super_admin
	if user.Role != "super_admin" && input.Role != "" {
		user.Role = input.Role
	}

	if err := h.userRepo.UpdateUser(user); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	helpers.SuccessResponse(c, "Cập nhật thông tin người dùng thành công", user.ToResponse())
}
