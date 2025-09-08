package handle

import (
	"backend/internal/consts"
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthHandler struct {
	userRepo *repo.UserRepository
}

func NewAuthHandler(userRepo *repo.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input model.UserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ValidationErrorResponse(c, consts.MSG_VALIDATION_ERROR)
		return
	}

	// Kiểm tra xem username đã tồn tại chưa
	if h.userRepo.IsUsernameExists(input.Username) {
		helpers.ErrorResponse(c, http.StatusBadRequest, consts.MSG_USERNAME_EXISTS, nil)
		return
	}

	// Kiểm tra xem email đã tồn tại chưa
	if h.userRepo.IsEmailExists(input.Email) {
		helpers.ErrorResponse(c, http.StatusBadRequest, consts.MSG_EMAIL_EXISTS, nil)
		return
	}

	// Mã hóa mật khẩu
	hashedPassword, err := helpers.HashPassword(input.Password)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	// Tạo người dùng
	user := model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: hashedPassword,
		FullName: input.FullName,
		Avatar:   input.Avatar,
		Role:     consts.ROLE_USER,
	}

	if err := h.userRepo.CreateUser(&user); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	// Tạo JWT token
	token, err := helpers.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	response := gin.H{
		"user":  user.ToResponse(),
		"token": token,
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, response)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input model.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ValidationErrorResponse(c, consts.MSG_VALIDATION_ERROR)
		return
	}

	// Tìm người dùng theo username
	user, err := h.userRepo.GetUserByUsername(input.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			helpers.ErrorResponse(c, http.StatusUnauthorized, consts.MSG_INVALID_CREDENTIALS, nil)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	// Kiểm tra mật khẩu
	if !helpers.CheckPasswordHash(input.Password, user.Password) {
		helpers.ErrorResponse(c, http.StatusUnauthorized, consts.MSG_INVALID_CREDENTIALS, nil)
		return
	}

	// Kiểm tra xem người dùng có đang hoạt động không
	if !user.IsActive {
		helpers.ErrorResponse(c, http.StatusUnauthorized, consts.MSG_UNAUTHORIZED, nil)
		return
	}

	// Tạo JWT token
	token, err := helpers.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, consts.MSG_INTERNAL_ERROR, err)
		return
	}

	response := gin.H{
		"user":  user.ToResponse(),
		"token": token,
	}

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, response)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
		return
	}

	user, err := h.userRepo.GetUserByID(userID.(uuid.UUID))
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

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, consts.MSG_UNAUTHORIZED)
		return
	}

	var input model.UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ValidationErrorResponse(c, consts.MSG_VALIDATION_ERROR)
		return
	}

	user, err := h.userRepo.GetUserByID(userID.(uuid.UUID))
	if err != nil {
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

	helpers.SuccessResponse(c, consts.MSG_SUCCESS, user.ToResponse())
}

