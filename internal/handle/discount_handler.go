package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DiscountHandler struct {
	discountRepo *repo.DiscountRepo
}

func NewDiscountHandler() *DiscountHandler {
	return &DiscountHandler{
		discountRepo: repo.NewDiscountRepo(nil),
	}
}

// CreateDiscount tạo mã giảm giá mới
func (h *DiscountHandler) CreateDiscount(c *gin.Context) {
	var input struct {
		Code             string    `json:"code" binding:"required,min=3,max=50"`
		Name             string    `json:"name" binding:"required,min=1,max=200"`
		Description      string    `json:"description"`
		Type             string    `json:"type" binding:"required,oneof=percentage fixed"`
		Value            float64   `json:"value" binding:"required,gt=0"`
		MinOrderAmount   float64   `json:"min_order_amount"`
		MaxDiscountValue float64   `json:"max_discount_value"`
		UsageLimit       int       `json:"usage_limit"`
		IsActive         bool      `json:"is_active"`
		StartDate        time.Time `json:"start_date" binding:"required"`
		EndDate          time.Time `json:"end_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Validate dates
	if !input.EndDate.After(input.StartDate) {
		helpers.ErrorResponse(c, http.StatusBadRequest, "End date must be after start date", nil)
		return
	}

	// Check if code already exists
	if exists, err := h.discountRepo.CheckCodeExists(input.Code, uuid.Nil); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	} else if exists {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount code already exists", nil)
		return
	}

	discount := &model.Discount{
		Code:             input.Code,
		Name:             input.Name,
		Description:      input.Description,
		Type:             input.Type,
		Value:            input.Value,
		MinOrderAmount:   input.MinOrderAmount,
		MaxDiscountValue: input.MaxDiscountValue,
		UsageLimit:       input.UsageLimit,
		IsActive:         input.IsActive,
		StartDate:        input.StartDate,
		EndDate:          input.EndDate,
	}

	if err := h.discountRepo.Create(discount); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to create discount", err)
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Discount created successfully",
		"data":    discount,
	})
}

// GetDiscounts lấy danh sách mã giảm giá
func (h *DiscountHandler) GetDiscounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	activeOnly := c.Query("active_only") == "true"

	discounts, total, err := h.discountRepo.GetAll(page, limit, activeOnly)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get discounts", err)
		return
	}

	response := map[string]interface{}{
		"discounts": discounts,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	helpers.SuccessResponse(c, "Discounts retrieved successfully", response)
}

// GetDiscountByCode lấy mã giảm giá theo code
func (h *DiscountHandler) GetDiscountByCode(c *gin.Context) {
	code := c.Param("code")

	discount, err := h.discountRepo.GetByCode(code)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	helpers.SuccessResponse(c, "Discount retrieved successfully", discount)
}

// UpdateDiscount cập nhật mã giảm giá
func (h *DiscountHandler) UpdateDiscount(c *gin.Context) {
	id := c.Param("id")
	discountID, err := uuid.Parse(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid discount ID", err)
		return
	}

	var input struct {
		Code             string    `json:"code"`
		Name             string    `json:"name"`
		Description      string    `json:"description"`
		Type             string    `json:"type" binding:"omitempty,oneof=percentage fixed"`
		Value            float64   `json:"value"`
		MinOrderAmount   float64   `json:"min_order_amount"`
		MaxDiscountValue float64   `json:"max_discount_value"`
		UsageLimit       int       `json:"usage_limit"`
		IsActive         *bool     `json:"is_active"`
		StartDate        time.Time `json:"start_date"`
		EndDate          time.Time `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Get existing discount
	discount, err := h.discountRepo.GetByID(discountID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	// Update fields if provided
	if input.Code != "" {
		if exists, err := h.discountRepo.CheckCodeExists(input.Code, discountID); err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
			return
		} else if exists {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Discount code already exists", nil)
			return
		}
		discount.Code = input.Code
	}
	
	if input.Name != "" {
		discount.Name = input.Name
	}
	
	if input.Description != "" {
		discount.Description = input.Description
	}
	
	if input.Type != "" {
		discount.Type = input.Type
	}
	
	if input.Value > 0 {
		discount.Value = input.Value
	}
	
	if input.MinOrderAmount >= 0 {
		discount.MinOrderAmount = input.MinOrderAmount
	}
	
	if input.MaxDiscountValue >= 0 {
		discount.MaxDiscountValue = input.MaxDiscountValue
	}
	
	if input.UsageLimit >= 0 {
		discount.UsageLimit = input.UsageLimit
	}
	
	if input.IsActive != nil {
		discount.IsActive = *input.IsActive
	}
	
	if !input.StartDate.IsZero() {
		discount.StartDate = input.StartDate
	}
	
	if !input.EndDate.IsZero() {
		discount.EndDate = input.EndDate
	}

	// Validate dates
	if !discount.EndDate.After(discount.StartDate) {
		helpers.ErrorResponse(c, http.StatusBadRequest, "End date must be after start date", nil)
		return
	}

	if err := h.discountRepo.Update(discount); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to update discount", err)
		return
	}

	helpers.SuccessResponse(c, "Discount updated successfully", discount)
}

// DeleteDiscount xóa mã giảm giá
func (h *DiscountHandler) DeleteDiscount(c *gin.Context) {
	id := c.Param("id")
	discountID, err := uuid.Parse(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid discount ID", err)
		return
	}

	if err := h.discountRepo.Delete(discountID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete discount", err)
		return
	}

	helpers.SuccessResponse(c, "Discount deleted successfully", nil)
}

// ValidateDiscount kiểm tra mã giảm giá có hợp lệ không
func (h *DiscountHandler) ValidateDiscount(c *gin.Context) {
	var input struct {
		Code        string  `json:"code" binding:"required"`
		OrderAmount float64 `json:"order_amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	discount, err := h.discountRepo.GetByCode(input.Code)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount code not found", err)
		return
	}

	if !discount.CanApply(input.OrderAmount) {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount code cannot be applied", nil)
		return
	}

	discountAmount := discount.CalculateDiscount(input.OrderAmount)
	finalAmount := input.OrderAmount - discountAmount

	response := map[string]interface{}{
		"valid":           true,
		"discount":        discount,
		"discount_amount": discountAmount,
		"final_amount":    finalAmount,
	}

	helpers.SuccessResponse(c, "Discount code is valid", response)
}

