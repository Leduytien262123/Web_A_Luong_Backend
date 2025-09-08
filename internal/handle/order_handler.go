package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderRepo   *repo.OrderRepo
	productRepo *repo.ProductRepo
	cartRepo    *repo.CartRepo
}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{
		orderRepo:   repo.NewOrderRepo(),
		productRepo: repo.NewProductRepo(),
		cartRepo:    repo.NewCartRepo(),
	}
}

// CreateOrder tạo đơn hàng mới
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var input model.OrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Cố gắng lấy user_id từ context (sẽ là nil đối với đơn hàng khách)
	userID, exists := c.Get("user_id")
	if exists && userID != nil {
		// Nếu người dùng đã đăng nhập, sử dụng ID của họ
		userIDValue := userID.(uuid.UUID)
		input.UserID = &userIDValue
	}
	// Nếu người dùng chưa đăng nhập, input.UserID sẽ là nil (đơn hàng khách)

	// Tạo số đơn hàng duy nhất
	orderNumber := h.generateUniqueOrderNumber()

	// Tính tổng tiền
	var totalAmount float64 = 0
	var orderItems []model.OrderItem

	for _, item := range input.Items {
		// Lấy thông tin sản phẩm
		product, err := h.productRepo.GetByID(item.ProductID)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Không tìm thấy sản phẩm", err)
			return
		}

		// Kiểm tra tồn kho
		if product.Stock < item.Quantity {
			helpers.ErrorResponse(c, http.StatusBadRequest, 
				fmt.Sprintf("Không đủ hàng tồn kho cho sản phẩm %s", product.Name), nil)
			return
		}

		itemTotal := float64(item.Quantity) * product.Price
		totalAmount += itemTotal

		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     product.Price,
			Total:     itemTotal,
		})
	}

	// Áp dụng phí vận chuyển
	shippingAmount := 30000.0 // Phí vận chuyển cố định
	if totalAmount >= 500000 { // Miễn phí vận chuyển cho đơn hàng >= 500k
		shippingAmount = 0
	}

	finalAmount := totalAmount + shippingAmount

	// Xác định xem đây có phải là đơn hàng khách không
	isGuestOrder := input.UserID == nil

	// Tạo đơn hàng
	order := model.Order{
		UserID:          input.UserID,
		OrderNumber:     orderNumber,
		Status:          "pending",
		PaymentStatus:   "pending",
		PaymentMethod:   input.PaymentMethod,
		TotalAmount:     totalAmount,
		ShippingAmount:  shippingAmount,
		FinalAmount:     finalAmount,
		CouponCode:      input.CouponCode,
		ShippingAddress: input.ShippingAddress,
		BillingAddress:  input.BillingAddress,
		CustomerName:    input.CustomerName,
		CustomerPhone:   input.CustomerPhone,
		CustomerEmail:   input.CustomerEmail,
		Notes:           input.Notes,
		IsGuestOrder:    isGuestOrder,
		OrderItems:      orderItems,
	}

	if err := h.orderRepo.Create(&order); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo đơn hàng", err)
		return
	}

	// Cập nhật tồn kho sản phẩm
	for _, item := range input.Items {
		product, _ := h.productRepo.GetByID(item.ProductID)
		newStock := product.Stock - item.Quantity
		h.productRepo.UpdateStock(item.ProductID, newStock)
	}

	// Xóa giỏ hàng người dùng chỉ khi đó là người dùng đã đăng nhập
	if input.UserID != nil {
		h.cartRepo.ClearCart(*input.UserID)
	}

	// Tải đơn hàng đã tạo với thông tin chi tiết
	createdOrder, err := h.orderRepo.GetByID(order.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tải đơn hàng đã tạo", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "Tạo đơn hàng thành công",
		Data:    createdOrder.ToResponse(),
	})
}

// GetOrders lấy danh sách đơn hàng với phân trang
func (h *OrderHandler) GetOrders(c *gin.Context) {
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

	orders, total, err := h.orderRepo.GetAll(page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách đơn hàng", err)
		return
	}

	var response []model.OrderResponse
	for _, order := range orders {
		response = append(response, order.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách đơn hàng thành công",
		Data: map[string]interface{}{
			"orders":      response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"has_next":    page < int(totalPages),
			"has_prev":    page > 1,
		},
	})
}

