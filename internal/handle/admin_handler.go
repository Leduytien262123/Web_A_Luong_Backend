package handle

import (
	"backend/internal/consts"
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct {
	userRepo *repo.UserRepository
}

func NewAdminHandler(userRepo *repo.UserRepository) *AdminHandler {
	return &AdminHandler{userRepo: userRepo}
}

func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	var response []model.UserResponse
	for _, user := range users {
		response = append(response, user.ToResponse())
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, response)
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
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.BadRequestResponse(c, consts.MSG_VALIDATION_ERROR)
		return
	}

	// Xác thực vai trò
	if input.Role != consts.ROLE_ADMIN && input.Role != consts.ROLE_USER {
		helpers.BadRequestResponse(c, "Vai trò không hợp lệ")
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

	// Kiểm tra xem người dùng có tồn tại không
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

// CreateUser tạo tài khoản người dùng mới (chỉ Owner và Admin)
func (h *AdminHandler) CreateUser(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ owner và admin mới có thể tạo người dùng
	if currentUserRole != "owner" && currentUserRole != "admin" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không đủ quyền", nil)
		return
	}

	var input model.CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Admin không thể tạo tài khoản owner
	if currentUserRole == "admin" && input.Role == "owner" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Admin không thể tạo tài khoản owner", nil)
		return
	}

	// Kiểm tra xem owner đã tồn tại chưa khi cố gắng tạo owner
	if input.Role == "owner" {
		exists, err := h.userRepo.CheckOwnerExists()
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		}
		if exists {
			helpers.ErrorResponse(c, http.StatusConflict, "Tài khoản owner đã tồn tại", nil)
			return
		}
	}

	// Kiểm tra xem username đã tồn tại chưa
	if h.userRepo.IsUsernameExists(input.Username) {
		helpers.ErrorResponse(c, http.StatusConflict, "Tên người dùng đã tồn tại", nil)
		return
	}

	// Kiểm tra xem email đã tồn tại chưa
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
	user := model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
		FullName: input.FullName,
		Avatar:   input.Avatar,
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

// GetUsersByRole lấy danh sách người dùng được lọc theo vai trò
func (h *AdminHandler) GetUsersByRole(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ owner và admin mới có thể xem người dùng theo vai trò
	if currentUserRole != "owner" && currentUserRole != "admin" {
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

// AssignUserRole gán hoặc cập nhật vai trò người dùng (chỉ Owner và Admin)
func (h *AdminHandler) AssignUserRole(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ owner và admin mới có thể gán vai trò
	if currentUserRole != "owner" && currentUserRole != "admin" {
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

	// Kiểm tra xem người dùng có thể quản lý người dùng đích không
	canManage, err := h.userRepo.CheckUserCanManage(currentUserID.(uuid.UUID), targetUserID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	if !canManage {
		helpers.ErrorResponse(c, http.StatusForbidden, "Không thể quản lý người dùng này", nil)
		return
	}

	// Admin không thể gán vai trò owner
	if currentUserRole == "admin" && input.Role == "owner" {
		helpers.ErrorResponse(c, http.StatusForbidden, "Admin không thể gán vai trò owner", nil)
		return
	}

	// Kiểm tra xem có cố gắng tạo owner thứ hai không
	if input.Role == "owner" {
		exists, err := h.userRepo.CheckOwnerExists()
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		}
		if exists {
			helpers.ErrorResponse(c, http.StatusConflict, "Tài khoản owner đã tồn tại", nil)
			return
		}
	}

	// Cập nhật vai trò người dùng
	if err := h.userRepo.UpdateUserRole(targetUserID, input.Role); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật vai trò người dùng", err)
		return
	}

	// Lấy thông tin người dùng đã cập nhật
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

// GetUserStats lấy thống kê người dùng (chỉ Owner và Admin)
func (h *AdminHandler) GetUserStats(c *gin.Context) {
	currentUserRole, exists := c.Get("user_role")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	// Chỉ owner và admin mới có thể xem thống kê
	if currentUserRole != "owner" && currentUserRole != "admin" {
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

// UpdateUser cập nhật thông tin user theo ID (dành cho admin)
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID người dùng không hợp lệ", err)
		return
	}
	
	var input model.UpdateUserInput
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

	// Cập nhật các trường
	if input.FullName != "" {
		user.FullName = input.FullName
	}
	if input.Email != "" && input.Email != user.Email {
		// Kiểm tra xem email đã tồn tại chưa
		if h.userRepo.IsEmailExists(input.Email) {
			helpers.ErrorResponse(c, http.StatusBadRequest, consts.MSG_EMAIL_EXISTS, nil)
			return
		}
		user.Email = input.Email
	}
	if input.Avatar != "" {
		user.Avatar = input.Avatar
	}

	if err := h.userRepo.UpdateUser(user); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	helpers.SuccessResponse(c, "Cập nhật thông tin người dùng thành công", user.ToResponse())
}
