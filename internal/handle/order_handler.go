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
	var userIDPtr *uuid.UUID
	if exists && userID != nil {
		// Nếu người dùng đã đăng nhập, sử dụng ID của họ
		userIDValue := userID.(uuid.UUID)
		userIDPtr = &userIDValue
		input.UserID = userIDPtr
	}
	// Nếu người dùng chưa đăng nhập, input.UserID sẽ là nil (đơn hàng khách)

	// Tạo số đơn hàng duy nhất
	orderCode := h.generateUniqueOrderCode()

	// Tính tổng tiền
	var totalAmount float64 = 0
	var orderItems []model.OrderItem

	for _, item := range input.Items {
		// Lấy thông tin sản phẩm (nếu cần kiểm tra tồn kho)
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

		itemTotal := float64(item.Quantity) * item.Price
		totalAmount += itemTotal

		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
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
		OrderCode:     orderCode,
		Status:          "pending",
		PaymentStatus:   "pending",
		PaymentMethod:   input.PaymentMethod,
		TotalAmount:     totalAmount,
		ShippingAmount:  shippingAmount,
		FinalAmount:     finalAmount,
		DiscountCode:    input.DiscountCode,
		Address:         input.Address,
		Name:    input.Name,
		Phone:   input.Phone,
		Email:   input.Email,
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
	
	// Lưu địa chỉ vào hệ thống nếu số điện thoại chưa tồn tại
	addressRepo := repo.NewAddressRepo()
	addressRepo.SaveAddressFromOrder(userIDPtr, input.Name, input.Phone, input.Address)

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
		Data:    order.ToDetailResponse(),
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

func (h *OrderHandler) generateUniqueOrderCode() string {
	for {
		timestamp := time.Now().Format("20060102150405")
		randomNum := rand.Intn(9999)
		orderCode := fmt.Sprintf("ORD-%s-%04d", timestamp, randomNum)
		
		exists, _ := h.orderRepo.CheckOrderCodeExists(orderCode)
		if (!exists) {
			return orderCode
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
	orderCode := c.Param("order_code")
	if orderCode == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Số đơn hàng là bắt buộc", nil)
		return
	}

	order, err := h.orderRepo.GetByOrderCode(orderCode)
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
		Data:    order.ToDetailResponse(),
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

// AdminCreateOrder tạo đơn hàng mới bởi admin
func (h *OrderHandler) AdminCreateOrder(c *gin.Context) {
	var input model.AdminOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Tạo số đơn hàng duy nhất theo định dạng WebShop-DDMMYYNNN
	orderCode, err := h.orderRepo.GenerateOrderCodeForDate(time.Now())
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo mã đơn hàng", err)
		return
	}

	// Tính tổng tiền từ danh sách sản phẩm
	var totalAmount float64 = 0
	var orderItems []model.OrderItem

	for _, item := range input.Products {
		// Lấy thông tin sản phẩm nếu cần kiểm tra tồn kho

		product, err := h.productRepo.GetByID(item.ProductID)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Không tìm thấy sản phẩm với ID: %s", item.ProductID), err)
			return
		}
		// Kiểm tra tồn kho
		if product.Stock < item.Quantity {
			helpers.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Không đủ hàng tồn kho cho sản phẩm %s", product.Name), nil)
			return
		}

		// Có thể kiểm tra tồn kho nếu muốn
		itemTotal := float64(item.Quantity) * item.Price
		totalAmount += itemTotal
		orderItems = append(orderItems, model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Total:     itemTotal,
		})
	}

	finalAmount := totalAmount + input.ShippingFee

	// Xác định payment status dựa trên payment method
	paymentStatus := "pending"
	if input.PaymentMethod == "unpaid" {
		paymentStatus = "unpaid"
	}

	// Tạo đơn hàng
	order := model.Order{
		UserID:          nil, // Admin tạo đơn không gắn với user cụ thể
		CreatorID:       &input.CreatorID,
		CreatorName:     input.CreatorName,
		OrderCode:     orderCode,
		Status:          func() string { if input.Status == "new" { return "pending" } else { return input.Status } }(),
		PaymentStatus:   paymentStatus,
		PaymentMethod:   input.PaymentMethod,
		TotalAmount:     totalAmount,
		ShippingAmount:  input.ShippingFee,
		FinalAmount:     finalAmount,
		DiscountCode:    "",
		Address:         input.Address,
		Name:    input.Name,
		Phone:   input.Phone,
		Email:   input.Email,
		Notes:           input.Note,
		IsGuestOrder:    true, // Admin tạo đơn được coi là guest order
		OrderType:       input.OrderType,
		OrderItems:      orderItems,
	}

	if input.DiscountCode != nil {
		order.DiscountCode = *input.DiscountCode
	}

	if err := h.orderRepo.Create(&order); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo đơn hàng", err)
		return
	}

	// Nếu có userID và có mảng addresses từ FE thì lưu toàn bộ địa chỉ
	if order.UserID != nil && len(input.Addresses) > 0 {
		var addresses []model.Address
		for i, addr := range input.Addresses {
			addresses = append(addresses, model.Address{
				UserID:       *order.UserID,
				Name:         order.Name,
				Phone:        order.Phone,
				AddressLine1: addr,
				AddressLine2: "",
				City:         "N/A",
				State:        "N/A",
				PostalCode:   "000000",
				Country:      "Vietnam",
				IsDefault:    i == 0,
			})
		}
		addressRepo := repo.NewAddressRepo()
		addressRepo.BulkCreateAddresses(*order.UserID, addresses)
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

// AdminGetOrders lấy danh sách đơn hàng với bộ lọc cho admin
func (h *OrderHandler) AdminGetOrders(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	status := c.Query("status")
	paymentStatus := c.Query("payment_status")
	orderType := c.Query("order_type")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	orders, total, err := h.orderRepo.GetAllWithFilters(page, limit, status, paymentStatus, orderType)
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
			"filters": map[string]interface{}{
				"status":         status,
				"payment_status": paymentStatus,
				"order_type":     orderType,
			},
		},
	})
}

