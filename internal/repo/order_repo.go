package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepo struct {
	db *gorm.DB
	addressRepo *AddressRepo
}

func NewOrderRepo() *OrderRepo {
	return &OrderRepo{
		db: app.DB,
		addressRepo: NewAddressRepo(),
	}
}

// Create tạo mới một đơn hàng với logic tự động tạo user và áp dụng mã giảm giá
func (r *OrderRepo) Create(order *model.Order) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Nếu không có UserID nhưng có thông tin , tự động tạo user mới
		if order.UserID == nil && order.Email != "" && order.Phone != "" {
			// Kiểm tra xem đã có user với email hoặc phone này chưa
			var existingUser model.User
			err := tx.Where("email = ? OR phone = ?", order.Email, order.Phone).First(&existingUser).Error
			
			if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
				// Tạo user mới
				newUser := model.User{
					ID:       uuid.New(),
					Username: order.Email, // Sử dụng email làm username
					FullName: order.Name,
					Email:    order.Email,
					Phone:    order.Phone,
					Password: "auto_generated", // Có thể hash password mặc định hoặc để trống
					Role:     "user",
					IsActive: true,
				}
				
				if err := tx.Create(&newUser).Error; err != nil {
					return err
				}
				
				// Gán UserID cho order
				order.UserID = &newUser.ID
				order.IsGuestOrder = false
			} else if err == nil {
				// User đã tồn tại, gán UserID
				order.UserID = &existingUser.ID
				order.IsGuestOrder = false
			} else {
				return err
			}
		}
		
		// Xử lý mã giảm giá nếu có
		if order.DiscountCode != "" && order.UserID != nil {
			discountRepo := NewDiscountRepo(tx)
			
			// Lấy thông tin mã giảm giá
			discount, err := discountRepo.GetByCode(order.DiscountCode)
			if err != nil {
				return errors.New("invalid discount code")
			}
			
			// Kiểm tra mã giảm giá có hợp lệ không
			if !discount.CanApply(order.TotalAmount) {
				return errors.New("discount cannot be applied to this order")
			}
			
			// Kiểm tra user có thể sử dụng mã này không
			if !discount.CanUserUse(*order.UserID, tx) {
				return errors.New("user has exceeded usage limit for this discount")
			}
			
			// Kiểm tra sản phẩm có được áp dụng mã giảm giá không (nếu có items)
			if len(order.OrderItems) > 0 {
				hasApplicableProduct := false
				for _, item := range order.OrderItems {
					if discount.CheckProductApplicable(item.ProductID, tx) {
						hasApplicableProduct = true
						break
					}
				}
				if !hasApplicableProduct {
					return errors.New("discount code cannot be applied to any products in this order")
				}
			}
			
			// Tính toán số tiền giảm giá
			discountAmount := discount.CalculateDiscount(order.TotalAmount)
			order.DiscountAmount = discountAmount
			order.FinalAmount = order.TotalAmount - discountAmount + order.ShippingAmount
		}
		
		// Tạo order
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		
		// Ghi lại việc sử dụng mã giảm giá nếu có
		if order.DiscountCode != "" && order.UserID != nil {
			discountRepo := NewDiscountRepo(tx)
			discount, err := discountRepo.GetByCode(order.DiscountCode)
			if err == nil {
				// Ghi lại việc sử dụng
				if err := discountRepo.RecordUserUsage(*order.UserID, discount.ID, order.ID); err != nil {
					return err
				}
				
				// Tăng used_count
				if err := discountRepo.IncrementUsedCount(discount.ID); err != nil {
					return err
				}
			}
		}
		
		// Cập nhật thống kê user nếu có UserID
		if order.UserID != nil {
			if err := r.updateUserOrderStats(tx, *order.UserID, order.FinalAmount, 1); err != nil {
				return err
			}
		}
		
		return nil
	})
}

