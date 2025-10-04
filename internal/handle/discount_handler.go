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

// calculateDiscountStatus tính toán trạng thái của mã giảm giá
func (h *DiscountHandler) calculateDiscountStatus(discount *model.Discount) string {
	now := time.Now()
	
	// Nếu không active (bị pause) thì là stopped
	if (!discount.IsActive) {
		return "stopped"
	}
	
	// Nếu chưa đến ngày bắt đầu
	if (now.Before(discount.StartDate)) {
		return "upcoming"
	}
	
	// Nếu đã qua ngày kết thúc
	if (now.After(discount.EndDate)) {
		return "ended"
	}
	
	// Đang trong thời gian hiệu lực
	return "active"
}

func discountToMapWithIds(discount *model.Discount, status string) map[string]interface{} {
	m := helpers.StructToMap(*discount)
	// applied_products chỉ trả về mảng id
	productIDs := make([]string, 0)
	for _, ap := range discount.AppliedProducts {
		productIDs = append(productIDs, ap.ProductID.String())
	}
	m["applied_products"] = productIDs
	// applied_category_products chỉ trả về mảng id
	categoryIDs := make([]string, 0)
	for _, ac := range discount.AppliedCategoryProducts {
		categoryIDs = append(categoryIDs, ac.CategoryID.String())
	}
	m["applied_category_products"] = categoryIDs
	m["status"] = status
	return m
}

