package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReviewHandler struct {
	reviewRepo *repo.ReviewRepo
}

func NewReviewHandler() *ReviewHandler {
	return &ReviewHandler{
		reviewRepo: repo.NewReviewRepo(),
	}
}

// CreateReview tạo đánh giá mới
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	var input struct {
		ProductID uuid.UUID `json:"product_id" binding:"required"`
		Rating    int       `json:"rating" binding:"required,min=1,max=5"`
		Title     string    `json:"title" binding:"max=200"`
		Comment   string    `json:"comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	// Kiểm tra người dùng đã đánh giá sản phẩm này chưa
	exists, err := h.reviewRepo.CheckUserReviewExists(userID.(uuid.UUID), input.ProductID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi hệ thống", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Bạn đã đánh giá sản phẩm này rồi", nil)
		return
	}

	review := &model.Review{
		UserID:    userID.(uuid.UUID),
		ProductID: input.ProductID,
		Rating:    input.Rating,
		Title:     input.Title,
		Comment:   input.Comment,
		IsActive:  true,
	}

	if err := h.reviewRepo.Create(review); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo đánh giá", err)
		return
	}

	helpers.SuccessResponse(c, "Đánh giá được tạo thành công", review)
}

// GetReviewsByProduct lấy đánh giá theo sản phẩm
func (h *ReviewHandler) GetReviewsByProduct(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID sản phẩm không hợp lệ", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	reviews, total, err := h.reviewRepo.GetByProductID(productID, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách đánh giá", err)
		return
	}

	result := map[string]interface{}{
		"reviews": reviews,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	helpers.SuccessResponse(c, "Lấy danh sách đánh giá thành công", result)
}

// GetReviewsByUser lấy đánh giá theo người dùng
func (h *ReviewHandler) GetReviewsByUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	reviews, total, err := h.reviewRepo.GetByUserID(userID.(uuid.UUID), page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách đánh giá", err)
		return
	}

	result := map[string]interface{}{
		"reviews": reviews,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	helpers.SuccessResponse(c, "Lấy danh sách đánh giá thành công", result)
}

// GetLatestReviews lấy đánh giá mới nhất
func (h *ReviewHandler) GetLatestReviews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 50 {
		limit = 50
	}

	reviews, err := h.reviewRepo.GetLatestReviews(limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách đánh giá", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy danh sách đánh giá mới nhất thành công", reviews)
}

// GetProductRatingStats lấy thống kê rating của sản phẩm
func (h *ReviewHandler) GetProductRatingStats(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID sản phẩm không hợp lệ", err)
		return
	}

	stats, err := h.reviewRepo.GetProductRatingStats(productID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy thống kê đánh giá", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thống kê đánh giá thành công", stats)
}

// UpdateReview cập nhật đánh giá
func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đánh giá không hợp lệ", err)
		return
	}

	var input struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Title   string `json:"title" binding:"max=200"`
		Comment string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu không hợp lệ", err)
		return
	}

	// Lấy đánh giá hiện tại
	review, err := h.reviewRepo.GetByID(reviewID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy đánh giá", err)
		return
	}

	// Kiểm tra quyền sở hữu
	if review.UserID != userID.(uuid.UUID) {
		helpers.ErrorResponse(c, http.StatusForbidden, "Bạn không có quyền sửa đánh giá này", nil)
		return
	}

	// Cập nhật thông tin
	review.Rating = input.Rating
	review.Title = input.Title
	review.Comment = input.Comment

	if err := h.reviewRepo.Update(review); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật đánh giá", err)
		return
	}

	helpers.SuccessResponse(c, "Cập nhật đánh giá thành công", review)
}

// DeleteReview xóa đánh giá
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Chưa xác thực")
		return
	}

	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đánh giá không hợp lệ", err)
		return
	}

	// Lấy đánh giá hiện tại
	review, err := h.reviewRepo.GetByID(reviewID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy đánh giá", err)
		return
	}

	// Kiểm tra quyền sở hữu
	if review.UserID != userID.(uuid.UUID) {
		helpers.ErrorResponse(c, http.StatusForbidden, "Bạn không có quyền xóa đánh giá này", nil)
		return
	}

	if err := h.reviewRepo.Delete(reviewID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa đánh giá", err)
		return
	}

	helpers.SuccessResponse(c, "Xóa đánh giá thành công", nil)
}

// AdminToggleReviewStatus admin đảo trạng thái đánh giá
func (h *ReviewHandler) AdminToggleReviewStatus(c *gin.Context) {
	reviewIDStr := c.Param("id")
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đánh giá không hợp lệ", err)
		return
	}

	if err := h.reviewRepo.ToggleStatus(reviewID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể thay đổi trạng thái đánh giá", err)
		return
	}

	helpers.SuccessResponse(c, "Thay đổi trạng thái đánh giá thành công", nil)
}