// AdminUpdateOrder cập nhật đơn hàng bởi admin
func (h *OrderHandler) AdminUpdateOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đơn hàng không hợp lệ", err)
		return
	}

	var input model.AdminOrderUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra đơn hàng có tồn tại không
	_, err = h.orderRepo.GetByID(id)
	if err != nil {
		if err.Error() == "order not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy đơn hàng", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	// Tạo map updates chỉ với các trường được cung cấp
	updates := make(map[string]interface{})

	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Phone != nil {
		updates["phone"] = *input.Phone
	}
	if input.Email != nil {
		updates["email"] = *input.Email
	}
	if input.Address != nil {
		updates["shipping_address"] = *input.Address
	}
	if input.Note != nil {
		updates["notes"] = *input.Note
	}
	if input.DiscountCode != nil {
		updates["discount_code"] = *input.DiscountCode
	}
	if input.ShippingFee != nil {
		updates["shipping_fee"] = *input.ShippingFee
		// Cập nhật lại final amount nếu có thay đổi shipping fee
		order, _ := h.orderRepo.GetByID(id)
		updates["final_amount"] = order.TotalAmount + *input.ShippingFee
	}
	if input.Status != nil {
		updates["status"] = *input.Status
		// Ghi nhận thời gian cho một số trạng thái cụ thể
		now := time.Now()
		switch *input.Status {
		case "shipped":
			updates["shipped_at"] = &now
		case "delivered":
			updates["delivered_at"] = &now
		case "cancelled":
			updates["cancelled_at"] = &now
		}
	}
	if input.PaymentMethod != nil {
		updates["payment_method"] = *input.PaymentMethod
		// Tự động cập nhật payment status
		if *input.PaymentMethod == "unpaid" {
			updates["payment_status"] = "unpaid"
		} else {
			updates["payment_status"] = "pending"
		}
	}

	if input.Products != nil && len(input.Products) > 0 {
		// Xóa toàn bộ order_items cũ
		h.orderRepo.DeleteOrderItems(id)
		// Thêm lại order_items mới
		var orderItems []model.OrderItem
		for _, item := range input.Products {
			orderItems = append(orderItems, model.OrderItem{
				OrderID:   id,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     item.Price,
				Total:     float64(item.Quantity) * item.Price,
			})
		}
		h.orderRepo.BulkInsertOrderItems(orderItems)
	}

	if len(updates) == 0 {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Không có dữ liệu để cập nhật", nil)
		return
	}

	if err := h.orderRepo.AdminUpdate(id, updates); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật đơn hàng", err)
		return
	}

	// Lấy đơn hàng đã cập nhật
	updatedOrder, err := h.orderRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy đơn hàng đã cập nhật", err)
		return
	}

	// Nếu có địa chỉ mới từ FE và đơn hàng liên kết với user
	if updatedOrder.UserID != nil && len(input.Addresses) > 0 {
		var addresses []model.Address
		for i, addr := range input.Addresses {
			addresses = append(addresses, model.Address{
				UserID:       *updatedOrder.UserID,
				Name:         updatedOrder.Name,
				Phone:        updatedOrder.Phone,
				AddressLine1: addr,
				AddressLine2: "",
				City:         "N/A",
				State:        "N/A",
				PostalCode:   "000000",
				Country:      "Vietnam",
				IsDefault:    i == 0,
			})
		}
		addressRepo := repo.NewAddressRepo()
		addressRepo.BulkCreateAddresses(*updatedOrder.UserID, addresses)
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật đơn hàng thành công",
		Data:    updatedOrder.ToResponse(),
	})
}

// AdminDeleteOrder xóa đơn hàng bởi admin
func (h *OrderHandler) AdminDeleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID đơn hàng không hợp lệ", err)
		return
	}

	// Kiểm tra đơn hàng có tồn tại không
	order, err := h.orderRepo.GetByID(id)
	if err != nil {
		if err.Error() == "order not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy đơn hàng", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	// Kiểm tra trạng thái đơn hàng có cho phép xóa không
	if order.Status == "shipped" || order.Status == "delivered" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Không thể xóa đơn hàng đã giao hoặc đang giao", nil)
		return
	}

	if err := h.orderRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa đơn hàng", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Xóa đơn hàng thành công",
		Data:    map[string]interface{}{"deleted_order_id": id},
	})
}