// CreateDiscount tạo mã giảm giá mới theo format yêu cầu
func (h *DiscountHandler) CreateDiscount(c *gin.Context) {
	var input struct {
		Name                    string      `json:"name" binding:"required"`
		DiscountCode            string      `json:"discount_code" binding:"required,min=3,max=50"`
		StartAt                 time.Time   `json:"start_at" binding:"required"`
		EndAt                   time.Time   `json:"end_at" binding:"required"`
		Type                    string      `json:"type" binding:"required,oneof=percentage fixed"`
		ValueVoucher            float64     `json:"value_voucher" binding:"required,gt=0"`
		Condition               float64     `json:"condition" binding:"gte=0"`
		Quantity                int         `json:"quantity" binding:"required,gte=0"`
		UsageLimit              int         `json:"usage_limit" binding:"oneof=0 1"`
		UsageCount              *int        `json:"usage_count"`
		AppliedProducts         []string    `json:"applied_products"`
		AppliedCategoryProducts []string    `json:"applied_category_products"`
		Description             string      `json:"description"`
		MaximumDiscount         *float64    `json:"maximum_discount"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Validate ngày bắt đầu không được trước ngày hiện tại
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if input.StartAt.Before(startOfToday) {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Start date cannot be before today", nil)
		return
	}

	// Validate dates
	if !input.EndAt.After(input.StartAt) {
		helpers.ErrorResponse(c, http.StatusBadRequest, "End date must be after start date", nil)
		return
	}

	// Check if code already exists
	if exists, err := h.discountRepo.CheckCodeExists(input.DiscountCode, uuid.Nil); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	} else if exists {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount code already exists", nil)
		return
	}

	// Validate usage_count when usage_limit = 1
	usageCount := 0
	if input.UsageLimit == 1 {
		if input.UsageCount == nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "usage_count is required when usage_limit = 1", nil)
			return
		}
		usageCount = *input.UsageCount
	}

	// Validate và set maximum_discount for percentage type
	maximumDiscount := float64(0)
	if input.Type == "percentage" {
		if input.MaximumDiscount != nil && *input.MaximumDiscount > 0 {
			maximumDiscount = *input.MaximumDiscount
		}
	}

	discount := &model.Discount{
		DiscountCode:    input.DiscountCode,
		Name:            input.Name,
		Description:     input.Description,
		Type:            input.Type,
		ValueVoucher:    input.ValueVoucher,
		MinOrderAmount:  input.Condition, // Sử dụng condition để set min_order_amount
		Condition:       input.Condition,
		MaximumDiscount: maximumDiscount,
		Quantity:        input.Quantity,
		UsageLimit:      input.UsageLimit,
		UsageCount:      usageCount,
		IsActive:        true,
		StartDate:       input.StartAt,
		EndDate:         input.EndAt,
	}

	// Parse applied products
	var productIDs []uuid.UUID
	for _, productIDStr := range input.AppliedProducts {
		if productID, err := uuid.Parse(productIDStr); err == nil {
			productIDs = append(productIDs, productID)
		}
	}

	// Parse applied category products
	var categoryIDs []uuid.UUID
	for _, categoryIDStr := range input.AppliedCategoryProducts {
		if categoryID, err := uuid.Parse(categoryIDStr); err == nil {
			categoryIDs = append(categoryIDs, categoryID)
		}
	}

	// Tạo discount với associations
	if err := h.discountRepo.CreateWithAssociations(discount, productIDs, categoryIDs); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to create discount", err)
		return
	}

	// Reload discount với thông tin liên kết
	createdDiscount, err := h.discountRepo.GetByID(discount.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get created discount", err)
		return
	}

	// Thêm status vào response
	response := discountToMapWithIds(createdDiscount, h.calculateDiscountStatus(createdDiscount))

	c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Discount created successfully",
		"data":    response,
	})
}

// GetDiscounts lấy danh sách mã giảm giá với bộ lọc
func (h *DiscountHandler) GetDiscounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	length, _ := strconv.Atoi(c.DefaultQuery("length", "10")) // Sử dụng length thay vì limit
	search := c.DefaultQuery("search", "")
	discountType := c.DefaultQuery("type", "")
	status := c.DefaultQuery("status", "")
	conditionStr := c.Query("condition") // Lấy condition từ query params

	var conditionFilter *float64
	if conditionStr != "" {
		if condition, err := strconv.ParseFloat(conditionStr, 64); err == nil {
			conditionFilter = &condition
		}
	}

	discounts, total, err := h.discountRepo.GetAllWithCondition(page, length, search, discountType, status, conditionFilter)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get discounts", err)
		return
	}

	// Thêm status cho từng discount
	discountsWithStatus := make([]map[string]interface{}, len(discounts))
	for i, discount := range discounts {
		discountsWithStatus[i] = discountToMapWithIds(&discount, h.calculateDiscountStatus(&discount))
	}

	response := map[string]interface{}{
		"discounts": discountsWithStatus,
		"pagination": map[string]interface{}{
			"page":   page,
			"length": length,
			"total":  total,
		},
	}

	helpers.SuccessResponse(c, "Discounts retrieved successfully", response)
}

// GetDiscountByID lấy chi tiết mã giảm giá theo ID
func (h *DiscountHandler) GetDiscountByID(c *gin.Context) {
	id := c.Param("id")
	discountID, err := uuid.Parse(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid discount ID", err)
		return
	}

	discount, err := h.discountRepo.GetByID(discountID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	response := discountToMapWithIds(discount, h.calculateDiscountStatus(discount))

	helpers.SuccessResponse(c, "Discount retrieved successfully", response)
}

// GetDiscountByCode lấy mã giảm giá theo code
func (h *DiscountHandler) GetDiscountByCode(c *gin.Context) {
	code := c.Param("code")

	discount, err := h.discountRepo.GetByCode(code)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	response := discountToMapWithIds(discount, h.calculateDiscountStatus(discount))

	helpers.SuccessResponse(c, "Discount retrieved successfully", response)
}

// UpdateDiscount cập nhật mã giảm giá
func (h *DiscountHandler) UpdateDiscount(c *gin.Context) {
	discountID := c.Param("id")
	if discountID == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount ID is required", nil)
		return
	}

	// Parse UUID
	id, err := uuid.Parse(discountID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid discount ID format", err)
		return
	}

	var input struct {
		Name                    string      `json:"name" binding:"required"`
		DiscountCode            string      `json:"discount_code" binding:"required,min=3,max=50"`
		StartAt                 time.Time   `json:"start_at" binding:"required"`
		EndAt                   time.Time   `json:"end_at" binding:"required"`
		Type                    string      `json:"type" binding:"required,oneof=percentage fixed"`
		ValueVoucher            float64     `json:"value_voucher" binding:"required,gt=0"`
		Condition               float64     `json:"condition" binding:"gte=0"`
		Quantity                int         `json:"quantity" binding:"required,gte=0"`
		UsageLimit              int         `json:"usage_limit" binding:"oneof=0 1"`
		UsageCount              *int        `json:"usage_count"`
		AppliedProducts         []string    `json:"applied_products"`
		AppliedCategoryProducts []string    `json:"applied_category_products"`
		Description             string      `json:"description"`
		MaximumDiscount         *float64    `json:"maximum_discount"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Get existing discount
	discount, err := h.discountRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	// Validate dates
	if !input.EndAt.After(input.StartAt) {
		helpers.ErrorResponse(c, http.StatusBadRequest, "End date must be after start date", nil)
		return
	}

	// Check if code already exists (excluding current discount)
	if exists, err := h.discountRepo.CheckCodeExists(input.DiscountCode, id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	} else if exists {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount code already exists", nil)
		return
	}

	// Validate usage_count when usage_limit = 1
	usageCount := 0
	if input.UsageLimit == 1 {
		if input.UsageCount == nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "usage_count is required when usage_limit = 1", nil)
			return
		}
		usageCount = *input.UsageCount
	}

	// Validate và set maximum_discount for percentage type
	maximumDiscount := float64(0)
	if input.Type == "percentage" {
		if input.MaximumDiscount != nil && *input.MaximumDiscount > 0 {
			maximumDiscount = *input.MaximumDiscount
		}
	}

	// Update discount fields
	discount.Name = input.Name
	discount.DiscountCode = input.DiscountCode
	discount.Description = input.Description
	discount.Type = input.Type
	discount.ValueVoucher = input.ValueVoucher
	discount.MinOrderAmount = input.Condition // Sử dụng condition để set min_order_amount
	discount.Condition = input.Condition
	discount.MaximumDiscount = maximumDiscount
	discount.Quantity = input.Quantity
	discount.UsageLimit = input.UsageLimit
	discount.UsageCount = usageCount
	discount.StartDate = input.StartAt
	discount.EndDate = input.EndAt

	// Parse applied products
	var productIDs []uuid.UUID
	for _, productIDStr := range input.AppliedProducts {
		if productID, err := uuid.Parse(productIDStr); err == nil {
			productIDs = append(productIDs, productID)
		}
	}

	// Parse applied category products
	var categoryIDs []uuid.UUID
	for _, categoryIDStr := range input.AppliedCategoryProducts {
		if categoryID, err := uuid.Parse(categoryIDStr); err == nil {
			categoryIDs = append(categoryIDs, categoryID)
		}
	}

	// Update discount với associations
	if err := h.discountRepo.UpdateWithAssociations(discount, productIDs, categoryIDs); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to update discount", err)
		return
	}

	// Reload discount với thông tin liên kết
	updatedDiscount, err := h.discountRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get updated discount", err)
		return
	}

	// Thêm status vào response
	response := discountToMapWithIds(updatedDiscount, h.calculateDiscountStatus(updatedDiscount))

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Discount updated successfully",
		"data":    response,
	})
}