// GetMyOrders lấy đơn hàng của người dùng
func (h *OrderHandler) GetMyOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Không có quyền truy cập")
		return
	}

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

	orders, total, err := h.orderRepo.GetByUserID(userID.(uuid.UUID), page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách đơn hàng", err)
		return
	}

	var response []model.OrderResponse
	for _, order := range orders {
		response = append(response, order.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách đơn hàng thành công",
		Data: map[string]interface{}{
			"orders":      response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"has_next":    page < int(totalPages),
			"has_prev":    page > 1,
		},
	})
}

// GetOrderByID lấy đơn hàng theo ID
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đơn hàng không hợp lệ", err)
		return
	}

	order, err := h.orderRepo.GetByID(id)
	if err != nil {
		if err.Error() == "order not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy đơn hàng", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy đơn hàng thành công",
		Data:    order.ToResponse(),
	})
}

// UpdateOrderStatus cập nhật trạng thái đơn hàng
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đơn hàng không hợp lệ", err)
		return
	}

	var input struct {
		Status string `json:"status" binding:"required,oneof=pending confirmed processing shipped delivered cancelled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	if err := h.orderRepo.UpdateStatus(id, input.Status); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật trạng thái đơn hàng", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật trạng thái đơn hàng thành công",
		Data:    map[string]interface{}{"status": input.Status},
	})
}

// UpdatePaymentStatus cập nhật trạng thái thanh toán
func (h *OrderHandler) UpdatePaymentStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đơn hàng không hợp lệ", err)
		return
	}

	var input struct {
		PaymentStatus string `json:"payment_status" binding:"required,oneof=pending paid failed refunded"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	if err := h.orderRepo.UpdatePaymentStatus(id, input.PaymentStatus); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật trạng thái thanh toán", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật trạng thái thanh toán thành công",
		Data:    map[string]interface{}{"payment_status": input.PaymentStatus},
	})
}

// GetOrderStats lấy thống kê đơn hàng
func (h *OrderHandler) GetOrderStats(c *gin.Context) {
	stats, err := h.orderRepo.GetOrderStats()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy thống kê đơn hàng", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thống kê đơn hàng thành công",
		Data:    stats,
	})
}

func (h *OrderHandler) generateUniqueOrderNumber() string {
	for {
		timestamp := time.Now().Format("20060102150405")
		randomNum := rand.Intn(9999)
		orderNumber := fmt.Sprintf("ORD-%s-%04d", timestamp, randomNum)
		
		exists, _ := h.orderRepo.CheckOrderNumberExists(orderNumber)
		if !exists {
			return orderNumber
		}
	}
}

// LookupGuestOrders lấy đơn hàng theo email hoặc số điện thoại (cho khách hàng)
func (h *OrderHandler) LookupGuestOrders(c *gin.Context) {
	var input model.GuestOrderLookupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

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

	orders, total, err := h.orderRepo.GetByEmailOrPhone(input.EmailOrPhone, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách đơn hàng", err)
		return
	}

	var response []model.OrderResponse
	for _, order := range orders {
		response = append(response, order.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách đơn hàng thành công",
		Data: map[string]interface{}{
			"orders":      response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"has_next":    page < int(totalPages),
			"has_prev":    page > 1,
		},
	})
}

// TrackOrderByNumber lấy đơn hàng theo số đơn hàng (endpoint công khai)
func (h *OrderHandler) TrackOrderByNumber(c *gin.Context) {
	orderNumber := c.Param("order_number")
	if orderNumber == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Số đơn hàng là bắt buộc", nil)
		return
	}

	order, err := h.orderRepo.GetByOrderNumber(orderNumber)
	if err != nil {
		if err.Error() == "order not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy đơn hàng", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy đơn hàng thành công",
		Data:    order.ToResponse(),
	})
}

// GetGuestOrderStats lấy thống kê đơn hàng khách (chỉ admin)
func (h *OrderHandler) GetGuestOrderStats(c *gin.Context) {
	stats, err := h.orderRepo.GetGuestOrderStats()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy thống kê đơn hàng khách", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thống kê đơn hàng khách thành công",
		Data:    stats,
	})
}