// GetByID lấy đơn hàng theo ID kèm dữ liệu liên quan
func (r *OrderRepo) GetByID(id uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("User.Addresses").Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

// GetByOrderCode lấy đơn hàng theo mã đơn
func (r *OrderRepo) GetByOrderCode(orderCode string) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("User.Addresses").Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("order_code = ?", orderCode).
		First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

// GetByUserID lấy danh sách đơn hàng theo userID có phân trang
func (r *OrderRepo) GetByUserID(userID uuid.UUID, page, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Order{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy đơn hàng kèm dữ liệu liên quan
	err := r.db.Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// GetAll lấy tất cả đơn hàng có phân trang
func (r *OrderRepo) GetAll(page, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	// Đếm tổng số bản ghi
	if err := r.db.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy đơn hàng kèm dữ liệu liên quan
	err := r.db.Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// Update cập nhật đơn hàng
func (r *OrderRepo) Update(order *model.Order) error {
	return r.db.Save(order).Error
}

// UpdateStatus cập nhật trạng thái đơn hàng
func (r *OrderRepo) UpdateStatus(id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Ghi nhận thời gian cho một số trạng thái cụ thể
	now := time.Now()
	switch status {
	case "shipped":
		updates["shipped_at"] = &now
	case "delivered":
		updates["delivered_at"] = &now
	case "cancelled":
		updates["cancelled_at"] = &now
	}

	return r.db.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error
}

// UpdatePaymentStatus cập nhật trạng thái thanh toán
func (r *OrderRepo) UpdatePaymentStatus(id uuid.UUID, paymentStatus string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("payment_status", paymentStatus).Error
}

// GenerateOrderCodeForDate tạo mã đơn hàng theo định dạng WebShop-DDMMYYNNN
func (r *OrderRepo) GenerateOrderCodeForDate(t time.Time) (string, error) {
	datePart := t.Format("020106") // ddmmyy
	prefix := fmt.Sprintf("WebShop-%s", datePart)

	var lastOrder model.Order
	err := r.db.Where("order_code LIKE ?", prefix+"%").Order("order_code DESC").Limit(1).First(&lastOrder).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prefix + "001", nil
		}
		return "", err
	}

	// Lấy phần số cuối của mã đơn
	suffix := lastOrder.OrderCode[len(prefix):]
	var lastSeq int
	if suffix != "" {
		fmt.Sscanf(suffix, "%d", &lastSeq)
	}

	nextSeq := lastSeq + 1
	var seqStr string
	if nextSeq <= 999 {
		seqStr = fmt.Sprintf("%03d", nextSeq)
	} else {
		seqStr = fmt.Sprintf("%d", nextSeq)
	}

	return prefix + seqStr, nil
}

// CheckOrderCodeExists kiểm tra mã đơn đã tồn tại chưa
func (r *OrderRepo) CheckOrderCodeExists(orderCode string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Order{}).Where("order_code = ?", orderCode).Count(&count).Error
	return count > 0, err
}

// GetOrderStats lấy thống kê đơn hàng
func (r *OrderRepo) GetOrderStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Tổng số đơn hàng
	var totalOrders int64
	if err := r.db.Model(&model.Order{}).Count(&totalOrders).Error; err != nil {
		return nil, err
	}
	stats["total_orders"] = totalOrders

	// Số lượng đơn theo trạng thái
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := r.db.Model(&model.Order{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, err
	}
	stats["orders_by_status"] = statusCounts

	// Tổng doanh thu
	var totalRevenue float64
	if err := r.db.Model(&model.Order{}).
		Where("payment_status = ?", "paid").
		Select("SUM(final_amount)").
		Scan(&totalRevenue).Error; err != nil {
		return nil, err
	}
	stats["total_revenue"] = totalRevenue

	return stats, nil
}

// GetByEmailOrPhone lấy đơn hàng theo email hoặc số điện thoại khách
func (r *OrderRepo) GetByEmailOrPhone(emailOrPhone string, page, limit int) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := r.db.Model(&model.Order{}).Where("email = ? OR phone = ?", emailOrPhone, emailOrPhone)

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Lấy đơn hàng kèm dữ liệu liên quan
	err := r.db.Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where("email = ? OR phone = ?", emailOrPhone, emailOrPhone).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// GetGuestOrderStats lấy thống kê đơn của khách vãng lai
func (r *OrderRepo) GetGuestOrderStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Tổng số đơn của khách vãng lai
	var totalGuestOrders int64
	if err := r.db.Model(&model.Order{}).Where("is_guest_order = ?", true).Count(&totalGuestOrders).Error; err != nil {
		return nil, err
	}
	stats["total_guest_orders"] = totalGuestOrders

	// Tổng số đơn của người dùng đã đăng ký
	var totalUserOrders int64
	if err := r.db.Model(&model.Order{}).Where("is_guest_order = ?", false).Count(&totalUserOrders).Error; err != nil {
		return nil, err
	}
	stats["total_user_orders"] = totalUserOrders

	// Doanh thu từ đơn của khách vãng lai
	var guestRevenue float64
	if err := r.db.Model(&model.Order{}).
		Where("is_guest_order = ? AND payment_status = ?", true, "paid").
		Select("SUM(final_amount)").
		Scan(&guestRevenue).Error; err != nil {
		return nil, err
	}
	stats["guest_revenue"] = guestRevenue

	return stats, nil
}