// PauseDiscount tạm dừng mã giảm giá
func (h *DiscountHandler) PauseDiscount(c *gin.Context) {
	id := c.Param("id")
	discountID, err := uuid.Parse(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid discount ID", err)
		return
	}

	discount, err := h.discountRepo.GetByID(discountID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	status := h.calculateDiscountStatus(discount)
	if status == "ended" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Cannot pause ended discount", nil)
		return
	}

	if status == "stopped" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount is already paused", nil)
		return
	}

	discount.IsActive = false
	if err := h.discountRepo.Update(discount); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to pause discount", err)
		return
	}

	response := discountToMapWithIds(discount, "stopped")

	helpers.SuccessResponse(c, "Discount paused successfully", response)
}

// ResumeDiscount tiếp tục mã giảm giá
func (h *DiscountHandler) ResumeDiscount(c *gin.Context) {
	id := c.Param("id")
	discountID, err := uuid.Parse(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid discount ID", err)
		return
	}

	discount, err := h.discountRepo.GetByID(discountID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	// Temporarily set active to check what status would be
	originalActive := discount.IsActive
	discount.IsActive = true
	status := h.calculateDiscountStatus(discount)
	discount.IsActive = originalActive

	if status == "ended" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Cannot resume ended discount", nil)
		return
	}

	if discount.IsActive {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount is already active", nil)
		return
	}

	discount.IsActive = true
	if err := h.discountRepo.Update(discount); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to resume discount", err)
		return
	}

	response := discountToMapWithIds(discount, h.calculateDiscountStatus(discount))

	helpers.SuccessResponse(c, "Discount resumed successfully", response)
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

// ValidateDiscount kiểm tra mã giảm giá có hợp lệ không với user cụ thể
func (h *DiscountHandler) ValidateDiscount(c *gin.Context) {
	var input struct {
		Code        string             `json:"code" binding:"required"`
		OrderAmount float64            `json:"order_amount" binding:"required,gt=0"`
		UserID      *uuid.UUID         `json:"user_id"`
		Products    []OrderProductInfo `json:"products" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	userID := uuid.Nil
	if input.UserID != nil {
		userID = *input.UserID
	}

	discount, err := h.discountRepo.GetValidDiscountForOrder(input.Code, input.OrderAmount, userID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// Kiểm tra xem có sản phẩm nào được áp dụng mã giảm giá không
	hasApplicableProduct := false
	for _, product := range input.Products {
		if applicable, err := h.discountRepo.CheckProductApplicable(discount.ID, product.ProductID); err == nil && applicable {
			hasApplicableProduct = true
			break
		}
	}

	if !hasApplicableProduct {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Discount code cannot be applied to any products in the order", nil)
		return
	}

	discountAmount := discount.CalculateDiscount(input.OrderAmount)
	finalAmount := input.OrderAmount - discountAmount

	response := map[string]interface{}{
		"valid":           true,
		"discount":        discount,
		"discount_amount": discountAmount,
		"final_amount":    finalAmount,
		"maximum_discount": discount.MaximumDiscount,
	}

	helpers.SuccessResponse(c, "Discount code is valid", response)
}

// ApplyDiscountToOrder áp dụng mã giảm giá cho đơn hàng và ghi lại usage
func (h *DiscountHandler) ApplyDiscountToOrder(c *gin.Context) {
	var input struct {
		DiscountCode string     `json:"discount_code" binding:"required"`
		OrderID      uuid.UUID  `json:"order_id" binding:"required"`
		UserID       uuid.UUID  `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	discount, err := h.discountRepo.GetByCode(input.DiscountCode)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Discount not found", err)
		return
	}

	// Ghi lại việc sử dụng
	if err := h.discountRepo.RecordUserUsage(input.UserID, discount.ID, input.OrderID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to record usage", err)
		return
	}

	// Tăng used_count
	if err := h.discountRepo.IncrementUsedCount(discount.ID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to increment used count", err)
		return
	}

	helpers.SuccessResponse(c, "Discount applied successfully", nil)
}

// GetDiscountsByProduct lấy mã giảm giá áp dụng cho sản phẩm
func (h *DiscountHandler) GetDiscountsByProduct(c *gin.Context) {
	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	discounts, err := h.discountRepo.GetDiscountsByProduct(productID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get discounts", err)
		return
	}

	helpers.SuccessResponse(c, "Discounts retrieved successfully", discounts)
}

// GetDiscountsByCategory lấy mã giảm giá áp dụng cho danh mục
func (h *DiscountHandler) GetDiscountsByCategory(c *gin.Context) {
	categoryIDStr := c.Param("category_id")
	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid category ID", err)
		return
	}

	discounts, err := h.discountRepo.GetDiscountsByCategory(categoryID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get discounts", err)
		return
	}

	helpers.SuccessResponse(c, "Discounts retrieved successfully", discounts)
}

type OrderProductInfo struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
}