// AdminUpdate cập nhật đơn hàng bởi admin
func (r *OrderRepo) AdminUpdate(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Lấy thông tin đơn hàng hiện tại
		var order model.Order
		if err := tx.Where("id = ?", id).First(&order).Error; err != nil {
			return err
		}
		
		// Cập nhật đơn hàng
		if err := tx.Model(&model.Order{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		
		// Lưu địa chỉ nếu có thay đổi thông tin địa chỉ và có UserID
		if order.UserID != nil {
			name := order.Name
			phone := order.Phone
			address := order.Address
			
			// Cập nhật từ updates nếu có
			if newName, exists := updates["name"]; exists {
				if nameStr, ok := newName.(string); ok {
					name = nameStr
				}
			}
			if newPhone, exists := updates["phone"]; exists {
				if phoneStr, ok := newPhone.(string); ok {
					phone = phoneStr
				}
			}
			if newAddress, exists := updates["shipping_address"]; exists {
				if addressStr, ok := newAddress.(string); ok {
					address = addressStr
				}
			}
			
			// Lưu địa chỉ mới nếu có thay đổi
			if order.UserID != nil {
				r.addressRepo.SaveAddressFromOrder(order.UserID, name, phone, address)
			} else {
				r.addressRepo.SaveAddressFromOrder(nil, name, phone, address)
			}
		}
		
		return nil
	})
}

// Delete xóa đơn hàng (soft delete)
func (r *OrderRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Order{}, id).Error
}

// GetAllWithFilters lấy đơn hàng với bộ lọc cho admin
func (r *OrderRepo) GetAllWithFilters(page, limit int, status, paymentStatus, orderType string) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	query := r.db.Model(&model.Order{})

	// Áp dụng bộ lọc
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if paymentStatus != "" {
		query = query.Where("payment_status = ?", paymentStatus)
	}
	if orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}

	// Đếm tổng số bản ghi
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Tính offset
	offset := (page - 1) * limit

	// Sử dụng lại query và thêm preload + phân trang
	err := r.db.Model(&model.Order{}).
		Preload("User").
		Preload("Creator").
		Preload("OrderItems").
		Preload("OrderItems.Product").
		Where(query.Statement.SQL.String(), query.Statement.Vars...).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error

	return orders, total, err
}

// DeleteOrderItems xóa toàn bộ order_items của một đơn hàng
func (r *OrderRepo) DeleteOrderItems(orderID uuid.UUID) error {
	return r.db.Where("order_id = ?", orderID).Delete(&model.OrderItem{}).Error
}

// BulkInsertOrderItems thêm nhiều order_items mới cho một đơn hàng
func (r *OrderRepo) BulkInsertOrderItems(items []model.OrderItem) error {
	return r.db.Create(&items).Error
}

// UpdateOrderAddresses cập nhật địa chỉ đơn hàng (method placeholder)
func (r *OrderRepo) UpdateOrderAddresses(orderID uuid.UUID, addresses []model.Address) error {
	// Có thể implement nếu cần lưu multiple addresses cho order
	return nil
}

// DeleteAllAddressesOfOrder xóa toàn bộ địa chỉ của đơn hàng (method placeholder)
func (r *OrderRepo) DeleteAllAddressesOfOrder(orderID uuid.UUID) error {
	// Có thể implement nếu cần
	return nil
}

// updateUserOrderStats cập nhật thống kê đơn hàng của user
func (r *OrderRepo) updateUserOrderStats(tx *gorm.DB, userID uuid.UUID, orderAmount float64, orderCount int) error {
	updates := map[string]interface{}{
		"total_orders": gorm.Expr("total_orders + ?", orderCount),
		"total_spent":  gorm.Expr("total_spent + ?", orderAmount),
		"last_order_at": time.Now(),
	}
	
	return tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}